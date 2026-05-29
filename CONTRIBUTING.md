# Contributing to Aegis

## Development Setup

### Prerequisites

- Go 1.24+
- Node.js 18+ (for frontend)
- [Wails v2](https://wails.io/docs/gettingstarted/installation) (for desktop builds)

### Quick Start

```bash
cd aegis-app

# Install frontend dependencies
cd frontend && npm install && cd ..

# Run tests
go test ./...

# Development mode (hot reload)
wails dev
```

### Build

```bash
wails build
```

## Project Structure

```
aegis-app/
├── main.go              # Desktop app entry point
├── main_relay.go        # Relay node entry point (build tag: relay)
├── app.go               # App struct, identity, startup/shutdown
├── p2p.go               # libp2p networking, pubsub, sync
├── p2p_peers.go         # Peer management, known peers
├── p2p_config.go        # P2P configuration
├── p2p_public_ip.go     # Public IP detection
├── db.go                # Core database operations
├── db_schema.go         # Schema migrations
├── db_posts.go          # Post CRUD and feed queries
├── db_comments.go       # Comment operations
├── db_favorites.go      # Favorites with Lamport ordering
├── db_moderation.go     # Moderation state and logs
├── db_subs.go           # Sub-community management
├── community_ops.go     # Sub settings, pin/lock operations
├── authenticated_messages.go  # Message signing and verification
├── types.go             # All data model types
├── constants.go         # Shared constants
├── notifications.go     # Notification system
├── message_outbox.go    # Outbox for offline-first publishing
├── alerting.go          # Release alerting system
├── observability.go     # Metrics and observability
├── updates.go           # Version/update management
├── recommendation.go    # Feed recommendation strategies
├── logging.go           # Logging utilities
├── frontend/            # React + TypeScript frontend
│   ├── src/
│   │   ├── App.tsx      # Main application component
│   │   ├── components/  # UI components
│   │   ├── lib/         # Utility libraries
│   │   └── types/       # TypeScript type definitions
│   └── ...
└── tests/               # Test documentation
```

## Testing

### Run All Tests

```bash
go test -timeout 300s ./...
```

### Run Specific Tests

```bash
# Unit tests only (fast)
go test -run "TestIdentity|TestSign|TestCreateSub|TestPublishPost|TestAddComment" ./...

# P2P integration tests (slower, requires network)
go test -run "TestA2|TestA3|TestB1|TestC1|TestG7" -timeout 180s ./...

# Lamport consistency tests
go test -run "TestLamport|TestPostDelete|TestEqualLamport|TestDigest" ./...
```

### Test Coverage

```bash
go test -coverprofile=coverage.out -timeout 300s ./...
go tool cover -html=coverage.out  # View in browser
go tool cover -func=coverage.out  # Summary by function
```

### Frontend Linting

```bash
cd frontend
npm run lint        # Check for issues
npm run lint:fix    # Auto-fix what's possible
```

## Code Style

### Go

- Standard `gofmt` formatting
- `go vet` must pass with no warnings
- Error handling: always check and handle errors
- Use `dbMu` mutex for all database write operations
- Use `p2pMu` mutex for all P2P state access
- Lamport clock operations must be atomic with their database writes

### TypeScript/React

- ESLint with `@typescript-eslint` and `react-hooks` plugins
- Functional components with hooks
- Tailwind CSS for styling
- Wails runtime bindings in `wailsjs/` (auto-generated)

## Architecture Decisions

### Offline-First

All operations (post, comment, vote, moderate) write to local SQLite first, then queue for network broadcast via the message outbox. This ensures the app works without connectivity.

### Lamport Ordering

Governance operations (ban/unban, pin/lock) use Lamport clocks for deterministic conflict resolution across nodes. When two conflicting operations arrive, the one with the higher Lamport value wins. Equal Lamport values are resolved by OpID comparison.

### Signed Messages

All user-generated content is signed with the author's ed25519 key derived from their BIP-39 mnemonic. This prevents forgery and enables trustless replication.

### Anti-Entropy Sync

Nodes periodically exchange content summaries and request missing items. This ensures eventual consistency even when pubsub messages are lost.

## Common Tasks

### Adding a New API Method

1. Add the method to `app.go` or the relevant `db_*.go` file
2. The method will be automatically exposed to the frontend via Wails bindings
3. Run `wails dev` to regenerate TypeScript bindings in `frontend/wailsjs/`
4. Use the binding in the frontend

### Adding a New Message Type

1. Define the type constant in `constants.go`
2. Add handling in `ProcessIncomingMessage` (in `db.go`)
3. Add signature support in `authenticated_messages.go`
4. Add publish method in `p2p.go`
5. Add tests

### Database Schema Changes

Schema changes use an additive migration strategy in `db_schema.go`. Never modify existing columns; only add new ones or new tables.
