package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const forumTopicName = "aegis-forum-global"

type P2PStatus struct {
	Started        bool     `json:"started"`
	PeerID         string   `json:"peerId"`
	ListenAddrs    []string `json:"listenAddrs"`
	ConnectedPeers []string `json:"connectedPeers"`
	Topic          string   `json:"topic"`
}

func (a *App) StartP2P(listenPort int, bootstrapPeers []string) (P2PStatus, error) {
	a.p2pMu.Lock()
	defer a.p2pMu.Unlock()

	if a.p2pHost != nil {
		return a.getP2PStatusLocked(), nil
	}

	if listenPort <= 0 {
		listenPort = 40100
	}

	listenAddr := fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)
	host, err := libp2p.New(libp2p.ListenAddrStrings(listenAddr))
	if err != nil {
		return P2PStatus{}, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	gossip, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		_ = host.Close()
		cancel()
		return P2PStatus{}, err
	}

	topic, err := gossip.Join(forumTopicName)
	if err != nil {
		_ = host.Close()
		cancel()
		return P2PStatus{}, err
	}

	subscription, err := topic.Subscribe()
	if err != nil {
		_ = topic.Close()
		_ = host.Close()
		cancel()
		return P2PStatus{}, err
	}

	a.p2pCtx = ctx
	a.p2pCancel = cancel
	a.p2pHost = host
	a.p2pTopic = topic
	a.p2pSub = subscription

	go a.consumeP2PMessages(ctx, host.ID(), subscription)

	for _, address := range bootstrapPeers {
		trimmed := strings.TrimSpace(address)
		if trimmed == "" {
			continue
		}
		_ = a.connectPeerLocked(trimmed)
	}

	return a.getP2PStatusLocked(), nil
}

func (a *App) StopP2P() error {
	a.p2pMu.Lock()
	defer a.p2pMu.Unlock()

	var firstErr error

	if a.p2pCancel != nil {
		a.p2pCancel()
	}
	if a.p2pSub != nil {
		a.p2pSub.Cancel()
	}
	if a.p2pTopic != nil {
		if err := a.p2pTopic.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if a.p2pHost != nil {
		if err := a.p2pHost.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	a.p2pCtx = nil
	a.p2pCancel = nil
	a.p2pSub = nil
	a.p2pTopic = nil
	a.p2pHost = nil

	return firstErr
}

func (a *App) ConnectPeer(address string) error {
	a.p2pMu.Lock()
	defer a.p2pMu.Unlock()

	return a.connectPeerLocked(address)
}

func (a *App) connectPeerLocked(address string) error {
	if a.p2pHost == nil {
		return errors.New("p2p not started")
	}

	maddr, err := multiaddr.NewMultiaddr(strings.TrimSpace(address))
	if err != nil {
		return err
	}

	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(a.p2pCtx, 5*time.Second)
	defer cancel()

	return a.p2pHost.Connect(ctx, *info)
}

func (a *App) PublishPost(pubkey string, content string) error {
	if strings.TrimSpace(pubkey) == "" {
		return errors.New("pubkey is required")
	}
	if strings.TrimSpace(content) == "" {
		return errors.New("content is required")
	}

	a.p2pMu.Lock()
	topic := a.p2pTopic
	ctx := a.p2pCtx
	a.p2pMu.Unlock()

	if topic == nil || ctx == nil {
		return errors.New("p2p not started")
	}

	now := time.Now().Unix()
	msg := IncomingMessage{
		Type:      "POST",
		ID:        buildMessageID(pubkey, content, now),
		Pubkey:    pubkey,
		Content:   content,
		Timestamp: now,
		Signature: "",
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	if err = a.ProcessIncomingMessage(payload); err != nil {
		return err
	}

	return topic.Publish(ctx, payload)
}

func (a *App) PublishShadowBan(targetPubkey string, adminPubkey string, reason string) error {
	return a.publishGovernanceMessage("SHADOW_BAN", targetPubkey, adminPubkey, reason)
}

func (a *App) PublishUnban(targetPubkey string, adminPubkey string, reason string) error {
	return a.publishGovernanceMessage("UNBAN", targetPubkey, adminPubkey, reason)
}

func (a *App) publishGovernanceMessage(action string, targetPubkey string, adminPubkey string, reason string) error {
	action = strings.ToUpper(strings.TrimSpace(action))
	if action != "SHADOW_BAN" && action != "UNBAN" {
		return errors.New("invalid governance action")
	}

	if strings.TrimSpace(targetPubkey) == "" {
		return errors.New("target pubkey is required")
	}
	if strings.TrimSpace(adminPubkey) == "" {
		return errors.New("admin pubkey is required")
	}

	now := time.Now().Unix()
	msg := IncomingMessage{
		Type:         action,
		TargetPubkey: strings.TrimSpace(targetPubkey),
		AdminPubkey:  strings.TrimSpace(adminPubkey),
		Timestamp:    now,
		Reason:       strings.TrimSpace(reason),
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	if err = a.ProcessIncomingMessage(payload); err != nil {
		return err
	}

	a.p2pMu.Lock()
	topic := a.p2pTopic
	ctx := a.p2pCtx
	a.p2pMu.Unlock()

	if topic == nil || ctx == nil {
		return nil
	}

	return topic.Publish(ctx, payload)
}

func (a *App) GetP2PStatus() P2PStatus {
	a.p2pMu.Lock()
	defer a.p2pMu.Unlock()
	return a.getP2PStatusLocked()
}

func (a *App) getP2PStatusLocked() P2PStatus {
	status := P2PStatus{
		Started: false,
		Topic:   forumTopicName,
	}

	if a.p2pHost == nil {
		return status
	}

	status.Started = true
	status.PeerID = a.p2pHost.ID().String()
	status.ListenAddrs = make([]string, 0, len(a.p2pHost.Addrs()))
	for _, addr := range a.p2pHost.Addrs() {
		status.ListenAddrs = append(status.ListenAddrs, fmt.Sprintf("%s/p2p/%s", addr.String(), a.p2pHost.ID().String()))
	}

	connected := a.p2pHost.Network().Peers()
	status.ConnectedPeers = make([]string, 0, len(connected))
	for _, connectedPeer := range connected {
		status.ConnectedPeers = append(status.ConnectedPeers, connectedPeer.String())
	}

	return status
}

func (a *App) consumeP2PMessages(ctx context.Context, localPeerID peer.ID, sub *pubsub.Subscription) {
	for {
		message, err := sub.Next(ctx)
		if err != nil {
			return
		}

		if message.ReceivedFrom == localPeerID {
			continue
		}

		if err = a.ProcessIncomingMessage(message.Data); err != nil {
			if a.ctx != nil {
				runtime.LogErrorf(a.ctx, "process p2p message failed: %v", err)
			}
			continue
		}

		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "feed:updated")
		}
	}
}
