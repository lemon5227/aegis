package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

const (
	knownPeerBootstrapLimit = 64
	peerExchangeSendLimit   = 24
	peerExchangeTickSeconds = 90
)

func (a *App) getKnownPeerBootstrapAddresses(limit int) []string {
	if a.db == nil {
		return nil
	}
	if limit <= 0 || limit > 256 {
		limit = knownPeerBootstrapLimit
	}

	rows, err := a.db.Query(`
		SELECT peer_id, addrs_json
		FROM known_peers
		ORDER BY public_reachable DESC, relay_capable DESC, (success_count - fail_count) DESC, last_seen DESC
		LIMIT ?;
	`, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()

	result := make([]string, 0, limit)
	for rows.Next() {
		var (
			peerID    string
			addrsJSON string
		)
		if scanErr := rows.Scan(&peerID, &addrsJSON); scanErr != nil {
			continue
		}
		for _, addr := range decodeKnownPeerAddrs(addrsJSON) {
			result = append(result, addr)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return normalizePeerAddresses(result)
}

func (a *App) upsertKnownPeer(entry KnownPeerExchange, successDelta int, failDelta int) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	peerID := strings.TrimSpace(entry.PeerID)
	if peerID == "" {
		return errors.New("peer id is required")
	}
	if entry.LastSeen <= 0 {
		entry.LastSeen = time.Now().Unix()
	}

	addrs := normalizeKnownPeerAddrs(peerID, entry.Addrs)
	addrsJSONBytes, err := json.Marshal(addrs)
	if err != nil {
		return err
	}

	relayCapable := 0
	if entry.RelayCapable {
		relayCapable = 1
	}
	publicReachable := 0
	if entry.PublicReachable {
		publicReachable = 1
	}

	_, err = a.db.Exec(`
		INSERT INTO known_peers (
			peer_id, addrs_json, last_seen, success_count, fail_count, relay_capable, public_reachable, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(peer_id) DO UPDATE SET
			addrs_json = CASE
				WHEN LENGTH(COALESCE(excluded.addrs_json, '')) > LENGTH(COALESCE(known_peers.addrs_json, '')) THEN excluded.addrs_json
				ELSE known_peers.addrs_json
			END,
			last_seen = CASE
				WHEN excluded.last_seen > known_peers.last_seen THEN excluded.last_seen
				ELSE known_peers.last_seen
			END,
			success_count = known_peers.success_count + ?,
			fail_count = known_peers.fail_count + ?,
			relay_capable = CASE WHEN excluded.relay_capable = 1 THEN 1 ELSE known_peers.relay_capable END,
			public_reachable = CASE WHEN excluded.public_reachable = 1 THEN 1 ELSE known_peers.public_reachable END,
			updated_at = excluded.updated_at;
	`, peerID, string(addrsJSONBytes), entry.LastSeen, maxInt(successDelta, 0), maxInt(failDelta, 0), relayCapable, publicReachable, time.Now().Unix(), maxInt(successDelta, 0), maxInt(failDelta, 0))
	return err
}

func (a *App) rememberConnectedPeer(info peer.AddrInfo, connectSuccess bool) {
	entry := KnownPeerExchange{
		PeerID:   strings.TrimSpace(info.ID.String()),
		Addrs:    make([]string, 0, len(info.Addrs)),
		LastSeen: time.Now().Unix(),
	}
	for _, addr := range info.Addrs {
		entry.Addrs = append(entry.Addrs, fmt.Sprintf("%s/p2p/%s", addr.String(), info.ID.String()))
	}
	if connectSuccess {
		_ = a.upsertKnownPeer(entry, 1, 0)
	} else {
		_ = a.upsertKnownPeer(entry, 0, 1)
	}
}

func (a *App) buildKnownPeerExchangePayload(limit int) []KnownPeerExchange {
	if a.db == nil {
		return nil
	}
	if limit <= 0 || limit > 64 {
		limit = peerExchangeSendLimit
	}

	payload := make([]KnownPeerExchange, 0, limit+1)

	a.p2pMu.Lock()
	host := a.p2pHost
	a.p2pMu.Unlock()
	if host != nil {
		self := KnownPeerExchange{
			PeerID:          host.ID().String(),
			Addrs:           make([]string, 0, len(host.Addrs())),
			RelayCapable:    resolveRelayCandidateEnabled(),
			PublicReachable: len(host.Addrs()) > 0,
			LastSeen:        time.Now().Unix(),
		}
		for _, addr := range host.Addrs() {
			self.Addrs = append(self.Addrs, fmt.Sprintf("%s/p2p/%s", addr.String(), host.ID().String()))
		}
		payload = append(payload, self)
	}

	rows, err := a.db.Query(`
		SELECT peer_id, addrs_json, relay_capable, public_reachable, last_seen
		FROM known_peers
		ORDER BY public_reachable DESC, relay_capable DESC, (success_count - fail_count) DESC, last_seen DESC
		LIMIT ?;
	`, limit)
	if err != nil {
		return payload
	}
	defer rows.Close()

	for rows.Next() {
		var (
			peerID          string
			addrsJSON       string
			relayCapableInt int
			publicInt       int
			lastSeen        int64
		)
		if scanErr := rows.Scan(&peerID, &addrsJSON, &relayCapableInt, &publicInt, &lastSeen); scanErr != nil {
			continue
		}
		payload = append(payload, KnownPeerExchange{
			PeerID:          strings.TrimSpace(peerID),
			Addrs:           decodeKnownPeerAddrs(addrsJSON),
			RelayCapable:    relayCapableInt == 1,
			PublicReachable: publicInt == 1,
			LastSeen:        lastSeen,
		})
	}

	if len(payload) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	filtered := make([]KnownPeerExchange, 0, len(payload))
	for _, item := range payload {
		id := strings.TrimSpace(item.PeerID)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		item.Addrs = normalizeKnownPeerAddrs(id, item.Addrs)
		filtered = append(filtered, item)
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].PublicReachable != filtered[j].PublicReachable {
			return filtered[i].PublicReachable
		}
		if filtered[i].RelayCapable != filtered[j].RelayCapable {
			return filtered[i].RelayCapable
		}
		return filtered[i].LastSeen > filtered[j].LastSeen
	})

	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered
}

func (a *App) ingestKnownPeers(entries []KnownPeerExchange) {
	if len(entries) == 0 {
		return
	}
	for _, item := range entries {
		_ = a.upsertKnownPeer(item, 0, 0)
	}
}

func (a *App) connectToKnownPeers(limit int) {
	a.p2pMu.Lock()
	host := a.p2pHost
	a.p2pMu.Unlock()
	if host == nil {
		return
	}

	candidates := a.getKnownPeerBootstrapAddresses(limit)
	if len(candidates) == 0 {
		return
	}

	for _, addr := range candidates {
		maddr, err := multiaddr.NewMultiaddr(strings.TrimSpace(addr))
		if err != nil {
			continue
		}
		info, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			continue
		}
		if info.ID == host.ID() {
			continue
		}
		a.p2pMu.Lock()
		_ = a.connectPeerLocked(addr)
		a.p2pMu.Unlock()
	}
}

func (a *App) runPeerExchangeWorker(ctx context.Context, localPeerID peer.ID) {
	ticker := time.NewTicker(peerExchangeTickSeconds * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.requestPeerExchange(localPeerID)
		}
	}
}

func (a *App) requestPeerExchange(localPeerID peer.ID) {
	a.p2pMu.Lock()
	host := a.p2pHost
	topic := a.p2pTopic
	ctx := a.p2pCtx
	a.p2pMu.Unlock()

	if host == nil || topic == nil || ctx == nil {
		return
	}
	peers := host.Network().Peers()
	if len(peers) == 0 {
		return
	}

	target := peers[time.Now().UnixNano()%int64(len(peers))]
	request := IncomingMessage{
		Type:            messageTypePeerExchangeRequest,
		RequestID:       buildMessageID(localPeerID.String(), fmt.Sprintf("peer-exchange|%d", time.Now().UnixNano()), time.Now().Unix()),
		RequesterPeerID: localPeerID.String(),
		ResponderPeerID: target.String(),
		Timestamp:       time.Now().Unix(),
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return
	}
	_ = topic.Publish(ctx, payload)
}

func (a *App) handlePeerExchangeRequest(localPeerID string, message IncomingMessage) {
	if strings.TrimSpace(message.ResponderPeerID) != strings.TrimSpace(localPeerID) {
		return
	}
	requester := strings.TrimSpace(message.RequesterPeerID)
	if requester == "" {
		return
	}

	payloadPeers := a.buildKnownPeerExchangePayload(peerExchangeSendLimit)
	response := IncomingMessage{
		Type:            messageTypePeerExchangeResponse,
		RequestID:       strings.TrimSpace(message.RequestID),
		RequesterPeerID: requester,
		ResponderPeerID: strings.TrimSpace(localPeerID),
		KnownPeers:      payloadPeers,
		Timestamp:       time.Now().Unix(),
	}

	a.p2pMu.Lock()
	topic := a.p2pTopic
	ctx := a.p2pCtx
	a.p2pMu.Unlock()
	if topic == nil || ctx == nil {
		return
	}
	payload, err := json.Marshal(response)
	if err != nil {
		return
	}
	_ = topic.Publish(ctx, payload)
}

func (a *App) handlePeerExchangeResponse(localPeerID string, message IncomingMessage) {
	if strings.TrimSpace(message.RequesterPeerID) != strings.TrimSpace(localPeerID) {
		return
	}
	a.ingestKnownPeers(message.KnownPeers)
	a.connectToKnownPeers(12)
}

func decodeKnownPeerAddrs(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var addrs []string
	if err := json.Unmarshal([]byte(raw), &addrs); err != nil {
		return nil
	}
	return normalizePeerAddresses(addrs)
}

func normalizeKnownPeerAddrs(peerID string, addrs []string) []string {
	peerID = strings.TrimSpace(peerID)
	if peerID == "" {
		return nil
	}
	result := make([]string, 0, len(addrs))
	for _, addr := range normalizePeerAddresses(addrs) {
		item := strings.TrimSpace(addr)
		if item == "" {
			continue
		}
		if !strings.Contains(item, "/p2p/") {
			item = strings.TrimSuffix(item, "/") + "/p2p/" + peerID
		}
		result = append(result, item)
	}
	return normalizePeerAddresses(result)
}

func resolveRelayCandidateEnabled() bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv("AEGIS_RELAY_CANDIDATE")))
	if raw == "" {
		return true
	}
	return !(raw == "0" || raw == "false" || raw == "no" || raw == "off")
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
