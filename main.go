package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

type endpoint struct {
	host     string
	username string
	password string
}

type healthState struct {
	Status      string    `json:"status"`
	LastRun     time.Time `json:"last_run,omitempty"`
	LastSuccess time.Time `json:"last_success,omitempty"`
	Errors      []string  `json:"errors,omitempty"`
}

func main() {
	backupDir := os.Getenv("BACKUP_DIR")
	if backupDir == "" {
		fmt.Fprintln(os.Stderr, "required env var: BACKUP_DIR")
		os.Exit(1)
	}

	endpoints := parseEndpoints()
	if len(endpoints) == 0 {
		fmt.Fprintln(os.Stderr, "no endpoints configured: set UNIFI_HOSTS/UNIFI_USERNAMES/UNIFI_PASSWORDS or UNIFI_HOST/UNIFI_USERNAME/UNIFI_PASSWORD")
		os.Exit(1)
	}

	intervalStr := os.Getenv("BACKUP_INTERVAL")
	if intervalStr == "" {
		if errs := runAll(endpoints, backupDir); len(errs) > 0 {
			os.Exit(1)
		}
		return
	}

	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid BACKUP_INTERVAL %q: %v\n", intervalStr, err)
		os.Exit(1)
	}

	var state atomic.Pointer[healthState]

	healthPort := os.Getenv("HEALTH_PORT")
	if healthPort == "" {
		healthPort = "8080"
	}
	go serveHealth(healthPort, &state)

	update := func() {
		errs := runAll(endpoints, backupDir)
		now := time.Now()
		hs := &healthState{LastRun: now}
		if len(errs) == 0 {
			hs.Status = "ok"
			hs.LastSuccess = now
		} else {
			hs.Status = "degraded"
			for _, e := range errs {
				hs.Errors = append(hs.Errors, e.Error())
			}
		}
		state.Store(hs)
	}

	update()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		update()
	}
}

func runAll(endpoints []endpoint, backupDir string) []error {
	var errs []error
	for _, ep := range endpoints {
		if err := backup(ep, backupDir); err != nil {
			fmt.Fprintf(os.Stderr, "[%s] backup failed: %v\n", ep.host, err)
			errs = append(errs, fmt.Errorf("%s: %w", ep.host, err))
		}
	}
	return errs
}

func backup(ep endpoint, backupDir string) error {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:     jar,
		Timeout: 60 * time.Second,
	}

	creds, _ := json.Marshal(map[string]string{"username": ep.username, "password": ep.password})
	resp, err := client.Post(ep.host+"/api/auth/login", "application/json", bytes.NewReader(creds))
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	io.Copy(io.Discard, resp.Body) //nolint:errcheck
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed: HTTP %d", resp.StatusCode)
	}

	resp, err = client.Get(ep.host + "/api/backup/download")
	if err != nil {
		return fmt.Errorf("backup request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("backup download failed: HTTP %d", resp.StatusCode)
	}

	if err := os.MkdirAll(backupDir, 0750); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	filename := fmt.Sprintf("unifi-backup-%s-%s.unf", hostLabel(ep.host), time.Now().UTC().Format("2006-01-02T150405Z"))
	outPath := filepath.Join(backupDir, filename)

	f, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()

	n, err := io.Copy(f, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write backup: %w", err)
	}

	fmt.Printf("[%s] saved %d bytes to %s\n", ep.host, n, outPath)
	return nil
}

func hostLabel(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Hostname() == "" {
		return strings.NewReplacer("://", "-", ".", "-", ":", "-", "/", "-").Replace(rawURL)
	}
	label := strings.ReplaceAll(u.Hostname(), ".", "-")
	if u.Port() != "" {
		label += "-" + u.Port()
	}
	return label
}

func parseEndpoints() []endpoint {
	hosts := splitCSV(os.Getenv("UNIFI_HOSTS"))
	usernames := splitCSV(os.Getenv("UNIFI_USERNAMES"))
	passwords := splitCSV(os.Getenv("UNIFI_PASSWORDS"))

	if len(hosts) == 0 {
		h, u, p := os.Getenv("UNIFI_HOST"), os.Getenv("UNIFI_USERNAME"), os.Getenv("UNIFI_PASSWORD")
		if h != "" && u != "" && p != "" {
			return []endpoint{{host: h, username: u, password: p}}
		}
		return nil
	}

	if len(usernames) != len(hosts) || len(passwords) != len(hosts) {
		fmt.Fprintln(os.Stderr, "UNIFI_HOSTS, UNIFI_USERNAMES, and UNIFI_PASSWORDS must have the same number of entries")
		os.Exit(1)
	}

	eps := make([]endpoint, len(hosts))
	for i := range hosts {
		eps[i] = endpoint{host: hosts[i], username: usernames[i], password: passwords[i]}
	}
	return eps
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}

func serveHealth(port string, state *atomic.Pointer[healthState]) {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		hs := state.Load()
		if hs == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "starting"}) //nolint:errcheck
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(hs) //nolint:errcheck
	})
	if err := (&http.Server{Addr: ":" + port, Handler: mux}).ListenAndServe(); err != nil {
		fmt.Fprintf(os.Stderr, "health server: %v\n", err)
	}
}
