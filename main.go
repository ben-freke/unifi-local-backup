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
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"
)

type endpoint struct {
	host     string
	username string
	password string
}

func main() {
	backupDir := os.Getenv("BACKUP_DIR")
	if backupDir == "" {
		fmt.Fprintln(os.Stderr, "required env var: BACKUP_DIR")
		os.Exit(1)
	}

	endpoints := parseEndpoints()

	schedule := os.Getenv("BACKUP_SCHEDULE")
	if schedule == "" {
		if errs := runAll(endpoints, backupDir); len(errs) > 0 {
			os.Exit(1)
		}
		return
	}

	c := cron.New()
	if _, err := c.AddFunc(schedule, func() { runAll(endpoints, backupDir) }); err != nil {
		fmt.Fprintf(os.Stderr, "invalid BACKUP_SCHEDULE %q: %v\n", schedule, err)
		os.Exit(1)
	}

	runAll(endpoints, backupDir)
	c.Start()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	<-sig
	c.Stop()
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
	username := os.Getenv("UNIFI_USERNAME")
	password := os.Getenv("UNIFI_PASSWORD")
	if username == "" || password == "" {
		fmt.Fprintln(os.Stderr, "required env vars: UNIFI_USERNAME, UNIFI_PASSWORD")
		os.Exit(1)
	}

	hosts := splitCSV(os.Getenv("UNIFI_HOSTS"))
	if len(hosts) == 0 {
		h := os.Getenv("UNIFI_HOST")
		if h == "" {
			fmt.Fprintln(os.Stderr, "required env var: UNIFI_HOST or UNIFI_HOSTS")
			os.Exit(1)
		}
		hosts = []string{h}
	}

	eps := make([]endpoint, len(hosts))
	for i, h := range hosts {
		eps[i] = endpoint{host: h, username: username, password: password}
	}
	return eps
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
