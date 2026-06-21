# Phase 2 — Make It HTTP-Aware

## Goal

Stop blindly piping bytes. Parse HTTP/1.1 yourself, understand what a request
and response actually look like on the wire, and start mutating them the way a
real reverse proxy does.

## Concepts to understand before coding

- **Request line:** `METHOD SP REQUEST-TARGET SP HTTP-VERSION CRLF`
  e.g. `GET /index.html HTTP/1.1\r\n`
- **Headers:** `Name: value\r\n`, one per line, terminated by a bare `\r\n`
  (empty line) marking end of headers.
- **Body framing — this is the part people get wrong:**
  - If `Content-Length` is present, the body is exactly that many bytes.
  - If `Transfer-Encoding: chunked` is present, the body is a series of
    chunks, each prefixed by its size in hex, terminated by a `0\r\n\r\n`
    chunk. You must implement chunk decoding/encoding to handle this
    correctly — many naive proxies break here.
  - If neither is present (some GET/HEAD requests), there is no body.
  - You cannot have both `Content-Length` and `Transfer-Encoding` — if a
    request claims both, that's a request smuggling red flag; reject it.
- **Why proxies rewrite headers:**
  - `Host` — needs to reflect the backend, not the original request, in
    *some* setups (depends on your design — decide and document why).
  - `X-Forwarded-For` — append the client's IP so the backend knows who the
    real client was (proxies obscure this otherwise).
  - `X-Forwarded-Proto` — tells backend if original request was http or https
    (matters once Phase 5 adds TLS termination — backend would otherwise
    think everything is plaintext).
  - `Connection` — hop-by-hop header, should generally not be forwarded as-is.
- **Hop-by-hop vs end-to-end headers.** Headers like `Connection`,
  `Keep-Alive`, `Proxy-Authenticate`, `TE`, `Trailer`, `Transfer-Encoding`,
  `Upgrade` apply only to the immediate connection and should not be blindly
  forwarded between client-proxy and proxy-backend hops.

## What to build

1. `internal/httpmsg/request.go`:
   - `ParseRequest(r *bufio.Reader) (*Request, error)` — reads request line,
     headers, and body (respecting Content-Length / chunked).
   - `(*Request) WriteTo(w io.Writer) error` — serializes back to wire format,
     for forwarding to the backend.
2. `internal/httpmsg/response.go`: same pair, for responses (status line
   instead of request line).
3. `internal/proxy/proxy.go`: `handleConn(clientConn net.Conn)`:
   - Parse the request.
   - Apply header rewrites (Host, X-Forwarded-*).
   - Dial the (still hardcoded) backend, write the rewritten request.
   - Parse the backend's response.
   - Write the response back to the client.
4. Start with **no keep-alive**: handle exactly one request, then close both
   connections. Add keep-alive only after this works correctly, as a
   deliberate second pass (loop on the client connection while
   `Connection: keep-alive` and no error, reusing or re-dialing the backend
   per request — your call, document the tradeoff you pick).

## Go APIs involved

- `bufio.NewReader(conn)`, `Reader.ReadString('\n')` or `ReadLine` for the
  request/status line and headers
- `strconv.Atoi` for parsing `Content-Length`
- `io.ReadFull` for reading an exact number of body bytes
- `net/http.Header` is fine to reuse as your headers map type (it's just a
  `map[string][]string`) — do not use `http.ReadRequest`/`http.ReadResponse`,
  that's the parsing you're meant to write yourself.

## Common mistakes

- Forgetting the blank-line terminator between headers and body — you'll
  either truncate or hang waiting for more data.
- Mishandling chunked encoding — test explicitly with a backend that sends
  chunked responses (most dev servers do this automatically for streamed
  content with unknown length).
- Forwarding hop-by-hop headers verbatim, which can cause subtle bugs against
  real-world backends.
- Not handling `HEAD` requests correctly (no body, but `Content-Length` is
  still set as if there were one).
- Case sensitivity bugs — header names are case-insensitive on the wire,
  `net/http.Header`'s `Get`/`Set` already canonicalize this for you, use it.

## Acceptance criteria

- [ ] A `curl -v` through your proxy round-trips correctly for: a plain GET
  with no body, a POST with a `Content-Length` body, and a response using
  chunked transfer-encoding.
- [ ] `X-Forwarded-For` and `X-Forwarded-Proto` show up correctly on the
  backend side (verify by logging headers on your test backend).
- [ ] Hop-by-hop headers (`Connection`, etc) are stripped, not forwarded.
- [ ] You can explain why `Content-Length` and `Transfer-Encoding` can't both
  be trusted blindly (request smuggling).
- [ ] Malformed input (garbage request line, missing terminator) doesn't crash
  the proxy — it returns an error response or closes the connection cleanly.

## What NOT to do yet

- No multiple backends / load balancing — that's Phase 3.
- No caching — Phase 4.
- No TLS — Phase 5.
