# Relay Cold-Start to Self-Sustaining Network Plan

Date: 2026-02-20

## Goal

Use one low-cost public relay for cold start now, then gradually let public-reachable community nodes take over relay/bootstrap duties so the official relay can be retired.

## Current Constraints

1. Only one public server is available.
2. Many end-user nodes are behind NAT and cannot accept inbound connections.
3. Relay must minimize bandwidth and CPU pressure while still enabling network join.

## Immediate Production Setup (Single Relay)

Use explicit announce addresses so peers can learn a public-reachable address instead of private addresses.

Required env on relay host:

- `AEGIS_BOOTSTRAP_PEERS` (optional seed list)
- `AEGIS_RELAY_PEERS` (optional static relay list)
- `AEGIS_PUBLIC_IP=<relay-public-ip>`
  - or `AEGIS_ANNOUNCE_ADDRS=/ip4/<relay-public-ip>/tcp/40100`

Runtime behavior:

1. Node listens on local interfaces (`listen_addrs`).
2. Node advertises public addresses (`announce_addrs`) when configured.
3. Other nodes use announced addresses for faster bootstrap.

## New Support Added

1. `P2PStatus` now includes `announceAddrs`.
2. Relay binary startup output now prints both:
   - `listen_addrs`
   - `announce_addrs`
3. New announce resolution:
   - `AEGIS_ANNOUNCE_ADDRS` (preferred, supports multiple multiaddrs)
   - fallback `AEGIS_PUBLIC_IP` + listen port.

## Suggested Deploy Command (systemd env example)

```bash
AEGIS_PUBLIC_IP=YOUR_PUBLIC_IP \
AEGIS_RELAY_PEERS="" \
AEGIS_BOOTSTRAP_PEERS="" \
./aegis-relay
```

Or explicit announce list:

```bash
AEGIS_ANNOUNCE_ADDRS="/ip4/YOUR_PUBLIC_IP/tcp/40100" ./aegis-relay
```

## Scale-Out Path (No Extra Budget First)

### Phase S1 - Known Peers Persistence

Each node stores successful peers locally:

- `peer_id`
- `addrs`
- `last_seen`
- `success_count`
- `fail_count`
- `relay_capable`

Startup order:

1. try known peers
2. fallback bootstrap relay

### Phase S2 - Peer Exchange

After handshake, peers exchange a small set of healthy known peers (top-N scored), reducing dependence on central relay for discovery.

### Phase S3 - Community Relay Candidates

Public-reachable nodes can opt into relay role and advertise that capability. Clients choose multiple relay candidates with scoring and load caps.

### Phase S4 - Official Relay Retirement

Retire official relay only when:

1. network has >= 3 stable community relay candidates,
2. new node join success remains high without official relay,
3. reconnect and anti-entropy continue to pass G6 checks.

## Operational Guardrails

1. Keep relay connection and rate limits strict.
2. Prefer direct peer connections when possible; relay is bootstrap/fallback.
3. Track join failures and relay load in release metrics before scaling down official infra.
