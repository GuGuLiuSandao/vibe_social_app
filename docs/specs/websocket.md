# WebSocket Behavior

## WS-001 — One current connection per user

- Registering a second client for the same UID replaces the first without increasing the online count.
- Unregistering the stale client cannot remove the replacement.
- A snapshot contains the current clients and can be changed by its caller without changing manager state.

## WS-HTTP-001 — Authenticated protobuf ping

- Missing or invalid JWT is rejected with HTTP 401 before WebSocket upgrade.
- A valid JWT upgrades the connection and marks its UID online in Redis.
- A protobuf `WS_TYPE_PING` receives `WS_TYPE_PONG` with the same request ID.
- Closing the current connection removes the UID from the Redis online set.
