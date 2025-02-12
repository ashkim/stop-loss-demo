# Stop-Loss Order System

This project implements a stop-loss order system with real-time order updates.


# Usage

## Quick Start
```bash
# Start all services
make up
```

Then visit http://localhost:3000

## Development Commands

### Testing
```bash
# Run all tests
make test

# Run linter
make lint

```

### Monitor the Price Simulator
```bash
# Connect to the price stream WebSocket (requires wscat)
make wscat-prices
```

### Database Operations
```bash
# Open SQLite CLI for direct database access
make sql

# View all orders in the database
make dump-orders
```

### Cleanup
```bash
# Stop all services and clean up volumes
make clean
```

## Prerequisites
- Docker and docker-compose

### optional:
- wscat (`npm install -g wscat`) for WebSocket testing
- sqlite3 command-line tool
- golangci-lint for linting

## Notes
- `make clean` will remove the SQLite database file to ensure sync with Temporal
- The database file is located at `./services/stop-loss/data/orders.db`
- WebSocket price stream is available at `ws://localhost:8081/prices`
