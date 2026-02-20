//go:build relay

package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	app := NewApp()

	if err := app.initDatabase(); err != nil {
		fmt.Printf("relay init database failed: %v\n", err)
		os.Exit(1)
	}

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

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	_ = app.StopP2P()
	if app.db != nil {
		_ = app.db.Close()
	}
}
