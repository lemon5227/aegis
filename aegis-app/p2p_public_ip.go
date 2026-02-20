package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

func resolveAutoAnnounceEnabled() bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv("AEGIS_AUTO_ANNOUNCE")))
	if raw == "" {
		return true
	}
	return !(raw == "0" || raw == "false" || raw == "no" || raw == "off")
}

func resolveAutoPublicIPv4() string {
	if value := strings.TrimSpace(fetchAzurePublicIP()); isPublicIPv4(value) {
		return value
	}
	if value := strings.TrimSpace(fetchAWSPublicIP()); isPublicIPv4(value) {
		return value
	}
	if value := strings.TrimSpace(fetchGCPPublicIP()); isPublicIPv4(value) {
		return value
	}

	for _, endpoint := range []string{
		"https://ifconfig.me",
		"https://api.ipify.org",
		"https://ipinfo.io/ip",
	} {
		if value := strings.TrimSpace(fetchExternalPlainText(endpoint, 1800*time.Millisecond)); isPublicIPv4(value) {
			return value
		}
	}

	return ""
}

func fetchAWSPublicIP() string {
	url := "http://169.254.169.254/latest/meta-data/public-ipv4"
	return fetchExternalPlainText(url, 900*time.Millisecond)
}

func fetchAzurePublicIP() string {
	ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://169.254.169.254/metadata/instance/network/interface/0/ipv4/ipAddress/0/publicIpAddress?api-version=2021-02-01&format=text", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Metadata", "true")

	client := &http.Client{Timeout: 1200 * time.Millisecond}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ""
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 128))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}

func fetchGCPPublicIP() string {
	ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://metadata.google.internal/computeMetadata/v1/instance/network-interfaces/0/access-configs/0/external-ip", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Metadata-Flavor", "Google")

	client := &http.Client{Timeout: 1200 * time.Millisecond}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ""
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 128))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}

func fetchExternalPlainText(url string, timeout time.Duration) string {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "aegis-auto-announce")

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ""
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 128))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}

func isPublicIPv4(value string) bool {
	ip := net.ParseIP(strings.TrimSpace(value))
	if ip == nil {
		return false
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	if ip4[0] == 10 {
		return false
	}
	if ip4[0] == 127 {
		return false
	}
	if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 {
		return false
	}
	if ip4[0] == 192 && ip4[1] == 168 {
		return false
	}
	if ip4[0] == 169 && ip4[1] == 254 {
		return false
	}
	return true
}
