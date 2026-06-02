package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"time"
)

func main() {
	host := os.Getenv("UNIFI_HOST")
	username := os.Getenv("UNIFI_USERNAME")
	password := os.Getenv("UNIFI_PASSWORD")
	backupDir := os.Getenv("BACKUP_DIR")

	if host == "" || username == "" || password == "" || backupDir == "" {
		fmt.Fprintln(os.Stderr, "required env vars: UNIFI_HOST, UNIFI_USERNAME, UNIFI_PASSWORD, BACKUP_DIR")
		os.Exit(1)
	}

	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:     jar,
		Timeout: 60 * time.Second,
	}

	creds, _ := json.Marshal(map[string]string{"username": username, "password": password})
	resp, err := client.Post(host+"/api/auth/login", "application/json", bytes.NewReader(creds))
	if err != nil {
		fmt.Fprintf(os.Stderr, "login request failed: %v\n", err)
		os.Exit(1)
	}
	io.Copy(io.Discard, resp.Body) //nolint:errcheck
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "login failed: HTTP %d\n", resp.StatusCode)
		os.Exit(1)
	}
	fmt.Println("logged in successfully")

	resp, err = client.Get(host + "/api/backup/download")
	if err != nil {
		fmt.Fprintf(os.Stderr, "backup request failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "backup download failed: HTTP %d\n", resp.StatusCode)
		os.Exit(1)
	}

	if err := os.MkdirAll(backupDir, 0750); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create backup directory: %v\n", err)
		os.Exit(1)
	}

	filename := fmt.Sprintf("unifi-backup-%s.unf", time.Now().UTC().Format("2006-01-02T150405Z"))
	outPath := filepath.Join(backupDir, filename)

	f, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create output file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	n, err := io.Copy(f, resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to write backup: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("saved %d bytes to %s\n", n, outPath)
}