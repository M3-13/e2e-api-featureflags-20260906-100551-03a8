# featureflags

A feature-flag service as a REST API in Go, built exclusively with the
standard library's `net/http`. Flags are managed in a thread-safe in-memory
store; an evaluation endpoint returns a deterministic per-user on/off decision
based on a stable hash against the rollout percentage.

## Tech stack

- **Language**: Go 1.22+
- **Framework**: `net/http` (standard library only, no external web framework)
- **Storage**: in-memory with `sync.RWMutex`
- **Tests**: `httptest` (Go standard library)

## Install

```sh
go mod download
```

## Run (development)

```sh
go run .
```

The server listens on the `PORT` environment variable (default `8080`).

## How to use

The service exposes a JSON REST API. Every JSON response sets
`Content-Type: application/json; charset=utf-8`. Errors use the shape
`{"error":"string"}`.

| Method   | Path                      | Description                                   | Success |
|----------|---------------------------|-----------------------------------------------|---------|
| `POST`   | `/flags`                  | Create a flag                                 | 201     |
| `GET`    | `/flags`                  | List all flags (empty store → `[]`)           | 200     |
| `GET`    | `/flags/{key}`            | Get a single flag                             | 200     |
| `PUT`    | `/flags/{key}`            | Update `enabled`, `description`, `rollout_percent` | 200 |
| `DELETE` | `/flags/{key}`            | Delete a flag                                 | 204     |
| `GET`    | `/flags/{key}/evaluate?user={id}` | Evaluate a flag for a user            | 200     |
| `GET`    | `/healthz`                | Health check                                  | 200     |

### Flag object

```json
{
  "key": "my-flag",
  "enabled": true,
  "description": "some flag",
  "rollout_percent": 50
}
```

### Examples

Create a flag:

```sh
curl -X POST http://localhost:8080/flags \
  -H "Content-Type: application/json" \
  -d '{"key":"my-flag","enabled":true,"rollout_percent":50}'
```

Health check:

```sh
curl http://localhost:8080/healthz
# {"status":"ok"}
```

## Features

- Thread-safe in-memory flag store
- CRUD endpoints for flags
- Deterministic per-user rollout evaluation
- JSON error objects for 400/404/405/409/413
- Health check endpoint
