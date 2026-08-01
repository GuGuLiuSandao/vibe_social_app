# Authentication Behavior

## AUTH-001 — JWT identity contract

- A generated token carries the requested user ID and username.
- `iat` and `nbf` represent token creation time at JWT second precision.
- `exp` is exactly seven days after `iat`.
- A malformed token, empty token, or token signed with another secret is rejected without returning claims.

## AUTH-HTTP-001 — Protobuf registration and login

- Register and login accept `application/x-protobuf` bodies and return protobuf responses.
- A successful registration persists the user and sets nickname to username.
- A subsequent login returns the same UID, username, email, and nickname and a non-empty JWT.
