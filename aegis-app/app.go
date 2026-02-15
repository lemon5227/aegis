package main

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/tyler-smith/go-bip39"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

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

	return &App{dbPath: databasePath}
}

func (a *App) SetDatabasePath(path string) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return
	}
	a.dbPath = trimmed
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

	if !shouldAutoStartP2P() {
		return
	}

	listenPort := resolveAutoStartP2PPort()
	bootstrapPeers := resolveBootstrapPeers()
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
	raw := strings.TrimSpace(os.Getenv("AEGIS_BOOTSTRAP_PEERS"))
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

	if len(peers) == 0 {
		return nil
	}

	return peers
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
