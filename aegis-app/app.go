package main

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"math"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/tyler-smith/go-bip39"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sync/singleflight"
	"time"
	"fmt"
)

var (
	defaultBootstrapPeersCSV string
	defaultRelayPeersCSV     string
)

const officialBootstrapRelayAddr = "/ip4/51.107.6.171/tcp/40100/p2p/12D3KooWAFxb45HZaK3rtSRAy6wTR7PN8nUaZrXgBNuBPJoNKqwg"

// App struct
type App struct {
	ctx    context.Context
	db     *sql.DB
	dbMu   sync.Mutex
	dbPath string

	p2pMu     sync.Mutex
	p2pCtx    context.Context
	p2pCancel context.CancelFunc
	p2pHost   host.Host
	p2pTopic  *pubsub.Topic
	p2pSub    *pubsub.Subscription
	mdnsSvc   io.Closer

	fetchRateMu    sync.Mutex
	fetchRateState map[string]fetchRateWindow
	peerPolicyMu   sync.Mutex
	peerBlacklist  map[string]struct{}
	peerGreylist   map[string]int64

	contentFetchGroup   singleflight.Group
	contentFetchWaiters map[string]chan IncomingMessage
	mediaFetchGroup     singleflight.Group
	mediaFetchWaiters   map[string]chan IncomingMessage
	postFetchGroup      singleflight.Group
	postFetchWaiters    map[string]chan IncomingMessage

	antiEntropyMu      sync.Mutex
	antiEntropyStats   AntiEntropyStats
	observabilityMu    sync.Mutex
	observabilityStats ObservabilityStats
	releaseAlertMu     sync.Mutex
	releaseAlertState  map[string]int64
	releaseAlertActive map[string]ReleaseAlert
	voteBroadcastMu    sync.Mutex
	voteBroadcastSeq   map[string]int64

	defaultRecStrategy string
}

type AntiEntropyStats struct {
	SyncRequestsSent       int64 `json:"syncRequestsSent"`
	SyncRequestsReceived   int64 `json:"syncRequestsReceived"`
	SyncResponsesReceived  int64 `json:"syncResponsesReceived"`
	SyncSummariesReceived  int64 `json:"syncSummariesReceived"`
	IndexInsertions        int64 `json:"indexInsertions"`
	BlobFetchAttempts      int64 `json:"blobFetchAttempts"`
	BlobFetchSuccess       int64 `json:"blobFetchSuccess"`
	BlobFetchFailures      int64 `json:"blobFetchFailures"`
	LastSyncAt             int64 `json:"lastSyncAt"`
	LastRemoteSummaryTs    int64 `json:"lastRemoteSummaryTs"`
	LastObservedSyncLagSec int64 `json:"lastObservedSyncLagSec"`
}

type Identity struct {
	Mnemonic  string `json:"mnemonic"`
	PublicKey string `json:"publicKey"`
}

// NewApp creates a new App application struct
func NewApp() *App {
	databasePath := strings.TrimSpace(os.Getenv("AEGIS_DB_PATH"))
	if databasePath == "" {
		databasePath = "aegis_node.db"
	}

	defaultStrategy := strings.TrimSpace(os.Getenv("AEGIS_DEFAULT_REC_STRATEGY"))
	if defaultStrategy == "" {
		defaultStrategy = "hot-v1"
	}

	return &App{
		dbPath:              databasePath,
		contentFetchWaiters: make(map[string]chan IncomingMessage),
		mediaFetchWaiters:   make(map[string]chan IncomingMessage),
		postFetchWaiters:    make(map[string]chan IncomingMessage),
		fetchRateState:      make(map[string]fetchRateWindow),
		peerBlacklist:       make(map[string]struct{}),
		peerGreylist:        make(map[string]int64),
		releaseAlertState:   make(map[string]int64),
		releaseAlertActive:  make(map[string]ReleaseAlert),
		voteBroadcastSeq:    make(map[string]int64),
		defaultRecStrategy:  defaultStrategy,
	}
}

func (a *App) SetDatabasePath(path string) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return
	}
	a.dbPath = trimmed
}

func (a *App) GetAntiEntropyStats() AntiEntropyStats {
	a.antiEntropyMu.Lock()
	defer a.antiEntropyMu.Unlock()
	return a.antiEntropyStats
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	if err := a.initDatabase(); err != nil {
		runtime.LogErrorf(ctx, "database initialization failed: %v", err)
		return
	}

	trustedAdminsEnv := strings.TrimSpace(os.Getenv("AEGIS_TRUSTED_ADMINS"))
	if trustedAdminsEnv != "" {
		for _, candidate := range strings.Split(trustedAdminsEnv, ",") {
			adminPubkey := strings.TrimSpace(candidate)
			if adminPubkey == "" {
				continue
			}
			if err := a.AddTrustedAdmin(adminPubkey, "genesis"); err != nil {
				runtime.LogErrorf(ctx, "seed trusted admin failed (%s): %v", adminPubkey, err)
			}
		}
	}

	autoStart, listenPort, bootstrapPeers := a.resolveAutoStartP2PSettings()
	if !autoStart {
		return
	}

	started := false
	for _, candidatePort := range resolveAutoStartPortCandidates(listenPort) {
		if !isTCPPortAvailable(candidatePort) {
			continue
		}

		if _, err := a.StartP2P(candidatePort, bootstrapPeers); err != nil {
			runtime.LogWarningf(ctx, "p2p auto start failed on port %d: %v", candidatePort, err)
			continue
		}

		started = true
		if candidatePort != listenPort {
			runtime.LogInfof(ctx, "p2p auto start fallback port selected: %d (preferred: %d)", candidatePort, listenPort)
		}
		break
	}

	if !started {
		runtime.LogErrorf(ctx, "p2p auto start failed: no available port in range [%d,%d]", listenPort, listenPort+20)
	}
}

func (a *App) shutdown(ctx context.Context) {
	if err := a.StopP2P(); err != nil {
		runtime.LogErrorf(ctx, "p2p shutdown failed: %v", err)
	}

	if a.db != nil {
		if err := a.db.Close(); err != nil {
			runtime.LogErrorf(ctx, "database close failed: %v", err)
		}
	}
}

func (a *App) deriveKeypairFromMnemonic(mnemonic string) (ed25519.PublicKey, ed25519.PrivateKey, error) {
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, nil, errors.New("invalid mnemonic")
	}

	seed := bip39.NewSeed(mnemonic, "")
	hash := sha256.Sum256(seed)
	privateKey := ed25519.NewKeyFromSeed(hash[:])
	publicKey := privateKey.Public().(ed25519.PublicKey)

	return publicKey, privateKey, nil
}

func (a *App) GenerateIdentity() (Identity, error) {
	entropy, err := bip39.NewEntropy(128)
	if err != nil {
		return Identity{}, err
	}

	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return Identity{}, err
	}

	publicKey, _, err := a.deriveKeypairFromMnemonic(mnemonic)
	if err != nil {
		return Identity{}, err
	}

	identity := Identity{
		Mnemonic:  mnemonic,
		PublicKey: hex.EncodeToString(publicKey),
	}

	if err = a.saveLocalIdentity(identity); err != nil {
		return Identity{}, err
	}

	return identity, nil
}

func (a *App) LoadSavedIdentity() (Identity, error) {
	return a.getLocalIdentity()
}

func (a *App) ImportIdentityFromMnemonic(mnemonic string) (Identity, error) {
	publicKey, _, err := a.deriveKeypairFromMnemonic(strings.TrimSpace(mnemonic))
	if err != nil {
		return Identity{}, err
	}

	identity := Identity{
		Mnemonic:  strings.TrimSpace(mnemonic),
		PublicKey: hex.EncodeToString(publicKey),
	}

	if err = a.saveLocalIdentity(identity); err != nil {
		return Identity{}, err
	}

	return identity, nil
}

func (a *App) SignMessage(mnemonic string, message string) (string, error) {
	_, privateKey, err := a.deriveKeypairFromMnemonic(mnemonic)
	if err != nil {
		return "", err
	}

	signature := ed25519.Sign(privateKey, []byte(message))
	return hex.EncodeToString(signature), nil
}

func (a *App) VerifyMessage(publicKeyHex string, message string, signatureHex string) (bool, error) {
	publicKey, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return false, err
	}

	signature, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false, err
	}

	if len(publicKey) != ed25519.PublicKeySize {
		return false, errors.New("invalid public key length")
	}

	if len(signature) != ed25519.SignatureSize {
		return false, errors.New("invalid signature length")
	}

	return ed25519.Verify(publicKey, []byte(message), signature), nil
}

func shouldAutoStartP2P() bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv("AEGIS_AUTOSTART_P2P")))
	if raw == "" {
		return true
	}

	switch raw {
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

func resolveAutoStartP2PPort() int {
	raw := strings.TrimSpace(os.Getenv("AEGIS_P2P_PORT"))
	if raw == "" {
		return 40100
	}

	port, err := strconv.Atoi(raw)
	if err != nil || port <= 0 {
		return 40100
	}

	return port
}

func resolveBootstrapPeers() []string {
	fromEnv := parsePeerAddressesCSV(os.Getenv("AEGIS_BOOTSTRAP_PEERS"))
	if len(fromEnv) > 0 {
		return fromEnv
	}
	fromBuild := parsePeerAddressesCSV(defaultBootstrapPeersCSV)
	if len(fromBuild) > 0 {
		return fromBuild
	}
	return []string{officialBootstrapRelayAddr}
}

func resolveAutoStartPortCandidates(preferredPort int) []int {
	if preferredPort <= 0 {
		preferredPort = 40100
	}

	result := make([]int, 0, 21)
	result = append(result, preferredPort)
	for offset := 1; offset <= 20; offset++ {
		result = append(result, preferredPort+offset)
	}

	return result
}

func isTCPPortAvailable(port int) bool {
	if port <= 0 {
		return false
	}

	listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return false
	}
	_ = listener.Close()
	return true
}

func (a *App) GetFeedStream(limit int) (FeedStream, error) {
	return a.GetFeedStreamWithStrategy(limit, a.defaultRecStrategy)
}

func (a *App) GetFeedStreamWithStrategy(limit int, algorithm string) (FeedStream, error) {
	if a.db == nil {
		return FeedStream{}, errors.New("database not initialized")
	}

	now := time.Now().Unix()
	limit = normalizeFeedStreamLimit(limit)
	algorithm = normalizeFeedStreamAlgorithm(algorithm)
	if algorithm == "" {
		algorithm = a.defaultRecStrategy
	}

	strategy, err := GetStrategy(algorithm)
	if err != nil {
		// Try fallback to hot-v1
		strategy, err = GetStrategy("hot-v1")
		if err != nil {
			return FeedStream{}, err
		}
		algorithm = "hot-v1"
	}

	viewerPubkey := ""
	if identity, err := a.getLocalIdentity(); err == nil {
		viewerPubkey = strings.TrimSpace(identity.PublicKey)
	}

	subscribedSubIDs, err := a.listSubscribedSubIDs()
	if err != nil {
		return FeedStream{}, err
	}

	subscribedQuota := int(math.Ceil(float64(limit) * 0.7))
	if subscribedQuota < 1 {
		subscribedQuota = 1
	}
	recommendedQuota := limit - subscribedQuota
	if recommendedQuota < 0 {
		recommendedQuota = 0
	}

	subscribedPosts := make([]ForumMessage, 0)
	if len(subscribedSubIDs) > 0 {
		subscribedPosts, err = a.queryPostsBySubSet(viewerPubkey, subscribedSubIDs, subscribedQuota*3)
		if err != nil {
			return FeedStream{}, err
		}
	}

	// Strategy ranks recommended posts
	// We fetch candidates first. Strategy interface might need to fetch itself or we fetch generic candidates?
	// For now, let's fetch a large pool of candidates and let the strategy sort them.
	// Current queryRecommendedPosts uses hot score logic inside SQL for ordering.
	// To be truly pluggable, we should fetch candidates (e.g. recent posts) and let strategy sort.
	// However, fetching ALL posts is expensive.
	// Compromise: Fetch recent posts (e.g. last 7 days) up to a larger limit, then let strategy rank.
	// Or, if strategy is just "hot-v1", use optimized SQL.
	// For N2 MVP, we can keep using SQL for candidates, but apply strategy logic for re-ranking/scoring.

	// Fetching raw candidates (latest 200 from non-subscribed subs)
	candidatePosts, err := a.queryRecommendedCandidates(viewerPubkey, subscribedSubIDs, max(limit*4, 100))
	if err != nil {
		return FeedStream{}, err
	}

	rankedRecommendations, err := strategy.Rank(candidatePosts, viewerPubkey, now)
	if err != nil {
		return FeedStream{}, err
	}

	items := make([]FeedStreamItem, 0, limit)
	seen := make(map[string]struct{}, limit)

	// Interleave subscribed and recommended
	// We re-rank subscribed posts using the same strategy?
	// Usually subscribed posts are chronological or also hot.
	// For now, keep subscribed posts as they come from DB (hot/time sorted),
	// and interleave with strategy-ranked recommendations.

	// Wrap subscribed posts
	subscribedItems := make([]FeedStreamItem, 0, len(subscribedPosts))
	for _, p := range subscribedPosts {
		// Use strategy to score them too for consistency if desired, or just use 0/timestamp
		// Let's rely on DB order for subscribed for now (hot)
		score := computeHotScore(p.Score, p.Timestamp, now)
		subscribedItems = append(subscribedItems, FeedStreamItem{
			Post:                p,
			Reason:              "subscribed",
			IsSubscribed:        true,
			RecommendationScore: score,
		})
	}

	si := 0
	ri := 0

	for len(items) < limit && (si < len(subscribedItems) || ri < len(rankedRecommendations)) {
		// Add 2 subscribed
		addedSub := 0
		for addedSub < 2 && si < len(subscribedItems) && len(items) < limit && countFeedItemsByReason(items, "subscribed") < subscribedQuota {
			item := subscribedItems[si]
			si++
			if _, exists := seen[item.Post.ID]; exists {
				continue
			}
			seen[item.Post.ID] = struct{}{}
			items = append(items, item)
			addedSub++
		}

		// Add 1 recommended
		for ri < len(rankedRecommendations) && len(items) < limit && countFeedItemsByReason(items, "recommended_" + algorithm) < recommendedQuota {
			item := rankedRecommendations[ri]
			ri++
			if _, exists := seen[item.Post.ID]; exists {
				continue
			}
			seen[item.Post.ID] = struct{}{}
			item.Reason = "recommended_" + algorithm // Override reason to include algo name
			items = append(items, item)
			break
		}

		// Fill remaining
		if si >= len(subscribedItems) && ri < len(rankedRecommendations) {
			for ri < len(rankedRecommendations) && len(items) < limit {
				item := rankedRecommendations[ri]
				ri++
				if _, exists := seen[item.Post.ID]; exists {
					continue
				}
				seen[item.Post.ID] = struct{}{}
				item.Reason = "recommended_" + algorithm
				items = append(items, item)
			}
		}

		if ri >= len(rankedRecommendations) && si < len(subscribedItems) {
			for si < len(subscribedItems) && len(items) < limit {
				item := subscribedItems[si]
				si++
				if _, exists := seen[item.Post.ID]; exists {
					continue
				}
				seen[item.Post.ID] = struct{}{}
				items = append(items, item)
			}
		}
	}

	return FeedStream{
		Items:       items,
		Algorithm:   algorithm,
		GeneratedAt: now,
	}, nil
}

func (a *App) queryRecommendedCandidates(viewerPubkey string, subscribedSubIDs []string, limit int) ([]ForumMessage, error) {
	if limit <= 0 {
		limit = 100
	}

	if len(subscribedSubIDs) == 0 {
		return a.queryForumMessages(`
			SELECT id, pubkey, title, body, content_cid, content, score, timestamp, size_bytes, zone, sub_id, is_protected, visibility
			FROM messages
			WHERE zone = 'public'
			  AND (visibility = 'normal' OR (pubkey = ? AND visibility != 'deleted'))
			ORDER BY timestamp DESC
			LIMIT ?;
		`, viewerPubkey, limit)
	}

	placeholders := makeSQLPlaceholders(len(subscribedSubIDs))
	args := make([]interface{}, 0, len(subscribedSubIDs)+2)
	args = append(args, viewerPubkey)
	for _, subID := range subscribedSubIDs {
		args = append(args, normalizeSubID(subID))
	}
	args = append(args, limit)

	query := fmt.Sprintf(`
		SELECT id, pubkey, title, body, content_cid, content, score, timestamp, size_bytes, zone, sub_id, is_protected, visibility
		FROM messages
		WHERE zone = 'public'
		  AND (visibility = 'normal' OR (pubkey = ? AND visibility != 'deleted'))
		  AND sub_id NOT IN (%s)
		ORDER BY timestamp DESC
		LIMIT ?;
	`, placeholders)

	return a.queryForumMessages(query, args...)
}
