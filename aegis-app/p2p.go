package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
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
	messageTypeContentFetchRequest    = "CONTENT_FETCH_REQUEST"
	messageTypeContentFetchResponse   = "CONTENT_FETCH_RESPONSE"
	messageTypeMediaFetchRequest      = "MEDIA_FETCH_REQUEST"
	messageTypeMediaFetchResponse     = "MEDIA_FETCH_RESPONSE"
	messageTypeSyncSummaryRequest     = "SYNC_SUMMARY_REQUEST"
	messageTypeSyncSummaryResponse    = "SYNC_SUMMARY_RESPONSE"
	messageTypeCommentSyncRequest     = "COMMENT_SYNC_REQUEST"
	messageTypeCommentSyncResponse    = "COMMENT_SYNC_RESPONSE"
	messageTypeGovernanceSyncRequest  = "GOVERNANCE_SYNC_REQUEST"
	messageTypeGovernanceSyncResponse = "GOVERNANCE_SYNC_RESPONSE"
	messageTypeFavoriteOp             = "FAVORITE_OP"
	messageTypeFavoriteSyncRequest    = "FAVORITE_SYNC_REQUEST"
	messageTypeFavoriteSyncResponse   = "FAVORITE_SYNC_RESPONSE"
	messageTypePeerExchangeRequest    = "PEER_EXCHANGE_REQUEST"
	messageTypePeerExchangeResponse   = "PEER_EXCHANGE_RESPONSE"
)

var (
	errContentFetchNoPeers   = errors.New("content fetch no peers")
	errContentFetchTimeout   = errors.New("content fetch timeout")
	errContentFetchNotFound  = errors.New("content fetch not found")
	errMediaFetchNoPeers     = errors.New("media fetch no peers")
	errMediaFetchTimeout     = errors.New("media fetch timeout")
	errMediaFetchNotFound    = errors.New("media fetch not found")
	errAntiEntropyNoPeers    = errors.New("anti-entropy no peers")
	errCommentSyncNoPeers    = errors.New("comment sync no peers")
	errGovernanceSyncNoPeers = errors.New("governance sync no peers")
	errFavoriteSyncNoPeers   = errors.New("favorite sync no peers")
)

type fetchRateWindow struct {
	StartedAt int64
	Count     int
}

type P2PStatus struct {
	Started        bool     `json:"started"`
	PeerID         string   `json:"peerId"`
	ListenAddrs    []string `json:"listenAddrs"`
	AnnounceAddrs  []string `json:"announceAddrs"`
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
	var startErr error
	for _, candidatePort := range resolveAutoStartPortCandidates(listenPort) {
		if !isTCPPortAvailable(candidatePort) {
			continue
		}

		status, err := a.startP2POnPortLocked(candidatePort, bootstrapPeers)
		if err != nil {
			startErr = err
			continue
		}

		if candidatePort != listenPort && a.ctx != nil {
			runtime.LogInfof(a.ctx, "p2p start fallback port selected: %d (preferred: %d)", candidatePort, listenPort)
		}

		return status, nil
	}

	if startErr != nil {
		return P2PStatus{}, startErr
	}

	return P2PStatus{}, fmt.Errorf("p2p start failed: no available port in range [%d,%d]", listenPort, listenPort+20)
}

func (a *App) startP2POnPortLocked(listenPort int, bootstrapPeers []string) (P2PStatus, error) {
	a.refreshPeerPoliciesFromEnv()

	listenAddrs := resolveP2PListenAddrs(listenPort)
	announceAddrs := resolveP2PAnnounceAddrs(listenPort)
	relayInfos := resolveRelayPeerInfos(bootstrapPeers)

	options := []libp2p.Option{
		libp2p.ListenAddrStrings(listenAddrs...),
		libp2p.NATPortMap(),
		libp2p.EnableNATService(),
		libp2p.EnableAutoNATv2(),
		libp2p.EnableHolePunching(),
		libp2p.EnableRelay(),
	}
	if resolveRelayServiceEnabled() {
		options = append(options, libp2p.EnableRelayService())
	}
	if len(announceAddrs) > 0 {
		announceAddrsCopy := append([]multiaddr.Multiaddr(nil), announceAddrs...)
		options = append(options, libp2p.AddrsFactory(func(_ []multiaddr.Multiaddr) []multiaddr.Multiaddr {
			return append([]multiaddr.Multiaddr(nil), announceAddrsCopy...)
		}))
	}
	if len(relayInfos) > 0 {
		options = append(options, libp2p.EnableAutoRelayWithStaticRelays(relayInfos))
	}

	host, err := libp2p.New(options...)
	if err != nil && len(listenAddrs) > 1 {
		dualStackErr := err
		fallbackOptions := make([]libp2p.Option, 0, len(options))
		fallbackOptions = append(fallbackOptions,
			libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)),
		)
		fallbackOptions = append(fallbackOptions, options[1:]...)
		host, err = libp2p.New(fallbackOptions...)
		if err == nil && a.ctx != nil {
			runtime.LogWarningf(a.ctx, "p2p dual-stack bind failed, fallback to ipv4-only: %v", dualStackErr)
		}
	}
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
	go a.runPeerExchangeWorker(ctx, host.ID())
	go a.runReleaseAlertWorker(ctx)

	knownBootstraps := a.getKnownPeerBootstrapAddresses(knownPeerBootstrapLimit)
	bootstrapTargets := mergePeerAddressLists(bootstrapPeers, knownBootstraps)
	a.connectBootstrapPeersAsync(bootstrapTargets)

	a.publishLocalProfileUpdateLocked()

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

	if err = a.p2pHost.Connect(ctx, *info); err != nil {
		a.rememberConnectedPeer(*info, false)
		return err
	}
	a.rememberConnectedPeer(*info, true)

	a.publishLocalProfileUpdateLocked()
	return nil
}

func (a *App) connectBootstrapPeersAsync(addresses []string) {
	filtered := make([]string, 0, len(addresses))
	seen := make(map[string]struct{}, len(addresses))
	for _, raw := range addresses {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		filtered = append(filtered, trimmed)
	}

	if len(filtered) == 0 {
		return
	}

	go func(targets []string) {
		for _, addr := range targets {
			if err := a.ConnectPeer(addr); err != nil && a.ctx != nil {
				runtime.LogDebugf(a.ctx, "bootstrap connect skipped addr=%s err=%v", addr, err)
			}
		}
	}(append([]string(nil), filtered...))
}

func (a *App) PublishPostStructured(pubkey string, title string, body string) error {
	return a.PublishPostStructuredToSub(pubkey, title, body, defaultSubID)
}

func (a *App) publishPayloadAsync(topic *pubsub.Topic, payload []byte, label string) {
	if topic == nil || len(payload) == 0 {
		return
	}

	a.p2pMu.Lock()
	baseCtx := a.p2pCtx
	a.p2pMu.Unlock()
	if baseCtx == nil {
		return
	}

	go func(ctx context.Context, t *pubsub.Topic, data []byte, kind string) {
		publishCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		if err := t.Publish(publishCtx, data); err != nil {
			if a.ctx != nil {
				runtime.LogWarningf(a.ctx, "async publish failed (%s): %v", kind, err)
			}
		}
	}(baseCtx, topic, append([]byte(nil), payload...), label)
}

func resolveVoteBroadcastDebounce() time.Duration {
	raw := strings.TrimSpace(os.Getenv("AEGIS_VOTE_BROADCAST_DEBOUNCE_MS"))
	if raw == "" {
		return 600 * time.Millisecond
	}
	ms, err := strconv.Atoi(raw)
	if err != nil {
		return 600 * time.Millisecond
	}
	if ms < 100 {
		ms = 100
	}
	if ms > 3000 {
		ms = 3000
	}
	return time.Duration(ms) * time.Millisecond
}

func (a *App) scheduleVoteStateBroadcast(voterPubkey string, postID string, commentID string) {
	voterPubkey = strings.TrimSpace(voterPubkey)
	postID = strings.TrimSpace(postID)
	commentID = strings.TrimSpace(commentID)
	if voterPubkey == "" || postID == "" {
		return
	}

	key := fmt.Sprintf("%s|%s|%s", voterPubkey, postID, commentID)
	a.voteBroadcastMu.Lock()
	seq := a.voteBroadcastSeq[key] + 1
	a.voteBroadcastSeq[key] = seq
	a.voteBroadcastMu.Unlock()

	go func(expected int64, voteKey string, pubkey string, pid string, cid string) {
		time.Sleep(resolveVoteBroadcastDebounce())

		a.voteBroadcastMu.Lock()
		current := a.voteBroadcastSeq[voteKey]
		if current != expected {
			a.voteBroadcastMu.Unlock()
			return
		}
		delete(a.voteBroadcastSeq, voteKey)
		a.voteBroadcastMu.Unlock()

		a.p2pMu.Lock()
		topic := a.p2pTopic
		a.p2pMu.Unlock()
		if topic == nil {
			return
		}

		now := time.Now().Unix()
		if cid == "" {
			state, err := a.getPostVoteState(pubkey, pid)
			if err != nil {
				if a.ctx != nil {
					runtime.LogWarningf(a.ctx, "vote state read failed (post): %v", err)
				}
				return
			}
			msg := IncomingMessage{
				Type:        "POST_VOTE_SET",
				OpID:        generateOperationID(pid, pubkey, time.Now().UnixNano()),
				Pubkey:      pubkey,
				VoterPubkey: pubkey,
				PostID:      pid,
				VoteState:   state,
				Timestamp:   now,
			}
			payload, err := json.Marshal(msg)
			if err != nil {
				return
			}
			a.publishPayloadAsync(topic, payload, "POST_VOTE_SET")
			return
		}

		state, err := a.getCommentVoteState(pubkey, cid)
		if err != nil {
			if a.ctx != nil {
				runtime.LogWarningf(a.ctx, "vote state read failed (comment): %v", err)
			}
			return
		}
		msg := IncomingMessage{
			Type:        "COMMENT_VOTE_SET",
			OpID:        generateOperationID(cid, pubkey, time.Now().UnixNano()),
			Pubkey:      pubkey,
			VoterPubkey: pubkey,
			PostID:      pid,
			CommentID:   cid,
			VoteState:   state,
			Timestamp:   now,
		}
		payload, err := json.Marshal(msg)
		if err != nil {
			return
		}
		a.publishPayloadAsync(topic, payload, "COMMENT_VOTE_SET")
	}(seq, key, voterPubkey, postID, commentID)
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
	a.p2pMu.Unlock()

	profile, profileErr := a.GetProfile(pubkey)
	if profileErr != nil {
		profile = Profile{}
	}

	localPost, err := a.AddLocalPostStructuredToSub(pubkey, title, body, "public", subID)
	if err != nil {
		return err
	}

	msg := IncomingMessage{
		Type:          "POST",
		OpType:        postOpTypeCreate,
		OpID:          localPost.OpID,
		SchemaVersion: lamportSchemaV2,
		AuthScope:     authScopeUser,
		ID:            localPost.ID,
		Pubkey:        pubkey,
		DisplayName:   strings.TrimSpace(profile.DisplayName),
		AvatarURL:     strings.TrimSpace(profile.AvatarURL),
		Title:         title,
		Body:          body,
		ContentCID:    localPost.ContentCID,
		Content:       "",
		SubID:         normalizeSubID(subID),
		Timestamp:     localPost.Timestamp,
		Lamport:       localPost.Lamport,
		Signature:     "",
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	a.publishPayloadAsync(topic, payload, "POST")
	return nil
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
	a.p2pMu.Unlock()

	profile, profileErr := a.GetProfile(pubkey)
	if profileErr != nil {
		profile = Profile{}
	}

	localPost, err := a.AddLocalPostWithImageToSub(pubkey, title, body, "public", subID, imageBase64, imageMIME)
	if err != nil {
		return err
	}

	msg := IncomingMessage{
		Type:          "POST",
		OpType:        postOpTypeCreate,
		OpID:          localPost.OpID,
		SchemaVersion: lamportSchemaV2,
		AuthScope:     authScopeUser,
		ID:            localPost.ID,
		Pubkey:        pubkey,
		DisplayName:   strings.TrimSpace(profile.DisplayName),
		AvatarURL:     strings.TrimSpace(profile.AvatarURL),
		Title:         title,
		Body:          body,
		ContentCID:    localPost.ContentCID,
		ImageCID:      localPost.ImageCID,
		ThumbCID:      localPost.ThumbCID,
		ImageMIME:     localPost.ImageMIME,
		ImageSize:     localPost.ImageSize,
		ImageWidth:    localPost.ImageWidth,
		ImageHeight:   localPost.ImageHeight,
		Content:       "",
		SubID:         normalizeSubID(subID),
		Timestamp:     localPost.Timestamp,
		Lamport:       localPost.Lamport,
		Signature:     "",
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	a.publishPayloadAsync(topic, payload, "POST_WITH_IMAGE")
	return nil
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

	profile, profileErr := a.GetProfile(pubkey)
	if profileErr != nil {
		profile = Profile{}
	}
	localComment, err := a.AddLocalComment(pubkey, postID, parentID, body)
	if err != nil {
		return err
	}
	msg := IncomingMessage{
		Type:               "COMMENT",
		OpType:             postOpTypeCreate,
		OpID:               localComment.OpID,
		SchemaVersion:      lamportSchemaV2,
		AuthScope:          authScopeUser,
		ID:                 localComment.ID,
		Pubkey:             pubkey,
		PostID:             postID,
		ParentID:           parentID,
		DisplayName:        strings.TrimSpace(profile.DisplayName),
		AvatarURL:          strings.TrimSpace(profile.AvatarURL),
		Body:               body,
		CommentAttachments: localComment.Attachments,
		Timestamp:          localComment.Timestamp,
		Lamport:            localComment.Lamport,
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	a.p2pMu.Lock()
	topic := a.p2pTopic
	a.p2pMu.Unlock()
	a.publishPayloadAsync(topic, payload, "COMMENT")
	return nil
}

func (a *App) PublishCommentWithAttachments(pubkey string, postID string, parentID string, body string, localImageDataURLs []string, externalImageURLs []string) error {
	pubkey = strings.TrimSpace(pubkey)
	postID = strings.TrimSpace(postID)
	parentID = strings.TrimSpace(parentID)
	body = strings.TrimSpace(body)

	if pubkey == "" || postID == "" {
		return errors.New("pubkey and post id are required")
	}
	if body == "" && len(localImageDataURLs) == 0 && len(externalImageURLs) == 0 {
		return errors.New("comment content is required")
	}

	attachments := make([]CommentAttachment, 0, len(localImageDataURLs)+len(externalImageURLs))
	for _, dataURL := range localImageDataURLs {
		dataURL = strings.TrimSpace(dataURL)
		if dataURL == "" {
			continue
		}
		item, err := a.StoreCommentImageDataURL(dataURL)
		if err != nil {
			return err
		}
		attachments = append(attachments, item)
	}
	for _, external := range externalImageURLs {
		external = strings.TrimSpace(external)
		if external == "" {
			continue
		}
		u, err := url.Parse(external)
		if err != nil {
			continue
		}
		scheme := strings.ToLower(strings.TrimSpace(u.Scheme))
		if scheme != "http" && scheme != "https" {
			continue
		}
		attachments = append(attachments, CommentAttachment{Kind: "external_url", Ref: u.String()})
	}
	attachments = normalizeCommentAttachments(attachments)
	if body == "" && len(attachments) == 0 {
		return errors.New("comment content is required")
	}

	shadowBanned, err := a.isShadowBanned(pubkey)
	if err != nil {
		return err
	}
	if shadowBanned {
		_, err = a.AddLocalCommentWithAttachments(pubkey, postID, parentID, body, attachments)
		return err
	}

	profile, profileErr := a.GetProfile(pubkey)
	if profileErr != nil {
		profile = Profile{}
	}
	localComment, err := a.AddLocalCommentWithAttachments(pubkey, postID, parentID, body, attachments)
	if err != nil {
		return err
	}

	msg := IncomingMessage{
		Type:               "COMMENT",
		OpType:             postOpTypeCreate,
		OpID:               localComment.OpID,
		SchemaVersion:      lamportSchemaV2,
		AuthScope:          authScopeUser,
		ID:                 localComment.ID,
		Pubkey:             pubkey,
		PostID:             postID,
		ParentID:           parentID,
		DisplayName:        strings.TrimSpace(profile.DisplayName),
		AvatarURL:          strings.TrimSpace(profile.AvatarURL),
		Body:               body,
		CommentAttachments: localComment.Attachments,
		Timestamp:          localComment.Timestamp,
		Lamport:            localComment.Lamport,
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	a.p2pMu.Lock()
	topic := a.p2pTopic
	a.p2pMu.Unlock()
	a.publishPayloadAsync(topic, payload, "COMMENT_WITH_ATTACHMENTS")
	return nil
}

func (a *App) PublishDeletePost(pubkey string, postID string) error {
	pubkey = strings.TrimSpace(pubkey)
	postID = strings.TrimSpace(postID)
	if pubkey == "" || postID == "" {
		return errors.New("pubkey and post id are required")
	}

	now := time.Now().Unix()
	lamport, err := a.nextLamport()
	if err != nil {
		return err
	}
	deleteOpID := generateOperationID(postID, pubkey, lamport)
	if err = a.deleteLocalPostAsAuthor(pubkey, postID, now, lamport, deleteOpID); err != nil {
		return err
	}

	a.p2pMu.Lock()
	topic := a.p2pTopic
	a.p2pMu.Unlock()
	if topic == nil {
		return nil
	}

	msg := IncomingMessage{
		Type:             "POST_DELETE",
		OpType:           postOpTypeDelete,
		OpID:             deleteOpID,
		SchemaVersion:    lamportSchemaV2,
		AuthScope:        authScopeUser,
		Pubkey:           pubkey,
		PostID:           postID,
		Timestamp:        now,
		Lamport:          lamport,
		DeletedAtLamport: lamport,
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	a.publishPayloadAsync(topic, payload, "POST_DELETE")
	return nil
}

func (a *App) PublishDeleteComment(pubkey string, commentID string) error {
	pubkey = strings.TrimSpace(pubkey)
	commentID = strings.TrimSpace(commentID)
	if pubkey == "" || commentID == "" {
		return errors.New("pubkey and comment id are required")
	}

	now := time.Now().Unix()
	lamport, err := a.nextLamport()
	if err != nil {
		return err
	}
	deleteOpID := generateOperationID(commentID, pubkey, lamport)
	postID, err := a.deleteLocalCommentAsAuthor(pubkey, commentID, now, lamport, deleteOpID)
	if err != nil {
		return err
	}

	a.p2pMu.Lock()
	topic := a.p2pTopic
	a.p2pMu.Unlock()
	if topic == nil {
		return nil
	}

	msg := IncomingMessage{
		Type:             "COMMENT_DELETE",
		OpType:           postOpTypeDelete,
		OpID:             deleteOpID,
		SchemaVersion:    lamportSchemaV2,
		AuthScope:        authScopeUser,
		Pubkey:           pubkey,
		CommentID:        commentID,
		PostID:           postID,
		Timestamp:        now,
		Lamport:          lamport,
		DeletedAtLamport: lamport,
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	a.publishPayloadAsync(topic, payload, "COMMENT_DELETE")
	return nil
}

func (a *App) PublishPostUpvote(pubkey string, postID string) error {
	pubkey = strings.TrimSpace(pubkey)
	postID = strings.TrimSpace(postID)
	if pubkey == "" || postID == "" {
		return errors.New("pubkey and post id are required")
	}

	msg := IncomingMessage{
		Type:        "POST_UPVOTE",
		OpID:        generateOperationID(postID, pubkey, time.Now().UnixNano()),
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

	a.scheduleVoteStateBroadcast(pubkey, postID, "")
	return nil
}

func (a *App) PublishPostDownvote(pubkey string, postID string) error {
	pubkey = strings.TrimSpace(pubkey)
	postID = strings.TrimSpace(postID)
	if pubkey == "" || postID == "" {
		return errors.New("pubkey and post id are required")
	}

	msg := IncomingMessage{
		Type:        "POST_DOWNVOTE",
		OpID:        generateOperationID(postID, pubkey, time.Now().UnixNano()),
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

	a.scheduleVoteStateBroadcast(pubkey, postID, "")
	return nil
}

func (a *App) PublishCommentUpvote(pubkey string, postID string, commentID string) error {
	pubkey = strings.TrimSpace(pubkey)
	postID = strings.TrimSpace(postID)
	commentID = strings.TrimSpace(commentID)
	if pubkey == "" || postID == "" || commentID == "" {
		return errors.New("pubkey, post id and comment id are required")
	}

	msg := IncomingMessage{
		Type:        "COMMENT_UPVOTE",
		OpID:        generateOperationID(commentID, pubkey, time.Now().UnixNano()),
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

	a.scheduleVoteStateBroadcast(pubkey, postID, commentID)
	return nil
}

func (a *App) PublishCommentDownvote(pubkey string, postID string, commentID string) error {
	pubkey = strings.TrimSpace(pubkey)
	postID = strings.TrimSpace(postID)
	commentID = strings.TrimSpace(commentID)
	if pubkey == "" || postID == "" || commentID == "" {
		return errors.New("pubkey, post id and comment id are required")
	}

	msg := IncomingMessage{
		Type:        "COMMENT_DOWNVOTE",
		OpID:        generateOperationID(commentID, pubkey, time.Now().UnixNano()),
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

	a.scheduleVoteStateBroadcast(pubkey, postID, commentID)
	return nil
}

func (a *App) publishFavoriteOperation(record FavoriteOpRecord) error {
	record.OpID = strings.TrimSpace(record.OpID)
	record.Pubkey = strings.TrimSpace(record.Pubkey)
	record.PostID = strings.TrimSpace(record.PostID)
	record.Signature = strings.TrimSpace(record.Signature)

	normalizedOp, err := normalizeFavoriteOperation(record.Op)
	if err != nil {
		return err
	}
	record.Op = normalizedOp

	if record.OpID == "" || record.Pubkey == "" || record.PostID == "" || record.CreatedAt <= 0 || record.Signature == "" {
		return errors.New("invalid favorite operation payload")
	}

	msg := IncomingMessage{
		Type:         messageTypeFavoriteOp,
		Pubkey:       record.Pubkey,
		PostID:       record.PostID,
		FavoriteOpID: record.OpID,
		FavoriteOp:   record.Op,
		Timestamp:    record.CreatedAt,
		Signature:    record.Signature,
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	a.p2pMu.Lock()
	topic := a.p2pTopic
	a.p2pMu.Unlock()
	a.publishPayloadAsync(topic, payload, "FAVORITE_OP")
	return nil
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

func (a *App) publishLocalProfileUpdateLocked() {
	if a.p2pTopic == nil || a.p2pCtx == nil {
		return
	}

	identity, err := a.getLocalIdentity()
	if err != nil {
		return
	}
	pubkey := strings.TrimSpace(identity.PublicKey)
	if pubkey == "" {
		return
	}

	profile, err := a.GetProfile(pubkey)
	if err != nil {
		profile = Profile{Pubkey: pubkey}
	}

	msg := IncomingMessage{
		Type:        "PROFILE_UPDATE",
		Pubkey:      pubkey,
		DisplayName: strings.TrimSpace(profile.DisplayName),
		AvatarURL:   strings.TrimSpace(profile.AvatarURL),
		Timestamp:   time.Now().Unix(),
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return
	}

	_ = a.p2pTopic.Publish(a.p2pCtx, payload)
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
	lamport, err := a.nextLamport()
	if err != nil {
		return err
	}
	msg := IncomingMessage{
		Type:         action,
		TargetPubkey: strings.TrimSpace(targetPubkey),
		AdminPubkey:  strings.TrimSpace(adminPubkey),
		Timestamp:    now,
		Lamport:      lamport,
		Reason:       strings.TrimSpace(reason),
	}

	if err = a.upsertModeration(msg.TargetPubkey, action, msg.AdminPubkey, now, lamport, msg.Reason); err != nil {
		return err
	}

	payload, err := json.Marshal(msg)
	if err != nil {
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
	startedAt := time.Now()
	a.noteContentFetchAttempt()

	_, err, shared := a.contentFetchGroup.Do("cid:"+contentCID, func() (any, error) {
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
		if a.ctx != nil {
			runtime.LogInfof(
				a.ctx,
				"content_fetch.request request_id=%s cid=%s peer_count=%d timeout_ms=%d retry_budget=%d",
				requestID,
				contentCID,
				len(host.Network().Peers()),
				timeout.Milliseconds(),
				resolveFetchRetryAttempts()-1,
			)
		}

		payload, marshalErr := json.Marshal(request)
		if marshalErr != nil {
			return nil, marshalErr
		}

		if publishErr := topic.Publish(ctx, payload); publishErr != nil {
			return nil, publishErr
		}

		deadline := time.Now().Add(timeout)
		notFoundCount := 0
		peerCount := len(host.Network().Peers())
		for {
			remaining := time.Until(deadline)
			if remaining <= 0 {
				if notFoundCount > 0 {
					return nil, errContentFetchNotFound
				}
				return nil, errContentFetchTimeout
			}

			select {
			case response := <-responseCh:
				if !response.Found {
					notFoundCount += 1
					if peerCount > 0 && notFoundCount >= peerCount {
						return nil, errContentFetchNotFound
					}
					continue
				}
				if upsertErr := a.upsertContentBlob(response.ContentCID, response.Content, response.SizeBytes); upsertErr != nil {
					return nil, upsertErr
				}
				return nil, nil
			case <-time.After(remaining):
				if notFoundCount > 0 {
					return nil, errContentFetchNotFound
				}
				return nil, errContentFetchTimeout
			}
		}
	})
	elapsedMs := time.Since(startedAt).Milliseconds()
	if err != nil {
		a.noteContentFetchResult(false, time.Since(startedAt))
		if a.ctx != nil {
			runtime.LogWarningf(
				a.ctx,
				"content_fetch.result cid=%s success=false elapsed_ms=%d dedup_shared=%t error=%v",
				contentCID,
				elapsedMs,
				shared,
				err,
			)
		}
		return err
	}
	a.noteContentFetchResult(true, time.Since(startedAt))
	if a.ctx != nil {
		runtime.LogInfof(
			a.ctx,
			"content_fetch.result cid=%s success=true elapsed_ms=%d dedup_shared=%t",
			contentCID,
			elapsedMs,
			shared,
		)
	}

	return nil
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
		if a.ctx != nil {
			runtime.LogInfof(
				a.ctx,
				"media_fetch.request request_id=%s cid=%s peer_count=%d timeout_ms=%d retry_budget=%d",
				requestID,
				contentCID,
				len(host.Network().Peers()),
				timeout.Milliseconds(),
				resolveFetchRetryAttempts()-1,
			)
		}

		payload, marshalErr := json.Marshal(request)
		if marshalErr != nil {
			return nil, marshalErr
		}

		if publishErr := topic.Publish(ctx, payload); publishErr != nil {
			return nil, publishErr
		}

		deadline := time.Now().Add(timeout)
		notFoundCount := 0
		peerCount := len(host.Network().Peers())
		for {
			remaining := time.Until(deadline)
			if remaining <= 0 {
				if notFoundCount > 0 {
					return nil, errMediaFetchNotFound
				}
				return nil, errMediaFetchTimeout
			}

			select {
			case response := <-responseCh:
				if !response.Found {
					notFoundCount += 1
					if peerCount > 0 && notFoundCount >= peerCount {
						return nil, errMediaFetchNotFound
					}
					continue
				}
				raw, decodeErr := base64.StdEncoding.DecodeString(strings.TrimSpace(response.ImageDataBase64))
				if decodeErr != nil || len(raw) == 0 {
					return nil, errors.New("invalid media response payload")
				}
				if upsertErr := a.upsertMediaBlobRaw(response.ContentCID, response.ImageMIME, raw, response.ImageWidth, response.ImageHeight, response.IsThumbnail); upsertErr != nil {
					return nil, upsertErr
				}
				return nil, nil
			case <-time.After(remaining):
				if notFoundCount > 0 {
					return nil, errMediaFetchNotFound
				}
				return nil, errMediaFetchTimeout
			}
		}
	})
	if a.ctx != nil {
		if err != nil {
			runtime.LogWarningf(a.ctx, "media_fetch.result cid=%s success=false error=%v", contentCID, err)
		} else {
			runtime.LogInfof(a.ctx, "media_fetch.result cid=%s success=true", contentCID)
		}
	}

	return err
}

func (a *App) TriggerAntiEntropySyncNow() error {
	return a.publishSyncSummaryRequest()
}

func (a *App) TriggerCommentSyncNow(postID string) error {
	return a.publishCommentSyncRequest(postID)
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
			if err := a.publishCommentSyncRequest(""); err != nil &&
				!errors.Is(err, errCommentSyncNoPeers) &&
				!strings.Contains(strings.ToLower(err.Error()), "p2p not started") &&
				a.ctx != nil {
				runtime.LogWarningf(a.ctx, "comment sync initial request failed: %v", err)
			}
			if err := a.publishGovernanceSyncRequest(); err != nil &&
				!errors.Is(err, errGovernanceSyncNoPeers) &&
				!strings.Contains(strings.ToLower(err.Error()), "p2p not started") &&
				a.ctx != nil {
				runtime.LogWarningf(a.ctx, "governance sync initial request failed: %v", err)
			}
			if err := a.publishFavoriteSyncRequest(); err != nil &&
				!errors.Is(err, errFavoriteSyncNoPeers) &&
				!strings.Contains(strings.ToLower(err.Error()), "p2p not started") &&
				!strings.Contains(strings.ToLower(err.Error()), "identity not found") &&
				a.ctx != nil {
				runtime.LogWarningf(a.ctx, "favorite sync initial request failed: %v", err)
			}
		case <-ticker.C:
			if err := a.publishSyncSummaryRequest(); err != nil &&
				!errors.Is(err, errAntiEntropyNoPeers) &&
				!strings.Contains(strings.ToLower(err.Error()), "p2p not started") &&
				a.ctx != nil {
				runtime.LogWarningf(a.ctx, "anti-entropy periodic sync failed: %v", err)
			}
			if err := a.publishCommentSyncRequest(""); err != nil &&
				!errors.Is(err, errCommentSyncNoPeers) &&
				!strings.Contains(strings.ToLower(err.Error()), "p2p not started") &&
				a.ctx != nil {
				runtime.LogWarningf(a.ctx, "comment sync periodic request failed: %v", err)
			}
			if err := a.publishGovernanceSyncRequest(); err != nil &&
				!errors.Is(err, errGovernanceSyncNoPeers) &&
				!strings.Contains(strings.ToLower(err.Error()), "p2p not started") &&
				a.ctx != nil {
				runtime.LogWarningf(a.ctx, "governance sync periodic request failed: %v", err)
			}
			if err := a.publishFavoriteSyncRequest(); err != nil &&
				!errors.Is(err, errFavoriteSyncNoPeers) &&
				!strings.Contains(strings.ToLower(err.Error()), "p2p not started") &&
				!strings.Contains(strings.ToLower(err.Error()), "identity not found") &&
				a.ctx != nil {
				runtime.LogWarningf(a.ctx, "favorite sync periodic request failed: %v", err)
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

func resolveGovernanceSyncBatchSize() int {
	raw := strings.TrimSpace(os.Getenv("AEGIS_GOVERNANCE_SYNC_BATCH_SIZE"))
	if raw != "" {
		if size, err := strconv.Atoi(raw); err == nil && size > 0 && size <= 500 {
			return size
		}
	}

	return 200
}

func resolveGovernanceLogSyncLimit() int {
	raw := strings.TrimSpace(os.Getenv("AEGIS_GOVERNANCE_LOG_SYNC_LIMIT"))
	if raw != "" {
		if size, err := strconv.Atoi(raw); err == nil && size > 0 && size <= 500 {
			return size
		}
	}

	return 200
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

func (a *App) publishCommentSyncRequest(postID string) error {
	a.p2pMu.Lock()
	topic := a.p2pTopic
	ctx := a.p2pCtx
	host := a.p2pHost
	a.p2pMu.Unlock()

	if topic == nil || ctx == nil || host == nil {
		return errors.New("p2p not started")
	}
	if len(host.Network().Peers()) == 0 {
		return errCommentSyncNoPeers
	}

	latestTimestamp, err := a.getLatestPublicCommentTimestamp()
	if err != nil {
		return err
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
		Type:             messageTypeCommentSyncRequest,
		RequestID:        buildMessageID(host.ID().String(), "comment-sync", time.Now().UnixNano()),
		RequesterPeerID:  host.ID().String(),
		PostID:           strings.TrimSpace(postID),
		CommentSinceTs:   sinceTimestamp,
		CommentBatchSize: batchSize,
		Timestamp:        time.Now().Unix(),
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return err
	}

	if a.ctx != nil {
		runtime.LogInfof(a.ctx, "comment_sync.request sent request_id=%s post_id=%s since=%d batch=%d", request.RequestID, request.PostID, request.CommentSinceTs, request.CommentBatchSize)
	}

	return topic.Publish(ctx, payload)
}

func (a *App) publishGovernanceSyncRequest() error {
	a.p2pMu.Lock()
	topic := a.p2pTopic
	ctx := a.p2pCtx
	host := a.p2pHost
	a.p2pMu.Unlock()

	if topic == nil || ctx == nil || host == nil {
		return errors.New("p2p not started")
	}
	if len(host.Network().Peers()) == 0 {
		return errGovernanceSyncNoPeers
	}

	latestTimestamp, err := a.getLatestModerationTimestamp()
	if err != nil {
		return err
	}
	latestLogTimestamp, err := a.getLatestAppliedModerationLogTimestamp()
	if err != nil {
		return err
	}

	request := IncomingMessage{
		Type:                 messageTypeGovernanceSyncRequest,
		RequestID:            buildMessageID(host.ID().String(), "governance-sync", time.Now().UnixNano()),
		RequesterPeerID:      host.ID().String(),
		GovernanceSinceTs:    latestTimestamp,
		GovernanceBatchSize:  resolveGovernanceSyncBatchSize(),
		GovernanceLogSinceTs: latestLogTimestamp,
		GovernanceLogLimit:   resolveGovernanceLogSyncLimit(),
		Timestamp:            time.Now().Unix(),
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return err
	}
	if a.ctx != nil {
		runtime.LogInfof(
			a.ctx,
			"governance_sync.request sent request_id=%s state_since=%d state_batch=%d log_since=%d log_limit=%d",
			request.RequestID,
			request.GovernanceSinceTs,
			request.GovernanceBatchSize,
			request.GovernanceLogSinceTs,
			request.GovernanceLogLimit,
		)
	}

	return topic.Publish(ctx, payload)
}

func (a *App) publishFavoriteSyncRequest() error {
	a.p2pMu.Lock()
	topic := a.p2pTopic
	ctx := a.p2pCtx
	host := a.p2pHost
	a.p2pMu.Unlock()

	if topic == nil || ctx == nil || host == nil {
		return errors.New("p2p not started")
	}
	if len(host.Network().Peers()) == 0 {
		return errFavoriteSyncNoPeers
	}

	identity, err := a.getLocalIdentity()
	if err != nil {
		return err
	}
	pubkey := strings.TrimSpace(identity.PublicKey)
	if pubkey == "" {
		return nil
	}

	latestTimestamp, err := a.getLatestFavoriteOpTimestamp(pubkey)
	if err != nil {
		return err
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
		Type:              messageTypeFavoriteSyncRequest,
		RequestID:         buildMessageID(host.ID().String(), "favorite-sync", time.Now().UnixNano()),
		RequesterPeerID:   host.ID().String(),
		Pubkey:            pubkey,
		FavoriteSinceTs:   sinceTimestamp,
		FavoriteBatchSize: batchSize,
		Timestamp:         time.Now().Unix(),
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return err
	}

	if a.ctx != nil {
		runtime.LogInfof(a.ctx, "favorite_sync.request sent request_id=%s pubkey=%s since=%d batch=%d", request.RequestID, request.Pubkey, request.FavoriteSinceTs, request.FavoriteBatchSize)
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
	listenRaw := a.p2pHost.Network().ListenAddresses()
	status.ListenAddrs = make([]string, 0, len(listenRaw))
	for _, addr := range listenRaw {
		status.ListenAddrs = append(status.ListenAddrs, fmt.Sprintf("%s/p2p/%s", addr.String(), a.p2pHost.ID().String()))
	}
	announced := a.p2pHost.Addrs()
	status.AnnounceAddrs = make([]string, 0, len(announced))
	for _, addr := range announced {
		status.AnnounceAddrs = append(status.AnnounceAddrs, fmt.Sprintf("%s/p2p/%s", addr.String(), a.p2pHost.ID().String()))
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
		if !a.allowIncomingMessage(remotePeerID, len(message.Data)) {
			a.p2pMu.Lock()
			host := a.p2pHost
			a.p2pMu.Unlock()
			if host != nil {
				_ = host.Network().ClosePeer(message.ReceivedFrom)
			}
			continue
		}
		if a.p2pHost != nil {
			if info := a.p2pHost.Peerstore().PeerInfo(message.ReceivedFrom); strings.TrimSpace(info.ID.String()) != "" {
				a.rememberConnectedPeer(info, true)
			}
		}
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
		case messageTypePeerExchangeRequest:
			a.handlePeerExchangeRequest(localPeerID.String(), incoming)
			continue
		case messageTypePeerExchangeResponse:
			a.handlePeerExchangeResponse(localPeerID.String(), incoming)
			continue
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
		case messageTypeCommentSyncRequest:
			a.handleCommentSyncRequest(localPeerID.String(), incoming)
			continue
		case messageTypeCommentSyncResponse:
			a.handleCommentSyncResponse(localPeerID.String(), incoming)
			continue
		case messageTypeGovernanceSyncRequest:
			a.handleGovernanceSyncRequest(localPeerID.String(), incoming)
			continue
		case messageTypeGovernanceSyncResponse:
			a.handleGovernanceSyncResponse(localPeerID.String(), incoming)
			continue
		case messageTypeFavoriteSyncRequest:
			a.handleFavoriteSyncRequest(localPeerID.String(), incoming)
			continue
		case messageTypeFavoriteSyncResponse:
			a.handleFavoriteSyncResponse(localPeerID.String(), incoming)
			continue
		}

		if err = a.ProcessIncomingMessage(message.Data); err != nil {
			if a.ctx != nil {
				runtime.LogErrorf(a.ctx, "process p2p message failed: %v", err)
			}
			continue
		}

		if a.ctx != nil {
			if messageType == messageTypeFavoriteOp {
				continue
			}
			if messageType == "SUB_CREATE" {
				runtime.EventsEmit(a.ctx, "subs:updated")
				continue
			}
			if messageType == "COMMENT" || messageType == "COMMENT_UPVOTE" || messageType == "COMMENT_DELETE" {
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

	shareable, shareErr := a.canServeContentBlobToNetwork(contentCID)
	if shareErr != nil || !shareable {
		if a.ctx != nil && shareErr == nil {
			runtime.LogInfof(a.ctx, "content_fetch.policy_block request_id=%s cid=%s requester=%s", requestID, contentCID, requester)
		}
		a.publishContentFetchNotFound(localPeerID, requester, requestID, contentCID)
		return
	}

	body, err := a.getContentBlobLocal(contentCID)
	if err != nil {
		a.publishContentFetchNotFound(localPeerID, requester, requestID, contentCID)
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

func (a *App) publishContentFetchNotFound(localPeerID string, requester string, requestID string, contentCID string) {
	a.p2pMu.Lock()
	topic := a.p2pTopic
	ctx := a.p2pCtx
	a.p2pMu.Unlock()
	if topic == nil || ctx == nil {
		return
	}

	response := IncomingMessage{
		Type:            messageTypeContentFetchResponse,
		RequestID:       requestID,
		RequesterPeerID: requester,
		ResponderPeerID: localPeerID,
		ContentCID:      contentCID,
		Found:           false,
		Timestamp:       time.Now().Unix(),
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

	shareable, shareErr := a.canServeMediaBlobToNetwork(contentCID)
	if shareErr != nil || !shareable {
		if a.ctx != nil && shareErr == nil {
			runtime.LogInfof(a.ctx, "media_fetch.policy_block request_id=%s cid=%s requester=%s", requestID, contentCID, requester)
		}
		a.publishMediaFetchNotFound(localPeerID, requester, requestID, contentCID)
		return
	}

	media, raw, err := a.getMediaBlobRawLocal(contentCID)
	if err != nil {
		a.publishMediaFetchNotFound(localPeerID, requester, requestID, contentCID)
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

func (a *App) publishMediaFetchNotFound(localPeerID string, requester string, requestID string, contentCID string) {
	a.p2pMu.Lock()
	topic := a.p2pTopic
	ctx := a.p2pCtx
	a.p2pMu.Unlock()
	if topic == nil || ctx == nil {
		return
	}

	response := IncomingMessage{
		Type:            messageTypeMediaFetchResponse,
		RequestID:       requestID,
		RequesterPeerID: requester,
		ResponderPeerID: localPeerID,
		ContentCID:      contentCID,
		Found:           false,
		Timestamp:       time.Now().Unix(),
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

func resolveIncomingMessageRateLimitPerWindow() int {
	raw := strings.TrimSpace(os.Getenv("AEGIS_MSG_RATE_LIMIT"))
	if raw == "" {
		return 240
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 240
	}
	return value
}

func resolveIncomingMessageRateWindowSeconds() int64 {
	raw := strings.TrimSpace(os.Getenv("AEGIS_MSG_RATE_WINDOW_SEC"))
	if raw == "" {
		return 60
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 60
	}
	return value
}

func resolveMaxIncomingMessageBytes() int {
	raw := strings.TrimSpace(os.Getenv("AEGIS_MSG_MAX_BYTES"))
	if raw == "" {
		return 2 * 1024 * 1024
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 2 * 1024 * 1024
	}
	return value
}

func resolveRelayServiceEnabled() bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv("AEGIS_RELAY_SERVICE_ENABLED")))
	if raw == "" {
		return true
	}
	return !(raw == "0" || raw == "false" || raw == "no" || raw == "off")
}

func (a *App) allowIncomingMessage(peerID string, payloadSize int) bool {
	peerID = strings.TrimSpace(peerID)
	if peerID == "" {
		return false
	}

	maxBytes := resolveMaxIncomingMessageBytes()
	if payloadSize > maxBytes {
		if a.ctx != nil {
			runtime.LogWarningf(a.ctx, "incoming message too large peer=%s size=%d max=%d", peerID, payloadSize, maxBytes)
		}
		a.markPeerGreylisted(peerID, "incoming-message-too-large")
		return false
	}

	limit := resolveIncomingMessageRateLimitPerWindow()
	windowSec := resolveIncomingMessageRateWindowSeconds()
	now := time.Now().Unix()
	stateKey := "msg:" + peerID

	a.fetchRateMu.Lock()
	defer a.fetchRateMu.Unlock()

	window, exists := a.fetchRateState[stateKey]
	if !exists || now-window.StartedAt >= windowSec {
		a.fetchRateState[stateKey] = fetchRateWindow{StartedAt: now, Count: 1}
		return true
	}

	if window.Count >= limit {
		if a.ctx != nil {
			runtime.LogWarningf(a.ctx, "incoming message rate limited peer=%s count=%d limit=%d window_sec=%d", peerID, window.Count, limit, windowSec)
		}
		a.markPeerGreylisted(peerID, "incoming-message-rate-limit")
		return false
	}

	window.Count += 1
	a.fetchRateState[stateKey] = window
	return true
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
		SchemaVersion:      lamportSchemaV2,
		AuthScope:          authScopeUser,
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
		if strings.TrimSpace(item.ID) == "" || strings.TrimSpace(item.Pubkey) == "" {
			continue
		}
		if !item.Deleted && strings.TrimSpace(item.ContentCID) == "" {
			continue
		}
		summaries = append(summaries, item)
	}

	sort.SliceStable(summaries, func(i int, j int) bool {
		left := LamportVersion{Lamport: summaries[i].Lamport, Author: summaries[i].Pubkey, OpID: summaries[i].OpID}
		right := LamportVersion{Lamport: summaries[j].Lamport, Author: summaries[j].Pubkey, OpID: summaries[j].OpID}
		return compareLamportVersion(left, right) > 0
	})

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
	viewerPubkey := ""
	if identity, err := a.getLocalIdentity(); err == nil {
		viewerPubkey = strings.TrimSpace(identity.PublicKey)
	}

	for _, digest := range summaries {
		allowed := true
		if !(digest.Deleted || normalizeOperationType(digest.OpType, postOpTypeCreate) == postOpTypeDelete) {
			allowResult, allowErr := a.shouldAcceptPublicContent(digest.Pubkey, digest.Lamport, digest.Timestamp, digest.ID, viewerPubkey)
			if allowErr != nil || !allowResult {
				continue
			}
		}
		if !allowed {
			continue
		}

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

		if !digest.Deleted && bodyFetchBudget > 0 {
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

func (a *App) handleCommentSyncRequest(localPeerID string, message IncomingMessage) {
	requester := strings.TrimSpace(message.RequesterPeerID)
	requestID := strings.TrimSpace(message.RequestID)
	if requester == "" || requestID == "" || requester == localPeerID {
		return
	}

	batchSize := message.CommentBatchSize
	if batchSize <= 0 || batchSize > 500 {
		batchSize = resolveAntiEntropyBatchSize()
	}

	sinceTimestamp := message.CommentSinceTs
	if sinceTimestamp < 0 {
		sinceTimestamp = 0
	}

	requestPostID := strings.TrimSpace(message.PostID)
	var comments []SyncCommentDigest
	var err error
	if requestPostID == "" {
		comments, err = a.listPublicCommentDigestsSince(sinceTimestamp, batchSize)
	} else {
		comments, err = a.listPublicCommentDigestsByPostSince(requestPostID, sinceTimestamp, batchSize)
	}
	if err != nil {
		if a.ctx != nil {
			runtime.LogWarningf(a.ctx, "comment_sync.build failed request_id=%s err=%v", requestID, err)
		}
		return
	}

	response := IncomingMessage{
		Type:             messageTypeCommentSyncResponse,
		SchemaVersion:    lamportSchemaV2,
		AuthScope:        authScopeUser,
		RequestID:        requestID,
		RequesterPeerID:  requester,
		ResponderPeerID:  localPeerID,
		PostID:           requestPostID,
		CommentSinceTs:   sinceTimestamp,
		CommentBatchSize: batchSize,
		CommentSummaries: comments,
		Timestamp:        time.Now().Unix(),
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

	if a.ctx != nil {
		runtime.LogInfof(a.ctx, "comment_sync.response sent request_id=%s post_id=%s since=%d batch=%d comments=%d", requestID, requestPostID, sinceTimestamp, batchSize, len(comments))
	}

	_ = topic.Publish(ctx, payload)
}

func (a *App) handleCommentSyncResponse(localPeerID string, message IncomingMessage) {
	requester := strings.TrimSpace(message.RequesterPeerID)
	if requester == "" || requester != localPeerID {
		return
	}

	if len(message.CommentSummaries) == 0 {
		return
	}

	commentSummaries := make([]SyncCommentDigest, 0, len(message.CommentSummaries))
	for _, item := range message.CommentSummaries {
		if strings.TrimSpace(item.ID) == "" || strings.TrimSpace(item.PostID) == "" || strings.TrimSpace(item.Pubkey) == "" {
			continue
		}
		if !item.Deleted && strings.TrimSpace(item.Body) == "" && len(item.Attachments) == 0 {
			continue
		}
		commentSummaries = append(commentSummaries, item)
	}

	if len(commentSummaries) == 0 {
		return
	}

	sort.SliceStable(commentSummaries, func(i int, j int) bool {
		left := LamportVersion{Lamport: commentSummaries[i].Lamport, Author: commentSummaries[i].Pubkey, OpID: commentSummaries[i].OpID}
		right := LamportVersion{Lamport: commentSummaries[j].Lamport, Author: commentSummaries[j].Pubkey, OpID: commentSummaries[j].OpID}
		return compareLamportVersion(left, right) > 0
	})

	inserted := 0
	updatedProfiles := 0
	updatedPostIDs := make(map[string]struct{})
	viewerPubkey := ""
	if identity, err := a.getLocalIdentity(); err == nil {
		viewerPubkey = strings.TrimSpace(identity.PublicKey)
	}
	for _, digest := range commentSummaries {
		if !(digest.Deleted || normalizeOperationType(digest.OpType, postOpTypeCreate) == postOpTypeDelete) {
			allowed, allowErr := a.shouldAcceptPublicContent(digest.Pubkey, digest.Lamport, digest.Timestamp, digest.ID, viewerPubkey)
			if allowErr != nil || !allowed {
				continue
			}
		}

		if strings.TrimSpace(digest.DisplayName) != "" || strings.TrimSpace(digest.AvatarURL) != "" {
			if _, err := a.upsertProfile(digest.Pubkey, digest.DisplayName, digest.AvatarURL, digest.Timestamp); err == nil {
				updatedProfiles++
			}
		}

		if digest.Deleted || normalizeOperationType(digest.OpType, postOpTypeCreate) == postOpTypeDelete {
			deleteLamport := digest.Lamport
			if digest.DeletedAtLamport > deleteLamport {
				deleteLamport = digest.DeletedAtLamport
			}
			if err := a.upsertCommentTombstone(digest.ID, digest.PostID, digest.Pubkey, digest.Timestamp, deleteLamport, digest.OpID); err != nil {
				continue
			}
		} else {
			if _, err := a.insertComment(Comment{
				ID:          digest.ID,
				PostID:      digest.PostID,
				ParentID:    digest.ParentID,
				Pubkey:      digest.Pubkey,
				OpID:        digest.OpID,
				Body:        digest.Body,
				Attachments: digest.Attachments,
				Score:       digest.Score,
				Timestamp:   digest.Timestamp,
				Lamport:     digest.Lamport,
			}); err != nil {
				continue
			}
		}

		inserted++
		updatedPostIDs[strings.TrimSpace(digest.PostID)] = struct{}{}
	}

	if a.ctx != nil {
		runtime.LogInfof(a.ctx, "comment_sync.response applied request_id=%s post_id=%s received=%d inserted=%d profiles=%d", strings.TrimSpace(message.RequestID), strings.TrimSpace(message.PostID), len(commentSummaries), inserted, updatedProfiles)
	}

	if inserted == 0 && updatedProfiles == 0 {
		return
	}

	if a.ctx != nil {
		for postID := range updatedPostIDs {
			if postID == "" {
				continue
			}
			runtime.EventsEmit(a.ctx, "comments:updated", map[string]string{"postId": postID})
		}
		runtime.EventsEmit(a.ctx, "feed:updated")
	}
}

func (a *App) handleGovernanceSyncRequest(localPeerID string, message IncomingMessage) {
	requester := strings.TrimSpace(message.RequesterPeerID)
	requestID := strings.TrimSpace(message.RequestID)
	if requester == "" || requestID == "" || requester == localPeerID {
		return
	}

	sinceTs := message.GovernanceSinceTs
	if sinceTs < 0 {
		sinceTs = 0
	}

	batchSize := message.GovernanceBatchSize
	if batchSize <= 0 || batchSize > 500 {
		batchSize = resolveGovernanceSyncBatchSize()
	}
	logSinceTs := message.GovernanceLogSinceTs
	if logSinceTs < 0 {
		logSinceTs = 0
	}
	logLimit := message.GovernanceLogLimit
	if logLimit <= 0 || logLimit > 500 {
		logLimit = resolveGovernanceLogSyncLimit()
	}

	states, err := a.listModerationSince(sinceTs, batchSize)
	if err != nil {
		if a.ctx != nil {
			runtime.LogWarningf(a.ctx, "governance_sync.build failed request_id=%s err=%v", requestID, err)
		}
		return
	}
	logs, err := a.listAppliedModerationLogsSince(logSinceTs, logLimit)
	if err != nil {
		if a.ctx != nil {
			runtime.LogWarningf(a.ctx, "governance_sync.build_logs failed request_id=%s err=%v", requestID, err)
		}
		return
	}

	response := IncomingMessage{
		Type:                 messageTypeGovernanceSyncResponse,
		RequestID:            requestID,
		RequesterPeerID:      requester,
		ResponderPeerID:      localPeerID,
		GovernanceSinceTs:    sinceTs,
		GovernanceBatchSize:  batchSize,
		GovernanceLogSinceTs: logSinceTs,
		GovernanceLogLimit:   logLimit,
		GovernanceStates:     states,
		GovernanceLogs:       logs,
		Timestamp:            time.Now().Unix(),
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

	if a.ctx != nil {
		runtime.LogInfof(a.ctx, "governance_sync.response sent request_id=%s state_since=%d states=%d log_since=%d logs=%d", requestID, sinceTs, len(states), logSinceTs, len(logs))
	}
}

func (a *App) handleGovernanceSyncResponse(localPeerID string, message IncomingMessage) {
	requester := strings.TrimSpace(message.RequesterPeerID)
	if requester == "" || requester != localPeerID {
		return
	}

	states := make([]ModerationState, 0, len(message.GovernanceStates))
	for _, row := range message.GovernanceStates {
		if strings.TrimSpace(row.TargetPubkey) == "" || strings.TrimSpace(row.SourceAdmin) == "" {
			continue
		}
		action := strings.ToUpper(strings.TrimSpace(row.Action))
		if action != "SHADOW_BAN" && action != "UNBAN" {
			continue
		}
		row.Action = action
		states = append(states, row)
	}
	if len(states) == 0 {
		// logs may still be present even when state delta is empty.
	}

	sort.SliceStable(states, func(i int, j int) bool {
		left := states[i]
		right := states[j]
		if left.Lamport > 0 && right.Lamport > 0 && left.Lamport != right.Lamport {
			return left.Lamport < right.Lamport
		}
		if left.Timestamp != right.Timestamp {
			return left.Timestamp < right.Timestamp
		}
		if left.TargetPubkey != right.TargetPubkey {
			return left.TargetPubkey < right.TargetPubkey
		}
		return left.Action < right.Action
	})

	applied := 0
	for _, row := range states {
		trusted, err := a.isTrustedAdmin(row.SourceAdmin)
		if err != nil || !trusted {
			if a.ctx != nil {
				runtime.LogWarningf(a.ctx, "governance_sync.skip_untrusted target=%s admin=%s action=%s", row.TargetPubkey, row.SourceAdmin, row.Action)
			}
			continue
		}
		if err := a.upsertModeration(row.TargetPubkey, row.Action, row.SourceAdmin, row.Timestamp, row.Lamport, row.Reason); err != nil {
			if a.ctx != nil {
				runtime.LogWarningf(a.ctx, "governance_sync.apply_failed target=%s action=%s err=%v", row.TargetPubkey, row.Action, err)
			}
			continue
		}
		applied++
	}

	logsInserted := 0
	for _, row := range message.GovernanceLogs {
		if strings.TrimSpace(row.TargetPubkey) == "" || strings.TrimSpace(row.SourceAdmin) == "" {
			continue
		}
		action := strings.ToUpper(strings.TrimSpace(row.Action))
		if action != "SHADOW_BAN" && action != "UNBAN" {
			continue
		}
		if strings.TrimSpace(strings.ToLower(row.Result)) != "applied" {
			continue
		}

		trusted, err := a.isTrustedAdmin(row.SourceAdmin)
		if err != nil || !trusted {
			if a.ctx != nil {
				runtime.LogWarningf(a.ctx, "governance_sync.skip_untrusted_log target=%s admin=%s action=%s", row.TargetPubkey, row.SourceAdmin, action)
			}
			continue
		}

		inserted, err := a.insertModerationLogIfAbsent(ModerationLog{
			TargetPubkey: row.TargetPubkey,
			Action:       action,
			SourceAdmin:  row.SourceAdmin,
			Timestamp:    row.Timestamp,
			Lamport:      row.Lamport,
			Reason:       row.Reason,
			Result:       "applied",
		})
		if err != nil {
			if a.ctx != nil {
				runtime.LogWarningf(a.ctx, "governance_sync.log_apply_failed target=%s action=%s err=%v", row.TargetPubkey, action, err)
			}
			continue
		}
		if inserted {
			logsInserted++
		}
	}

	if a.ctx != nil {
		runtime.LogInfof(a.ctx, "governance_sync.response applied request_id=%s states=%d applied=%d logs=%d logs_inserted=%d", strings.TrimSpace(message.RequestID), len(states), applied, len(message.GovernanceLogs), logsInserted)
		if applied > 0 || logsInserted > 0 {
			runtime.EventsEmit(a.ctx, "feed:updated")
		}
	}
}

func (a *App) handleFavoriteSyncRequest(localPeerID string, message IncomingMessage) {
	requester := strings.TrimSpace(message.RequesterPeerID)
	requestID := strings.TrimSpace(message.RequestID)
	pubkey := strings.TrimSpace(message.Pubkey)
	if requester == "" || requestID == "" || requester == localPeerID || pubkey == "" {
		return
	}

	identity, err := a.getLocalIdentity()
	if err != nil {
		return
	}
	localPubkey := strings.TrimSpace(identity.PublicKey)
	if localPubkey == "" || localPubkey != pubkey {
		return
	}

	sinceTs := message.FavoriteSinceTs
	if sinceTs < 0 {
		sinceTs = 0
	}
	batchSize := message.FavoriteBatchSize
	if batchSize <= 0 || batchSize > 500 {
		batchSize = resolveAntiEntropyBatchSize()
	}

	ops, err := a.listFavoriteOpsSince(pubkey, sinceTs, batchSize)
	if err != nil {
		if a.ctx != nil {
			runtime.LogWarningf(a.ctx, "favorite_sync.build failed request_id=%s err=%v", requestID, err)
		}
		return
	}

	response := IncomingMessage{
		Type:              messageTypeFavoriteSyncResponse,
		RequestID:         requestID,
		RequesterPeerID:   requester,
		ResponderPeerID:   localPeerID,
		Pubkey:            pubkey,
		FavoriteSinceTs:   sinceTs,
		FavoriteBatchSize: batchSize,
		FavoriteOps:       ops,
		Timestamp:         time.Now().Unix(),
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

	if a.ctx != nil {
		runtime.LogInfof(a.ctx, "favorite_sync.response sent request_id=%s pubkey=%s since=%d ops=%d", requestID, pubkey, sinceTs, len(ops))
	}
}

func (a *App) handleFavoriteSyncResponse(localPeerID string, message IncomingMessage) {
	requester := strings.TrimSpace(message.RequesterPeerID)
	if requester == "" || requester != localPeerID {
		return
	}

	identity, err := a.getLocalIdentity()
	if err != nil {
		return
	}
	localPubkey := strings.TrimSpace(identity.PublicKey)
	if localPubkey == "" {
		return
	}

	messagePubkey := strings.TrimSpace(message.Pubkey)
	if messagePubkey != "" && messagePubkey != localPubkey {
		return
	}

	ops := make([]FavoriteOpRecord, 0, len(message.FavoriteOps))
	for _, row := range message.FavoriteOps {
		record := FavoriteOpRecord{
			OpID:      strings.TrimSpace(row.OpID),
			Pubkey:    strings.TrimSpace(row.Pubkey),
			PostID:    strings.TrimSpace(row.PostID),
			Op:        strings.TrimSpace(row.Op),
			CreatedAt: row.CreatedAt,
			Signature: strings.TrimSpace(row.Signature),
		}
		if record.Pubkey == "" {
			record.Pubkey = messagePubkey
		}
		if record.Pubkey != localPubkey {
			continue
		}
		if record.OpID == "" || record.PostID == "" || record.CreatedAt <= 0 || record.Signature == "" {
			continue
		}
		if _, normalizeErr := normalizeFavoriteOperation(record.Op); normalizeErr != nil {
			continue
		}
		ops = append(ops, record)
	}

	sort.SliceStable(ops, func(i int, j int) bool {
		if ops[i].CreatedAt == ops[j].CreatedAt {
			return ops[i].OpID < ops[j].OpID
		}
		return ops[i].CreatedAt < ops[j].CreatedAt
	})

	applied := 0
	for _, row := range ops {
		changed, applyErr := a.applyFavoriteOperation(row, true)
		if applyErr != nil {
			if a.ctx != nil {
				runtime.LogWarningf(a.ctx, "favorite_sync.apply_failed op_id=%s post_id=%s err=%v", row.OpID, row.PostID, applyErr)
			}
			continue
		}
		if changed {
			applied++
		}
	}

	if a.ctx != nil {
		runtime.LogInfof(a.ctx, "favorite_sync.response applied request_id=%s received=%d applied=%d", strings.TrimSpace(message.RequestID), len(ops), applied)
	}
	if applied > 0 {
		a.emitFavoritesUpdated("")
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

	if n.app.p2pHost == nil || n.app.p2pCtx == nil {
		n.app.p2pMu.Unlock()
		return
	}

	peerID := strings.TrimSpace(info.ID.String())
	if blocked, reason := n.app.isPeerBlocked(peerID); blocked {
		n.app.p2pMu.Unlock()
		if n.app.ctx != nil {
			runtime.LogWarningf(n.app.ctx, "mdns peer rejected by policy peer=%s reason=%s", peerID, reason)
		}
		return
	}

	if n.app.p2pHost.Network().Connectedness(info.ID) != network.Connected && len(n.app.p2pHost.Network().Peers()) >= resolveMaxConnectedPeers() {
		n.app.p2pMu.Unlock()
		if n.app.ctx != nil {
			runtime.LogWarningf(n.app.ctx, "mdns peer rejected by max-peer limit peer=%s limit=%d", peerID, resolveMaxConnectedPeers())
		}
		return
	}

	ctx, cancel := context.WithTimeout(n.app.p2pCtx, 5*time.Second)
	defer cancel()

	if err := n.app.p2pHost.Connect(ctx, info); err != nil {
		n.app.p2pMu.Unlock()
		if n.app.ctx != nil {
			runtime.LogWarningf(n.app.ctx, "mdns connect failed: %v", err)
		}
		return
	}

	n.app.publishLocalProfileUpdateLocked()
	n.app.p2pMu.Unlock()

	if n.app.ctx != nil {
		runtime.LogInfof(n.app.ctx, "mdns connected peer: %s", info.ID.String())
		runtime.EventsEmit(n.app.ctx, "p2p:updated")
	}
}

func resolveRelayPeers() []string {
	fromEnv := parsePeerAddressesCSV(os.Getenv("AEGIS_RELAY_PEERS"))
	if len(fromEnv) > 0 {
		return fromEnv
	}
	fromBuild := parsePeerAddressesCSV(defaultRelayPeersCSV)
	if len(fromBuild) > 0 {
		return fromBuild
	}
	return []string{officialBootstrapRelayAddr}
}

func resolveP2PListenAddrs(listenPort int) []string {
	if listenPort <= 0 {
		listenPort = 40100
	}

	ipv4 := fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)
	ipv6 := fmt.Sprintf("/ip6/::/tcp/%d", listenPort)

	raw := strings.TrimSpace(strings.ToLower(os.Getenv("AEGIS_P2P_DUAL_STACK")))
	if raw == "0" || raw == "false" || raw == "no" || raw == "off" {
		return []string{ipv4}
	}

	return []string{ipv4, ipv6}
}

func resolveP2PAnnounceAddrs(listenPort int) []multiaddr.Multiaddr {
	if listenPort <= 0 {
		listenPort = 40100
	}

	raw := strings.TrimSpace(os.Getenv("AEGIS_ANNOUNCE_ADDRS"))
	if raw == "" {
		publicIP := strings.TrimSpace(os.Getenv("AEGIS_PUBLIC_IP"))
		if publicIP == "" && resolveAutoAnnounceEnabled() {
			publicIP = resolveAutoPublicIPv4()
		}
		if publicIP != "" {
			raw = fmt.Sprintf("/ip4/%s/tcp/%d", publicIP, listenPort)
		}
	}
	if raw == "" {
		return nil
	}

	parts := strings.FieldsFunc(raw, func(r rune) bool {
		switch r {
		case ',', ';', '\n', '\r':
			return true
		default:
			return false
		}
	})

	result := make([]multiaddr.Multiaddr, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		if idx := strings.Index(item, "/p2p/"); idx > 0 {
			item = strings.TrimSpace(item[:idx])
		}
		maddr, err := multiaddr.NewMultiaddr(item)
		if err != nil {
			continue
		}
		normalized := maddr.String()
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, maddr)
	}

	if len(result) == 0 {
		return nil
	}
	return result
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

func (a *App) PublishPostUpdate(pubkey string, postID string, title string, body string) error {
	pubkey = strings.TrimSpace(pubkey)
	postID = strings.TrimSpace(postID)
	if pubkey == "" || postID == "" {
		return errors.New("pubkey and post id are required")
	}
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	if title == "" && body == "" {
		return errors.New("title or body must be updated")
	}

	shadowBanned, err := a.isShadowBanned(pubkey)
	if err != nil {
		return err
	}
	if shadowBanned {
		_, err = a.UpdateLocalPost(pubkey, postID, title, body)
		return err
	}

	a.p2pMu.Lock()
	topic := a.p2pTopic
	a.p2pMu.Unlock()

	profile, profileErr := a.GetProfile(pubkey)
	if profileErr != nil {
		profile = Profile{}
	}

	localPost, err := a.UpdateLocalPost(pubkey, postID, title, body)
	if err != nil {
		return err
	}

	msg := IncomingMessage{
		Type:          "POST",
		OpType:        postOpTypeUpdate,
		OpID:          localPost.OpID,
		SchemaVersion: lamportSchemaV2,
		AuthScope:     authScopeUser,
		ID:            localPost.ID,
		Pubkey:        pubkey,
		DisplayName:   strings.TrimSpace(profile.DisplayName),
		AvatarURL:     strings.TrimSpace(profile.AvatarURL),
		Title:         localPost.Title,
		Body:          localPost.Body,
		ContentCID:    localPost.ContentCID,
		ImageCID:      localPost.ImageCID,
		ThumbCID:      localPost.ThumbCID,
		ImageMIME:     localPost.ImageMIME,
		ImageSize:     localPost.ImageSize,
		ImageWidth:    localPost.ImageWidth,
		ImageHeight:   localPost.ImageHeight,
		Content:       "",
		SubID:         normalizeSubID(localPost.SubID),
		Timestamp:     localPost.Timestamp,
		Lamport:       localPost.Lamport,
		Signature:     "",
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	a.publishPayloadAsync(topic, payload, "POST_UPDATE")
	return nil
}

func (a *App) PublishCommentUpdate(pubkey string, commentID string, body string) error {
	pubkey = strings.TrimSpace(pubkey)
	commentID = strings.TrimSpace(commentID)
	if pubkey == "" || commentID == "" {
		return errors.New("pubkey and comment id are required")
	}
	body = strings.TrimSpace(body)

	shadowBanned, err := a.isShadowBanned(pubkey)
	if err != nil {
		return err
	}
	if shadowBanned {
		_, err = a.UpdateLocalComment(pubkey, commentID, body)
		return err
	}

	a.p2pMu.Lock()
	topic := a.p2pTopic
	a.p2pMu.Unlock()

	profile, profileErr := a.GetProfile(pubkey)
	if profileErr != nil {
		profile = Profile{}
	}

	localComment, err := a.UpdateLocalComment(pubkey, commentID, body)
	if err != nil {
		return err
	}

	msg := IncomingMessage{
		Type:               "COMMENT",
		OpType:             postOpTypeUpdate,
		OpID:               localComment.OpID,
		SchemaVersion:      lamportSchemaV2,
		AuthScope:          authScopeUser,
		ID:                 localComment.ID,
		Pubkey:             pubkey,
		PostID:             localComment.PostID,
		ParentID:           localComment.ParentID,
		DisplayName:        strings.TrimSpace(profile.DisplayName),
		AvatarURL:          strings.TrimSpace(profile.AvatarURL),
		Body:               localComment.Body,
		CommentAttachments: localComment.Attachments,
		Timestamp:          localComment.Timestamp,
		Lamport:            localComment.Lamport,
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	a.publishPayloadAsync(topic, payload, "COMMENT_UPDATE")
	return nil
}
