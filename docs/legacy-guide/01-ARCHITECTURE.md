# Architecture

Read this before writing any code. It defines the shape everything else slots into,
so later phases don't force a rewrite.

## High-level flow

```
                 ┌─────────────┐
   client  ───── │   Listener   │  (TCP or TLS)
                 └──────┬───────┘
                         │ accept()
                         ▼
                 ┌─────────────┐
                 │ handleConn   │  one goroutine per connection
                 └──────┬───────┘
                         │ parse HTTP request (manual)
                         ▼
                 ┌─────────────┐
                 │  Cache.Get   │  hit? → write cached response, done
                 └──────┬───────┘
                         │ miss
                         ▼
                 ┌─────────────┐
                 │  Balancer    │  pick a healthy backend
                 │  .Next()     │
                 └──────┬───────┘
                         │
                         ▼
                 ┌─────────────┐
                 │ dial backend │  forward request (rewritten)
                 └──────┬───────┘
                         │
                         ▼
                 ┌─────────────┐
                 │ read response│  manual parse
                 └──────┬───────┘
                         │
                ┌────────┴────────┐
                ▼                 ▼
         Cache.Set (if        write response
         cacheable)            to client
```

## Package layout

Don't over-engineer this with too many packages early — start flat, split later
if a file gets unwieldy. Suggested end-state:

```
goproxy/
├── cmd/
│   └── proxy/
│       └── main.go          # flag/config parsing, wires everything together, starts listener
├── internal/
│   ├── httpmsg/              # manual HTTP/1.1 request & response parsing/serialization
│   │   ├── request.go
│   │   └── response.go
│   ├── proxy/                 # connection handling — the orchestrator
│   │   └── proxy.go
│   ├── balancer/              # load-balancing strategies
│   │   ├── balancer.go        # interface
│   │   ├── roundrobin.go
│   │   └── leastconn.go
│   ├── backend/                # backend pool + health checks
│   │   ├── backend.go
│   │   └── healthcheck.go
│   ├── cache/                  # in-memory response cache
│   │   └── cache.go
│   └── config/
│       └── config.go           # config struct + loader (JSON or YAML, your call)
├── configs/
│   └── proxy.example.json
├── test/
│   └── ...                     # integration tests, fake backends
├── go.mod
└── README.md
```

This is a real Go layout (`cmd/` for entrypoints, `internal/` for code that isn't
meant to be imported by other projects). Using it from the start avoids a painful
restructure mid-project.

## Core types (sketch — refine as you build)

You will define these properly in their respective phase docs. This is just so
you can see how they relate to each other before you start.

```go
// httpmsg
type Request struct {
    Method  string
    Path    string
    Proto   string
    Headers http.Header // reusing the type is fine; reusing the parser is not
    Body    []byte      // or io.Reader, decide in Phase 2
}

type Response struct {
    StatusCode int
    Status     string
    Headers    http.Header
    Body       []byte
}

// backend
type Backend struct {
    Addr        string
    Alive       bool
    ActiveConns int32 // atomic
}

// balancer
type Balancer interface {
    Next(backends []*Backend) *Backend
}

// cache
type Cache interface {
    Get(key string) (*httpmsg.Response, bool)
    Set(key string, resp *httpmsg.Response, ttl time.Duration)
}
```

Note: it's fine to use `net/http.Header` as a map type for convenience — that's
just a `map[string][]string` with helper methods, not the part we're avoiding.
What you may not use is `net/http.ReadRequest`, `http.Server`, or
`httputil.ReverseProxy` — those would parse/proxy *for* you.

## Concurrency model

One goroutine per client connection (the standard Go pattern — don't build a
worker pool, it adds complexity without benefit at this scale). Shared mutable
state (the backend pool, the cache) needs synchronization:

- Backend `Alive` flags and `ActiveConns`: use `sync/atomic` or a small mutex.
- Cache: a `sync.RWMutex` guarding a map is enough — don't reach for
  `sync.Map` until you've measured a reason to.

## Data flow decisions to lock in before Phase 2

- **Body handling:** buffer small bodies into memory (`[]byte`) to start —
  simpler to reason about. Streaming large bodies via `io.Reader` is a
  legitimate hardening step for Phase 6, not something to design for upfront.
- **Connection lifecycle:** Phase 2 will start with "one request per connection,
  then close" before adding keep-alive. Don't try to support keep-alive on the
  first pass — get correctness first.
