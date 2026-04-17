//go:build relay

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

func main() {
	app := NewApp()

	if err := app.initDatabase(); err != nil {
		fmt.Printf("relay init database failed: %v\n", err)
		os.Exit(1)
	}

	app.seedTrustedAdminsFromEnv()

	listenPort := resolveAutoStartP2PPort()
	bootstrapPeers := resolveBootstrapPeers()

	status, err := app.StartP2P(listenPort, bootstrapPeers)
	if err != nil {
		fmt.Printf("relay start p2p failed: %v\n", err)
		_ = app.db.Close()
		os.Exit(1)
	}

	fmt.Printf("relay started: peer_id=%s topic=%s\n", status.PeerID, status.Topic)
	if len(status.ListenAddrs) == 0 {
		fmt.Println("listen_addrs: none")
	} else {
		fmt.Println("listen_addrs:")
		for _, addr := range status.ListenAddrs {
			fmt.Printf("- %s\n", strings.TrimSpace(addr))
		}
	}

	if len(status.AnnounceAddrs) == 0 {
		fmt.Println("announce_addrs: none (set AEGIS_ANNOUNCE_ADDRS or AEGIS_PUBLIC_IP)")
	} else {
		fmt.Println("announce_addrs:")
		for _, addr := range status.AnnounceAddrs {
			fmt.Printf("- %s\n", strings.TrimSpace(addr))
		}
	}

	var startedAt atomic.Int64
	startedAt.Store(time.Now().Unix())

	httpPort := resolveRelayHTTPPort()
	if httpPort > 0 {
		mux := http.NewServeMux()

		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			p2pStatus := app.GetP2PStatus()
			uptime := time.Now().Unix() - startedAt.Load()
			fmt.Fprintf(w, `{"status":"ok","peer_id":"%s","uptime_seconds":%d}`, p2pStatus.PeerID, uptime)
		})

		mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			metrics := app.GetReleaseMetrics()
			data, err := json.Marshal(metrics)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.Write(data)
		})

		mux.HandleFunc("/peers", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			p2pStatus := app.GetP2PStatus()
			data, err := json.Marshal(p2pStatus)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.Write(data)
		})

		go func() {
			addr := fmt.Sprintf(":%d", httpPort)
			fmt.Printf("relay http listening on %s\n", addr)
			if err := http.ListenAndServe(addr, mux); err != nil {
				fmt.Printf("relay http server error: %v\n", err)
			}
		}()
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nrelay shutting down...")
	_ = app.StopP2P()
	if app.db != nil {
		_ = app.db.Close()
	}
	fmt.Println("relay stopped")
}

func resolveRelayHTTPPort() int {
	raw := strings.TrimSpace(os.Getenv("AEGIS_RELAY_HTTP_PORT"))
	if raw == "" {
		return 40101
	}
	port, err := strconv.Atoi(raw)
	if err != nil || port < 0 || port > 65535 {
		return 40101
	}
	return port
}
