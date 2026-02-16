package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

func parsePeerAddressesCSV(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	normalized := strings.NewReplacer("\n", ",", "\r", ",", ";", ",").Replace(raw)
	parts := strings.Split(normalized, ",")
	return normalizePeerAddresses(parts)
}

func normalizePeerAddresses(candidates []string) []string {
	if len(candidates) == 0 {
		return nil
	}

	result := make([]string, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		peerAddr := strings.TrimSpace(candidate)
		if peerAddr == "" {
			continue
		}
		if _, exists := seen[peerAddr]; exists {
			continue
		}
		seen[peerAddr] = struct{}{}
		result = append(result, peerAddr)
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

func mergePeerAddressLists(lists ...[]string) []string {
	merged := make([]string, 0)
	for _, list := range lists {
		merged = append(merged, list...)
	}
	return normalizePeerAddresses(merged)
}

func normalizeP2PListenPort(port int) (int, error) {
	if port <= 0 {
		return 40100, nil
	}
	if port > 65535 {
		return 0, fmt.Errorf("invalid listen port: %d", port)
	}
	return port, nil
}

func (a *App) getStoredP2PConfig() (*P2PConfig, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	var (
		listenPort int
		relayJSON  string
		autoStart  int
		updatedAt  int64
	)
	err := a.db.QueryRow(`
		SELECT listen_port, relay_peers_json, auto_start, updated_at
		FROM p2p_config
		WHERE id = 1;
	`).Scan(&listenPort, &relayJSON, &autoStart, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var relayPeers []string
	if strings.TrimSpace(relayJSON) != "" {
		if unmarshalErr := json.Unmarshal([]byte(relayJSON), &relayPeers); unmarshalErr != nil {
			return nil, unmarshalErr
		}
	}

	listenPort, err = normalizeP2PListenPort(listenPort)
	if err != nil {
		listenPort = resolveAutoStartP2PPort()
	}

	return &P2PConfig{
		ListenPort: listenPort,
		RelayPeers: normalizePeerAddresses(relayPeers),
		AutoStart:  autoStart == 1,
		UpdatedAt:  updatedAt,
	}, nil
}

func (a *App) defaultP2PConfig() P2PConfig {
	return P2PConfig{
		ListenPort: resolveAutoStartP2PPort(),
		RelayPeers: mergePeerAddressLists(resolveBootstrapPeers(), resolveRelayPeers()),
		AutoStart:  shouldAutoStartP2P(),
		UpdatedAt:  0,
	}
}

func (a *App) GetP2PConfig() (P2PConfig, error) {
	if a.db == nil {
		return P2PConfig{}, errors.New("database not initialized")
	}

	defaults := a.defaultP2PConfig()
	stored, err := a.getStoredP2PConfig()
	if err != nil {
		return P2PConfig{}, err
	}
	if stored == nil {
		return defaults, nil
	}

	cfg := *stored
	if cfg.ListenPort <= 0 {
		cfg.ListenPort = defaults.ListenPort
	}
	if len(cfg.RelayPeers) == 0 {
		cfg.RelayPeers = defaults.RelayPeers
	}

	return cfg, nil
}

func (a *App) SaveP2PConfig(listenPort int, relayPeers []string, autoStart bool) (P2PConfig, error) {
	if a.db == nil {
		return P2PConfig{}, errors.New("database not initialized")
	}

	normalizedPort, err := normalizeP2PListenPort(listenPort)
	if err != nil {
		return P2PConfig{}, err
	}
	normalizedPeers := normalizePeerAddresses(relayPeers)
	updatedAt := time.Now().Unix()

	relayJSONBytes, err := json.Marshal(normalizedPeers)
	if err != nil {
		return P2PConfig{}, err
	}

	autoStartInt := 0
	if autoStart {
		autoStartInt = 1
	}

	_, err = a.db.Exec(`
		INSERT INTO p2p_config (id, listen_port, relay_peers_json, auto_start, updated_at)
		VALUES (1, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			listen_port = excluded.listen_port,
			relay_peers_json = excluded.relay_peers_json,
			auto_start = excluded.auto_start,
			updated_at = excluded.updated_at;
	`, normalizedPort, string(relayJSONBytes), autoStartInt, updatedAt)
	if err != nil {
		return P2PConfig{}, err
	}

	return P2PConfig{
		ListenPort: normalizedPort,
		RelayPeers: normalizedPeers,
		AutoStart:  autoStart,
		UpdatedAt:  updatedAt,
	}, nil
}

func (a *App) resolveAutoStartP2PSettings() (bool, int, []string) {
	defaults := a.defaultP2PConfig()
	if a.db == nil {
		return defaults.AutoStart, defaults.ListenPort, defaults.RelayPeers
	}

	cfg, err := a.GetP2PConfig()
	if err != nil {
		return defaults.AutoStart, defaults.ListenPort, defaults.RelayPeers
	}

	bootstrapPeers := mergePeerAddressLists(cfg.RelayPeers, resolveBootstrapPeers())
	return cfg.AutoStart, cfg.ListenPort, bootstrapPeers
}
