# Client Identity and WebSocket Builders

## CLIENT-001 — UID parsing and whitelist

- `parseUid` accepts positive decimal digit text and positive `bigint` values.
- Whitespace around text is removed; leading zeroes in digit text are preserved.
- Zero, negative, fractional, empty, and non-digit values are rejected.
- The whitelist interval is inclusive from `10000000` through `20000000`.

## CLIENT-002 — WebSocket request builders

- The configured WebSocket base is a bare endpoint without an existing query string.
- UID is added as the first query parameter; a present token is URL encoded.
- Ping and auth protobuf builders preserve caller request IDs, message types, and payload variants.
