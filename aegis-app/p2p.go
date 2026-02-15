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
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/multiformats/go-multiaddr"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const forumTopicName = "aegis-forum-global"
const mdnsServiceTag = "aegis-forum-mdns"

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

	mdnsService := mdns.NewMdnsService(host, mdnsServiceTag, &mdnsNotifee{app: a})
	if err = mdnsService.Start(); err == nil {
		a.mdnsSvc = mdnsService
	} else if a.ctx != nil {
		runtime.LogWarningf(a.ctx, "mdns start failed: %v", err)
	}

	go a.consumeP2PMessages(ctx, host.ID(), subscription)

	for _, address := range bootstrapPeers {
		trimmed := strings.TrimSpace(address)
		if trimmed == "" {
			continue
		}
		_ = a.connectPeerLocked(trimmed)
	}

	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "p2p:updated")
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
		if closeErr := a.p2pTopic.Close(); closeErr != nil {
			firstErr = errors.Join(firstErr, closeErr)
		}
	}
	if a.mdnsSvc != nil {
		if closeErr := a.mdnsSvc.Close(); closeErr != nil {
			firstErr = errors.Join(firstErr, closeErr)
		}
	}
	if a.p2pHost != nil {
		if closeErr := a.p2pHost.Close(); closeErr != nil {
			firstErr = errors.Join(firstErr, closeErr)
		}
	}

	a.p2pCtx = nil
	a.p2pCancel = nil
	a.p2pSub = nil
	a.p2pTopic = nil
	a.p2pHost = nil
	a.mdnsSvc = nil

	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "p2p:updated")
	}

	return firstErr
}

func (a *App) ConnectPeer(address string) error {
	a.p2pMu.Lock()
	defer a.p2pMu.Unlock()

	err := a.connectPeerLocked(address)
	if err == nil && a.ctx != nil {
		runtime.EventsEmit(a.ctx, "p2p:updated")
	}

	return err
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

func (a *App) PublishPostStructured(pubkey string, title string, body string) error {
	return a.PublishPostStructuredToSub(pubkey, title, body, defaultSubID)
}

func (a *App) PublishPostStructuredToSub(pubkey string, title string, body string, subID string) error {
	pubkey = strings.TrimSpace(pubkey)
	if pubkey == "" {
		return errors.New("pubkey is required")
	}
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	if title == "" {
		title = deriveTitle(body)
	}
	if title == "" || body == "" {
		return errors.New("title and body are required")
	}

	shadowBanned, err := a.isShadowBanned(pubkey)
	if err != nil {
		return err
	}
	if shadowBanned {
		_, err = a.AddLocalPostStructuredToSub(pubkey, title, body, "public", subID)
		return err
	}

	a.p2pMu.Lock()
	topic := a.p2pTopic
	ctx := a.p2pCtx
	a.p2pMu.Unlock()

	if topic == nil || ctx == nil {
		return errors.New("p2p not started")
	}

	now := time.Now().Unix()
	profile, profileErr := a.GetProfile(pubkey)
	if profileErr != nil {
		profile = Profile{}
	}

	msg := IncomingMessage{
		Type:        "POST",
		ID:          buildMessageID(pubkey, body, now),
		Pubkey:      pubkey,
		DisplayName: strings.TrimSpace(profile.DisplayName),
		AvatarURL:   strings.TrimSpace(profile.AvatarURL),
		Title:       title,
		Body:        body,
		ContentCID:  buildContentCID(body),
		Content:     "",
		SubID:       normalizeSubID(subID),
		Timestamp:   now,
		Signature:   "",
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

func (a *App) PublishCreateSub(subID string, title string, description string) error {
	subID = normalizeSubID(subID)
	if strings.TrimSpace(subID) == "" {
		return errors.New("sub id is required")
	}

	now := time.Now().Unix()
	msg := IncomingMessage{
		Type:      "SUB_CREATE",
		SubID:     subID,
		SubTitle:  strings.TrimSpace(title),
		SubDesc:   strings.TrimSpace(description),
		Timestamp: now,
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

func (a *App) PublishComment(pubkey string, postID string, parentID string, body string) error {
	pubkey = strings.TrimSpace(pubkey)
	postID = strings.TrimSpace(postID)
	parentID = strings.TrimSpace(parentID)
	body = strings.TrimSpace(body)

	if pubkey == "" || postID == "" || body == "" {
		return errors.New("pubkey, post id and body are required")
	}

	shadowBanned, err := a.isShadowBanned(pubkey)
	if err != nil {
		return err
	}
	if shadowBanned {
		_, err = a.AddLocalComment(pubkey, postID, parentID, body)
		return err
	}

	a.p2pMu.Lock()
	topic := a.p2pTopic
	ctx := a.p2pCtx
	a.p2pMu.Unlock()

	if topic == nil || ctx == nil {
		return errors.New("p2p not started")
	}

	now := time.Now().Unix()
	raw := fmt.Sprintf("%s|%s|%s", postID, parentID, body)
	msg := IncomingMessage{
		Type:      "COMMENT",
		ID:        buildMessageID(pubkey, raw, now),
		Pubkey:    pubkey,
		PostID:    postID,
		ParentID:  parentID,
		Body:      body,
		Timestamp: now,
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

func (a *App) PublishPostUpvote(pubkey string, postID string) error {
	pubkey = strings.TrimSpace(pubkey)
	postID = strings.TrimSpace(postID)
	if pubkey == "" || postID == "" {
		return errors.New("pubkey and post id are required")
	}

	a.p2pMu.Lock()
	topic := a.p2pTopic
	ctx := a.p2pCtx
	a.p2pMu.Unlock()

	if topic == nil || ctx == nil {
		return errors.New("p2p not started")
	}

	msg := IncomingMessage{
		Type:        "POST_UPVOTE",
		Pubkey:      pubkey,
		VoterPubkey: pubkey,
		PostID:      postID,
		Timestamp:   time.Now().Unix(),
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

func (a *App) PublishCommentUpvote(pubkey string, postID string, commentID string) error {
	pubkey = strings.TrimSpace(pubkey)
	postID = strings.TrimSpace(postID)
	commentID = strings.TrimSpace(commentID)
	if pubkey == "" || postID == "" || commentID == "" {
		return errors.New("pubkey, post id and comment id are required")
	}

	a.p2pMu.Lock()
	topic := a.p2pTopic
	ctx := a.p2pCtx
	a.p2pMu.Unlock()

	if topic == nil || ctx == nil {
		return errors.New("p2p not started")
	}

	msg := IncomingMessage{
		Type:        "COMMENT_UPVOTE",
		Pubkey:      pubkey,
		VoterPubkey: pubkey,
		PostID:      postID,
		CommentID:   commentID,
		Timestamp:   time.Now().Unix(),
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

func (a *App) PublishProfileUpdate(pubkey string, displayName string, avatarURL string) error {
	pubkey = strings.TrimSpace(pubkey)
	if pubkey == "" {
		return errors.New("pubkey is required")
	}

	now := time.Now().Unix()
	msg := IncomingMessage{
		Type:        "PROFILE_UPDATE",
		Pubkey:      pubkey,
		DisplayName: strings.TrimSpace(displayName),
		AvatarURL:   strings.TrimSpace(avatarURL),
		Timestamp:   now,
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

func (a *App) PublishGovernancePolicy(hideHistoryOnShadowBan bool) error {
	a.p2pMu.Lock()
	topic := a.p2pTopic
	ctx := a.p2pCtx
	a.p2pMu.Unlock()

	if topic == nil || ctx == nil {
		return errors.New("p2p not started")
	}

	now := time.Now().Unix()
	msg := IncomingMessage{
		Type:                   "GOVERNANCE_POLICY_UPDATE",
		HideHistoryOnShadowBan: hideHistoryOnShadowBan,
		Timestamp:              now,
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

	a.p2pMu.Lock()
	topic := a.p2pTopic
	ctx := a.p2pCtx
	a.p2pMu.Unlock()

	if topic == nil || ctx == nil {
		return errors.New("p2p not started")
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

		var incoming IncomingMessage
		_ = json.Unmarshal(message.Data, &incoming)
		messageType := strings.ToUpper(strings.TrimSpace(incoming.Type))

		if err = a.ProcessIncomingMessage(message.Data); err != nil {
			if a.ctx != nil {
				runtime.LogErrorf(a.ctx, "process p2p message failed: %v", err)
			}
			continue
		}

		if a.ctx != nil {
			if messageType == "COMMENT" || messageType == "COMMENT_UPVOTE" {
				postID := strings.TrimSpace(incoming.PostID)
				if postID != "" {
					runtime.EventsEmit(a.ctx, "comments:updated", map[string]string{"postId": postID})
					continue
				}
			}

			runtime.EventsEmit(a.ctx, "feed:updated")
		}
	}
}

type mdnsNotifee struct {
	app *App
}

func (n *mdnsNotifee) HandlePeerFound(info peer.AddrInfo) {
	if n == nil || n.app == nil {
		return
	}

	n.app.p2pMu.Lock()
	defer n.app.p2pMu.Unlock()

	if n.app.p2pHost == nil || n.app.p2pCtx == nil {
		return
	}

	ctx, cancel := context.WithTimeout(n.app.p2pCtx, 5*time.Second)
	defer cancel()

	if err := n.app.p2pHost.Connect(ctx, info); err != nil && n.app.ctx != nil {
		runtime.LogWarningf(n.app.ctx, "mdns connect failed: %v", err)
		return
	}

	if n.app.ctx != nil {
		runtime.LogInfof(n.app.ctx, "mdns connected peer: %s", info.ID.String())
		runtime.EventsEmit(n.app.ctx, "p2p:updated")
	}
}
