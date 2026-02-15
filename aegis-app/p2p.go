package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/multiformats/go-multiaddr"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const forumTopicName = "aegis-forum-global"
const mdnsServiceTag = "aegis-forum-mdns"

const (
	messageTypeContentFetchRequest  = "CONTENT_FETCH_REQUEST"
	messageTypeContentFetchResponse = "CONTENT_FETCH_RESPONSE"
	messageTypeMediaFetchRequest    = "MEDIA_FETCH_REQUEST"
	messageTypeMediaFetchResponse   = "MEDIA_FETCH_RESPONSE"
	messageTypeSyncSummaryRequest   = "SYNC_SUMMARY_REQUEST"
	messageTypeSyncSummaryResponse  = "SYNC_SUMMARY_RESPONSE"
)

var (
	errContentFetchNoPeers  = errors.New("content fetch no peers")
	errContentFetchTimeout  = errors.New("content fetch timeout")
	errContentFetchNotFound = errors.New("content fetch not found")
	errMediaFetchNoPeers    = errors.New("media fetch no peers")
	errMediaFetchTimeout    = errors.New("media fetch timeout")
	errMediaFetchNotFound   = errors.New("media fetch not found")
	errAntiEntropyNoPeers   = errors.New("anti-entropy no peers")
)

type fetchRateWindow struct {
	StartedAt int64
	Count     int
}

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
	a.refreshPeerPoliciesFromEnv()

	listenAddr := fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)
	relayInfos := resolveRelayPeerInfos(bootstrapPeers)

	options := []libp2p.Option{
		libp2p.ListenAddrStrings(listenAddr),
		libp2p.NATPortMap(),
		libp2p.EnableNATService(),
		libp2p.EnableAutoNATv2(),
		libp2p.EnableHolePunching(),
		libp2p.EnableRelay(),
		libp2p.EnableRelayService(),
	}
	if len(relayInfos) > 0 {
		options = append(options, libp2p.EnableAutoRelayWithStaticRelays(relayInfos))
	}

	host, err := libp2p.New(options...)
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
	go a.runAntiEntropySyncWorker(ctx, host.ID())

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
	if blocked, reason := a.isPeerBlocked(info.ID.String()); blocked {
		return fmt.Errorf("peer blocked by policy: %s", reason)
	}

	peerCount := len(a.p2pHost.Network().Peers())
	if a.p2pHost.Network().Connectedness(info.ID) != network.Connected && peerCount >= resolveMaxConnectedPeers() {
		return fmt.Errorf("peer limit reached: %d", resolveMaxConnectedPeers())
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

func (a *App) PublishPostWithImageToSub(pubkey string, title string, body string, imageBase64 string, imageMIME string, subID string) error {
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
		_, err = a.AddLocalPostWithImageToSub(pubkey, title, body, "public", subID, imageBase64, imageMIME)
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

	localPost, err := a.AddLocalPostWithImageToSub(pubkey, title, body, "public", subID, imageBase64, imageMIME)
	if err != nil {
		return err
	}

	msg := IncomingMessage{
		Type:        "POST",
		ID:          localPost.ID,
		Pubkey:      pubkey,
		DisplayName: strings.TrimSpace(profile.DisplayName),
		AvatarURL:   strings.TrimSpace(profile.AvatarURL),
		Title:       title,
		Body:        body,
		ContentCID:  localPost.ContentCID,
		ImageCID:    localPost.ImageCID,
		ThumbCID:    localPost.ThumbCID,
		ImageMIME:   localPost.ImageMIME,
		ImageSize:   localPost.ImageSize,
		ImageWidth:  localPost.ImageWidth,
		ImageHeight: localPost.ImageHeight,
		Content:     "",
		SubID:       normalizeSubID(subID),
		Timestamp:   now,
		Signature:   "",
	}

	payload, err := json.Marshal(msg)
	if err != nil {
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
	identity, err := a.getLocalIdentity()
	if err != nil {
		return err
	}

	adminPubkey := strings.TrimSpace(identity.PublicKey)
	trusted, err := a.isTrustedAdmin(adminPubkey)
	if err != nil {
		return err
	}
	if !trusted {
		return errors.New("admin pubkey is not trusted")
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
		Type:                   "GOVERNANCE_POLICY_UPDATE",
		AdminPubkey:            adminPubkey,
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

func (a *App) fetchContentBlobFromNetwork(contentCID string, timeout time.Duration) error {
	contentCID = strings.TrimSpace(contentCID)
	if contentCID == "" {
		return errors.New("content cid is required")
	}
	if timeout <= 0 {
		timeout = 4 * time.Second
	}

	_, err, _ := a.contentFetchGroup.Do("cid:"+contentCID, func() (any, error) {
		a.p2pMu.Lock()
		topic := a.p2pTopic
		ctx := a.p2pCtx
		host := a.p2pHost
		a.p2pMu.Unlock()

		if topic == nil || ctx == nil || host == nil {
			return nil, errors.New("p2p not started")
		}
		if len(host.Network().Peers()) == 0 {
			return nil, errContentFetchNoPeers
		}

		requestID := buildMessageID(host.ID().String(), contentCID, time.Now().UnixNano())
		responseCh := make(chan IncomingMessage, 1)

		a.p2pMu.Lock()
		a.contentFetchWaiters[requestID] = responseCh
		a.p2pMu.Unlock()

		defer func() {
			a.p2pMu.Lock()
			delete(a.contentFetchWaiters, requestID)
			a.p2pMu.Unlock()
		}()

		request := IncomingMessage{
			Type:            messageTypeContentFetchRequest,
			RequestID:       requestID,
			RequesterPeerID: host.ID().String(),
			ContentCID:      contentCID,
			Timestamp:       time.Now().Unix(),
		}

		payload, marshalErr := json.Marshal(request)
		if marshalErr != nil {
			return nil, marshalErr
		}

		if publishErr := topic.Publish(ctx, payload); publishErr != nil {
			return nil, publishErr
		}

		select {
		case response := <-responseCh:
			if !response.Found {
				return nil, errContentFetchNotFound
			}
			if upsertErr := a.upsertContentBlob(response.ContentCID, response.Content, response.SizeBytes); upsertErr != nil {
				return nil, upsertErr
			}
			return nil, nil
		case <-time.After(timeout):
			return nil, errContentFetchTimeout
		}
	})

	return err
}

func (a *App) fetchMediaBlobFromNetwork(contentCID string, timeout time.Duration) error {
	contentCID = strings.TrimSpace(contentCID)
	if contentCID == "" {
		return errors.New("media cid is required")
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	_, err, _ := a.mediaFetchGroup.Do("media:"+contentCID, func() (any, error) {
		a.p2pMu.Lock()
		topic := a.p2pTopic
		ctx := a.p2pCtx
		host := a.p2pHost
		a.p2pMu.Unlock()

		if topic == nil || ctx == nil || host == nil {
			return nil, errors.New("p2p not started")
		}
		if len(host.Network().Peers()) == 0 {
			return nil, errMediaFetchNoPeers
		}

		requestID := buildMessageID(host.ID().String(), "media:"+contentCID, time.Now().UnixNano())
		responseCh := make(chan IncomingMessage, 1)

		a.p2pMu.Lock()
		a.mediaFetchWaiters[requestID] = responseCh
		a.p2pMu.Unlock()

		defer func() {
			a.p2pMu.Lock()
			delete(a.mediaFetchWaiters, requestID)
			a.p2pMu.Unlock()
		}()

		request := IncomingMessage{
			Type:            messageTypeMediaFetchRequest,
			RequestID:       requestID,
			RequesterPeerID: host.ID().String(),
			ContentCID:      contentCID,
			Timestamp:       time.Now().Unix(),
		}

		payload, marshalErr := json.Marshal(request)
		if marshalErr != nil {
			return nil, marshalErr
		}

		if publishErr := topic.Publish(ctx, payload); publishErr != nil {
			return nil, publishErr
		}

		select {
		case response := <-responseCh:
			if !response.Found {
				return nil, errMediaFetchNotFound
			}
			raw, decodeErr := base64.StdEncoding.DecodeString(strings.TrimSpace(response.ImageDataBase64))
			if decodeErr != nil || len(raw) == 0 {
				return nil, errors.New("invalid media response payload")
			}
			if upsertErr := a.upsertMediaBlobRaw(response.ContentCID, response.ImageMIME, raw, response.ImageWidth, response.ImageHeight, response.IsThumbnail); upsertErr != nil {
				return nil, upsertErr
			}
			return nil, nil
		case <-time.After(timeout):
			return nil, errMediaFetchTimeout
		}
	})

	return err
}

func (a *App) TriggerAntiEntropySyncNow() error {
	return a.publishSyncSummaryRequest()
}

func (a *App) runAntiEntropySyncWorker(ctx context.Context, localPeerID peer.ID) {
	_ = localPeerID

	interval := resolveAntiEntropyInterval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	initialTimer := time.NewTimer(2 * time.Second)
	defer initialTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-initialTimer.C:
			if err := a.publishSyncSummaryRequest(); err != nil &&
				!errors.Is(err, errAntiEntropyNoPeers) &&
				!strings.Contains(strings.ToLower(err.Error()), "p2p not started") &&
				a.ctx != nil {
				runtime.LogWarningf(a.ctx, "anti-entropy initial sync failed: %v", err)
			}
		case <-ticker.C:
			if err := a.publishSyncSummaryRequest(); err != nil &&
				!errors.Is(err, errAntiEntropyNoPeers) &&
				!strings.Contains(strings.ToLower(err.Error()), "p2p not started") &&
				a.ctx != nil {
				runtime.LogWarningf(a.ctx, "anti-entropy periodic sync failed: %v", err)
			}
		}
	}
}

func resolveAntiEntropyInterval() time.Duration {
	raw := strings.TrimSpace(os.Getenv("AEGIS_ANTI_ENTROPY_INTERVAL_SEC"))
	if raw != "" {
		if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}

	return 12 * time.Second
}

func resolveAntiEntropyWindowSeconds() int64 {
	raw := strings.TrimSpace(os.Getenv("AEGIS_ANTI_ENTROPY_WINDOW_SEC"))
	if raw != "" {
		if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
			return int64(seconds)
		}
	}

	return int64(30 * 24 * 3600)
}

func resolveAntiEntropyBatchSize() int {
	raw := strings.TrimSpace(os.Getenv("AEGIS_ANTI_ENTROPY_BATCH_SIZE"))
	if raw != "" {
		if size, err := strconv.Atoi(raw); err == nil && size > 0 && size <= 500 {
			return size
		}
	}

	return 200
}

func resolveAntiEntropyIndexInsertBudget() int {
	raw := strings.TrimSpace(os.Getenv("AEGIS_ANTI_ENTROPY_INDEX_BUDGET"))
	if raw != "" {
		if size, err := strconv.Atoi(raw); err == nil && size > 0 && size <= 1000 {
			return size
		}
	}

	return 240
}

func resolveAntiEntropyBodyFetchBudget() int {
	raw := strings.TrimSpace(os.Getenv("AEGIS_ANTI_ENTROPY_BODY_BUDGET"))
	if raw != "" {
		if size, err := strconv.Atoi(raw); err == nil && size > 0 && size <= 200 {
			return size
		}
	}

	return 16
}

func (a *App) updateAntiEntropyStats(apply func(stats *AntiEntropyStats)) {
	a.antiEntropyMu.Lock()
	defer a.antiEntropyMu.Unlock()
	apply(&a.antiEntropyStats)
}

func (a *App) publishSyncSummaryRequest() error {
	a.p2pMu.Lock()
	topic := a.p2pTopic
	ctx := a.p2pCtx
	host := a.p2pHost
	a.p2pMu.Unlock()

	if topic == nil || ctx == nil || host == nil {
		return errors.New("p2p not started")
	}
	if len(host.Network().Peers()) == 0 {
		return errAntiEntropyNoPeers
	}

	latestTimestamp, latestErr := a.getLatestPublicPostTimestamp()
	if latestErr != nil {
		return latestErr
	}

	windowSeconds := resolveAntiEntropyWindowSeconds()
	sinceTimestamp := int64(0)
	if latestTimestamp > 0 {
		sinceTimestamp = latestTimestamp - windowSeconds
		if sinceTimestamp < 0 {
			sinceTimestamp = 0
		}
	}

	batchSize := resolveAntiEntropyBatchSize()

	request := IncomingMessage{
		Type:               messageTypeSyncSummaryRequest,
		RequestID:          buildMessageID(host.ID().String(), "sync-summary", time.Now().UnixNano()),
		RequesterPeerID:    host.ID().String(),
		SyncSinceTimestamp: sinceTimestamp,
		SyncWindowSeconds:  windowSeconds,
		SyncBatchSize:      batchSize,
		Timestamp:          time.Now().Unix(),
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return err
	}

	a.updateAntiEntropyStats(func(stats *AntiEntropyStats) {
		stats.SyncRequestsSent++
	})
	if a.ctx != nil {
		runtime.LogInfof(a.ctx, "anti_entropy.request sent request_id=%s since=%d window=%d batch=%d", request.RequestID, request.SyncSinceTimestamp, request.SyncWindowSeconds, request.SyncBatchSize)
	}

	return topic.Publish(ctx, payload)
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

		remotePeerID := strings.TrimSpace(message.ReceivedFrom.String())
		if blocked, reason := a.isPeerBlocked(remotePeerID); blocked {
			a.p2pMu.Lock()
			host := a.p2pHost
			a.p2pMu.Unlock()
			if host != nil {
				_ = host.Network().ClosePeer(message.ReceivedFrom)
			}
			if a.ctx != nil {
				runtime.LogWarningf(a.ctx, "drop message from blocked peer peer=%s reason=%s", remotePeerID, reason)
			}
			continue
		}

		var incoming IncomingMessage
		_ = json.Unmarshal(message.Data, &incoming)
		messageType := strings.ToUpper(strings.TrimSpace(incoming.Type))

		switch messageType {
		case messageTypeContentFetchRequest:
			a.handleContentFetchRequest(localPeerID.String(), message.ReceivedFrom.String(), incoming)
			continue
		case messageTypeContentFetchResponse:
			a.handleContentFetchResponse(localPeerID.String(), incoming)
			continue
		case messageTypeMediaFetchRequest:
			a.handleMediaFetchRequest(localPeerID.String(), message.ReceivedFrom.String(), incoming)
			continue
		case messageTypeMediaFetchResponse:
			a.handleMediaFetchResponse(localPeerID.String(), incoming)
			continue
		case messageTypeSyncSummaryRequest:
			a.handleSyncSummaryRequest(localPeerID.String(), incoming)
			continue
		case messageTypeSyncSummaryResponse:
			a.handleSyncSummaryResponse(localPeerID.String(), incoming)
			continue
		}

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

func (a *App) handleContentFetchRequest(localPeerID string, remotePeerID string, message IncomingMessage) {
	requester := strings.TrimSpace(remotePeerID)
	contentCID := strings.TrimSpace(message.ContentCID)
	requestID := strings.TrimSpace(message.RequestID)
	if requester == "" || contentCID == "" || requestID == "" {
		return
	}
	if requester == localPeerID {
		return
	}
	if !a.allowFetchRequest(requester, "content") {
		return
	}

	body, err := a.getContentBlobLocal(contentCID)
	if err != nil {
		return
	}

	response := IncomingMessage{
		Type:            messageTypeContentFetchResponse,
		RequestID:       requestID,
		RequesterPeerID: requester,
		ResponderPeerID: localPeerID,
		ContentCID:      body.ContentCID,
		Content:         body.Body,
		SizeBytes:       body.SizeBytes,
		Found:           true,
		Timestamp:       time.Now().Unix(),
	}

	a.p2pMu.Lock()
	topic := a.p2pTopic
	ctx := a.p2pCtx
	a.p2pMu.Unlock()
	if topic == nil || ctx == nil {
		return
	}

	payload, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		return
	}
	_ = topic.Publish(ctx, payload)
}

func (a *App) handleContentFetchResponse(localPeerID string, message IncomingMessage) {
	requester := strings.TrimSpace(message.RequesterPeerID)
	if requester == "" || requester != localPeerID {
		return
	}
	requestID := strings.TrimSpace(message.RequestID)
	if requestID == "" {
		return
	}

	a.p2pMu.Lock()
	waiter, ok := a.contentFetchWaiters[requestID]
	a.p2pMu.Unlock()
	if !ok {
		return
	}

	select {
	case waiter <- message:
	default:
	}
}

func (a *App) handleMediaFetchRequest(localPeerID string, remotePeerID string, message IncomingMessage) {
	requester := strings.TrimSpace(remotePeerID)
	contentCID := strings.TrimSpace(message.ContentCID)
	requestID := strings.TrimSpace(message.RequestID)
	if requester == "" || contentCID == "" || requestID == "" {
		return
	}
	if requester == localPeerID {
		return
	}
	if !a.allowFetchRequest(requester, "media") {
		return
	}

	media, raw, err := a.getMediaBlobRawLocal(contentCID)
	if err != nil {
		return
	}

	response := IncomingMessage{
		Type:            messageTypeMediaFetchResponse,
		RequestID:       requestID,
		RequesterPeerID: requester,
		ResponderPeerID: localPeerID,
		ContentCID:      media.ContentCID,
		ImageDataBase64: base64.StdEncoding.EncodeToString(raw),
		ImageMIME:       media.Mime,
		ImageWidth:      media.Width,
		ImageHeight:     media.Height,
		IsThumbnail:     media.IsThumbnail,
		Found:           true,
		Timestamp:       time.Now().Unix(),
	}

	a.p2pMu.Lock()
	topic := a.p2pTopic
	ctx := a.p2pCtx
	a.p2pMu.Unlock()
	if topic == nil || ctx == nil {
		return
	}

	payload, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		return
	}
	_ = topic.Publish(ctx, payload)
}

func resolveFetchRateLimitPerWindow() int {
	raw := strings.TrimSpace(os.Getenv("AEGIS_FETCH_REQUEST_LIMIT"))
	if raw == "" {
		return 60
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 60
	}

	return value
}

func resolveFetchRateWindowSeconds() int64 {
	raw := strings.TrimSpace(os.Getenv("AEGIS_FETCH_REQUEST_WINDOW_SEC"))
	if raw == "" {
		return 60
	}

	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 60
	}

	return value
}

func (a *App) allowFetchRequest(peerID string, requestType string) bool {
	peerID = strings.TrimSpace(peerID)
	requestType = strings.TrimSpace(strings.ToLower(requestType))
	if peerID == "" || requestType == "" {
		return false
	}

	limit := resolveFetchRateLimitPerWindow()
	windowSec := resolveFetchRateWindowSeconds()
	now := time.Now().Unix()
	stateKey := requestType + ":" + peerID

	a.fetchRateMu.Lock()
	defer a.fetchRateMu.Unlock()

	window, exists := a.fetchRateState[stateKey]
	if !exists || now-window.StartedAt >= windowSec {
		a.fetchRateState[stateKey] = fetchRateWindow{StartedAt: now, Count: 1}
		return true
	}

	if window.Count >= limit {
		if a.ctx != nil {
			runtime.LogWarningf(a.ctx, "fetch request rate limited type=%s requester=%s count=%d limit=%d window_sec=%d", requestType, peerID, window.Count, limit, windowSec)
		}
		a.markPeerGreylisted(peerID, "fetch-rate-limit")
		return false
	}

	window.Count += 1
	a.fetchRateState[stateKey] = window
	return true
}

func (a *App) handleMediaFetchResponse(localPeerID string, message IncomingMessage) {
	requester := strings.TrimSpace(message.RequesterPeerID)
	if requester == "" || requester != localPeerID {
		return
	}
	requestID := strings.TrimSpace(message.RequestID)
	if requestID == "" {
		return
	}

	a.p2pMu.Lock()
	waiter, ok := a.mediaFetchWaiters[requestID]
	a.p2pMu.Unlock()
	if !ok {
		return
	}

	select {
	case waiter <- message:
	default:
	}
}

func (a *App) handleSyncSummaryRequest(localPeerID string, message IncomingMessage) {
	requester := strings.TrimSpace(message.RequesterPeerID)
	requestID := strings.TrimSpace(message.RequestID)
	if requester == "" || requestID == "" || requester == localPeerID {
		return
	}

	batchSize := message.SyncBatchSize
	if batchSize <= 0 || batchSize > 500 {
		batchSize = resolveAntiEntropyBatchSize()
	}

	sinceTimestamp := message.SyncSinceTimestamp
	if sinceTimestamp < 0 {
		sinceTimestamp = 0
	}

	summaries, err := a.listPublicPostDigestsSince(sinceTimestamp, batchSize)
	if err != nil {
		if a.ctx != nil {
			runtime.LogWarningf(a.ctx, "anti-entropy build summary failed: %v", err)
		}
		return
	}

	a.updateAntiEntropyStats(func(stats *AntiEntropyStats) {
		stats.SyncRequestsReceived++
	})
	if a.ctx != nil {
		runtime.LogInfof(a.ctx, "anti_entropy.request received request_id=%s since=%d batch=%d summaries=%d", requestID, sinceTimestamp, batchSize, len(summaries))
	}

	response := IncomingMessage{
		Type:               messageTypeSyncSummaryResponse,
		RequestID:          requestID,
		RequesterPeerID:    requester,
		ResponderPeerID:    localPeerID,
		SyncSinceTimestamp: sinceTimestamp,
		SyncBatchSize:      batchSize,
		Summaries:          summaries,
		Timestamp:          time.Now().Unix(),
	}

	a.p2pMu.Lock()
	topic := a.p2pTopic
	ctx := a.p2pCtx
	a.p2pMu.Unlock()
	if topic == nil || ctx == nil {
		return
	}

	payload, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		return
	}

	_ = topic.Publish(ctx, payload)
}

func (a *App) handleSyncSummaryResponse(localPeerID string, message IncomingMessage) {
	requester := strings.TrimSpace(message.RequesterPeerID)
	if requester == "" || requester != localPeerID {
		return
	}

	a.updateAntiEntropyStats(func(stats *AntiEntropyStats) {
		stats.SyncResponsesReceived++
		stats.SyncSummariesReceived += int64(len(message.Summaries))
		stats.LastSyncAt = time.Now().Unix()
	})

	summaries := make([]SyncPostDigest, 0, len(message.Summaries))
	for _, item := range message.Summaries {
		if strings.TrimSpace(item.ID) == "" || strings.TrimSpace(item.ContentCID) == "" {
			continue
		}
		summaries = append(summaries, item)
	}

	sort.SliceStable(summaries, func(i int, j int) bool {
		return summaries[i].Timestamp > summaries[j].Timestamp
	})

	if len(summaries) == 0 {
		return
	}

	insertedAny := false
	indexBudget := resolveAntiEntropyIndexInsertBudget()
	bodyFetchBudget := resolveAntiEntropyBodyFetchBudget()
	mediaFetchBudget := resolveAntiEntropyBodyFetchBudget()
	missingCIDs := make([]string, 0, bodyFetchBudget)
	missingMediaCIDs := make([]string, 0, mediaFetchBudget)
	seenCIDs := make(map[string]struct{}, bodyFetchBudget)
	seenMediaCIDs := make(map[string]struct{}, mediaFetchBudget)
	indexInsertions := int64(0)
	remoteMaxTimestamp := int64(0)

	for _, digest := range summaries {
		if digest.Timestamp > remoteMaxTimestamp {
			remoteMaxTimestamp = digest.Timestamp
		}

		if indexBudget > 0 {
			inserted, err := a.upsertPublicPostIndexFromDigest(digest)
			if err != nil {
				continue
			}
			if inserted {
				insertedAny = true
				indexInsertions++
			}
			indexBudget--
		}

		if bodyFetchBudget > 0 {
			hasBlob, blobErr := a.hasContentBlobLocal(digest.ContentCID)
			if blobErr == nil && !hasBlob {
				cid := strings.TrimSpace(digest.ContentCID)
				if cid != "" {
					if _, exists := seenCIDs[cid]; !exists {
						seenCIDs[cid] = struct{}{}
						missingCIDs = append(missingCIDs, cid)
						bodyFetchBudget--
					}
				}
			}
		}

		imageCID := strings.TrimSpace(digest.ImageCID)
		if imageCID == "" || mediaFetchBudget <= 0 {
			continue
		}
		hasMedia, mediaErr := a.hasMediaBlobLocal(imageCID)
		if mediaErr != nil || hasMedia {
			continue
		}
		if _, exists := seenMediaCIDs[imageCID]; exists {
			continue
		}
		seenMediaCIDs[imageCID] = struct{}{}
		missingMediaCIDs = append(missingMediaCIDs, imageCID)
		mediaFetchBudget--
	}

	a.updateAntiEntropyStats(func(stats *AntiEntropyStats) {
		stats.IndexInsertions += indexInsertions
		if remoteMaxTimestamp > stats.LastRemoteSummaryTs {
			stats.LastRemoteSummaryTs = remoteMaxTimestamp
		}
	})

	if a.ctx != nil {
		runtime.LogInfof(a.ctx, "anti_entropy.response applied request_id=%s summaries=%d inserted=%d text_fetch_candidates=%d media_fetch_candidates=%d", strings.TrimSpace(message.RequestID), len(summaries), indexInsertions, len(missingCIDs), len(missingMediaCIDs))
	}

	for _, contentCID := range missingCIDs {
		cid := contentCID
		if cid == "" {
			continue
		}

		a.updateAntiEntropyStats(func(stats *AntiEntropyStats) {
			stats.BlobFetchAttempts++
		})

		go func() {
			if err := a.fetchContentBlobFromNetwork(cid, 3*time.Second); err != nil {
				a.updateAntiEntropyStats(func(stats *AntiEntropyStats) {
					stats.BlobFetchFailures++
				})
				if a.ctx != nil {
					runtime.LogWarningf(a.ctx, "anti_entropy.blob_fetch failed cid=%s err=%v", cid, err)
				}
				return
			}

			a.updateAntiEntropyStats(func(stats *AntiEntropyStats) {
				stats.BlobFetchSuccess++
			})
			if a.ctx != nil {
				runtime.LogInfof(a.ctx, "anti_entropy.blob_fetch success cid=%s", cid)
			}
		}()
	}

	for _, mediaCID := range missingMediaCIDs {
		cid := mediaCID
		if cid == "" {
			continue
		}

		a.updateAntiEntropyStats(func(stats *AntiEntropyStats) {
			stats.BlobFetchAttempts++
		})

		go func() {
			if err := a.fetchMediaBlobFromNetwork(cid, 4*time.Second); err != nil {
				a.updateAntiEntropyStats(func(stats *AntiEntropyStats) {
					stats.BlobFetchFailures++
				})
				if a.ctx != nil {
					runtime.LogWarningf(a.ctx, "anti_entropy.media_fetch failed cid=%s err=%v", cid, err)
				}
				return
			}

			a.updateAntiEntropyStats(func(stats *AntiEntropyStats) {
				stats.BlobFetchSuccess++
			})
			if a.ctx != nil {
				runtime.LogInfof(a.ctx, "anti_entropy.media_fetch success cid=%s", cid)
			}
		}()
	}

	if remoteMaxTimestamp > 0 {
		latestLocalTs, err := a.getLatestPublicPostTimestamp()
		if err == nil {
			lag := remoteMaxTimestamp - latestLocalTs
			if lag < 0 {
				lag = 0
			}
			a.updateAntiEntropyStats(func(stats *AntiEntropyStats) {
				stats.LastObservedSyncLagSec = lag
			})
		}
	}

	if insertedAny && a.ctx != nil {
		runtime.EventsEmit(a.ctx, "feed:updated")
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

	peerID := strings.TrimSpace(info.ID.String())
	if blocked, reason := n.app.isPeerBlocked(peerID); blocked {
		if n.app.ctx != nil {
			runtime.LogWarningf(n.app.ctx, "mdns peer rejected by policy peer=%s reason=%s", peerID, reason)
		}
		return
	}

	if n.app.p2pHost.Network().Connectedness(info.ID) != network.Connected && len(n.app.p2pHost.Network().Peers()) >= resolveMaxConnectedPeers() {
		if n.app.ctx != nil {
			runtime.LogWarningf(n.app.ctx, "mdns peer rejected by max-peer limit peer=%s limit=%d", peerID, resolveMaxConnectedPeers())
		}
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

func resolveRelayPeers() []string {
	raw := strings.TrimSpace(os.Getenv("AEGIS_RELAY_PEERS"))
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	peers := make([]string, 0, len(parts))
	for _, candidate := range parts {
		peerAddr := strings.TrimSpace(candidate)
		if peerAddr == "" {
			continue
		}
		peers = append(peers, peerAddr)
	}
	return peers
}

func resolveRelayPeerInfos(bootstrapPeers []string) []peer.AddrInfo {
	candidates := make([]string, 0, len(bootstrapPeers)+4)
	candidates = append(candidates, bootstrapPeers...)
	candidates = append(candidates, resolveRelayPeers()...)

	infos := make([]peer.AddrInfo, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		trimmed := strings.TrimSpace(candidate)
		if trimmed == "" {
			continue
		}
		maddr, err := multiaddr.NewMultiaddr(trimmed)
		if err != nil {
			continue
		}
		info, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			continue
		}
		peerID := info.ID.String()
		if peerID == "" {
			continue
		}
		if _, exists := seen[peerID]; exists {
			continue
		}
		seen[peerID] = struct{}{}
		infos = append(infos, *info)
	}

	return infos
}

func resolveMaxConnectedPeers() int {
	raw := strings.TrimSpace(os.Getenv("AEGIS_MAX_CONNECTED_PEERS"))
	if raw == "" {
		return 64
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 64
	}

	return value
}

func resolveGreylistTTLSeconds() int64 {
	raw := strings.TrimSpace(os.Getenv("AEGIS_P2P_GREYLIST_TTL_SEC"))
	if raw == "" {
		return 300
	}

	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 300
	}

	return value
}

func parsePeerIDSet(raw string) map[string]struct{} {
	result := make(map[string]struct{})
	parts := strings.Split(strings.TrimSpace(raw), ",")
	for _, part := range parts {
		peerID := strings.TrimSpace(part)
		if peerID == "" {
			continue
		}
		result[peerID] = struct{}{}
	}
	return result
}

func (a *App) refreshPeerPoliciesFromEnv() {
	blacklist := parsePeerIDSet(os.Getenv("AEGIS_P2P_BLACKLIST_PEERS"))
	seedGrey := parsePeerIDSet(os.Getenv("AEGIS_P2P_GREYLIST_PEERS"))
	now := time.Now().Unix()
	until := now + resolveGreylistTTLSeconds()

	a.peerPolicyMu.Lock()
	defer a.peerPolicyMu.Unlock()

	a.peerBlacklist = blacklist
	if a.peerGreylist == nil {
		a.peerGreylist = make(map[string]int64)
	}
	for peerID := range seedGrey {
		a.peerGreylist[peerID] = until
	}
}

func (a *App) markPeerGreylisted(peerID string, reason string) {
	peerID = strings.TrimSpace(peerID)
	if peerID == "" {
		return
	}

	until := time.Now().Unix() + resolveGreylistTTLSeconds()
	a.peerPolicyMu.Lock()
	if a.peerGreylist == nil {
		a.peerGreylist = make(map[string]int64)
	}
	a.peerGreylist[peerID] = until
	a.peerPolicyMu.Unlock()

	if a.ctx != nil {
		runtime.LogWarningf(a.ctx, "peer moved to greylist peer=%s reason=%s until=%d", peerID, strings.TrimSpace(reason), until)
	}
}

func (a *App) isPeerBlocked(peerID string) (bool, string) {
	peerID = strings.TrimSpace(peerID)
	if peerID == "" {
		return false, ""
	}

	now := time.Now().Unix()
	a.peerPolicyMu.Lock()
	defer a.peerPolicyMu.Unlock()

	if _, blocked := a.peerBlacklist[peerID]; blocked {
		return true, "blacklist"
	}

	until, listed := a.peerGreylist[peerID]
	if !listed {
		return false, ""
	}
	if until <= now {
		delete(a.peerGreylist, peerID)
		return false, ""
	}

	return true, "greylist"
}
