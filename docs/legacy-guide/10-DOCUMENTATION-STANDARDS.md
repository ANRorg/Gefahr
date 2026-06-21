# Documentation Standards

This project is a lab, not a product. The code's job is to teach — both you,
reading it back in six months, and anyone you hand it to. "It compiles and
passes the acceptance criteria" is not sufficient on its own; it also has to
be *legible*. This file defines what "good documentation" means here,
concretely enough that an agent can't hand-wave past it.

## The standard, in one line

**Every piece of code should explain the *why*, the comments around it
should teach the underlying protocol/concept, and a stranger should be able
to read the repo top-to-bottom like a textbook.**

## Required, per file

1. **A file-level doc comment** at the top of every `.go` file explaining
   what this file is responsible for and how it fits into the proxy as a
   whole — not just "handles requests" but *why this exists as its own
   file*. Example:

   ```go
   // Package httpmsg implements manual parsing and serialization of HTTP/1.1
   // requests and responses, deliberately avoiding net/http's built-in
   // parser. This is the file where you can see exactly how an HTTP message
   // is framed on the wire: request line, headers, and a body whose length
   // is determined either by Content-Length or by chunked
   // transfer-encoding. See RFC 7230 §3 for the spec this implements.
   package httpmsg
   ```

2. **A doc comment on every exported function/type**, in standard Go doc
   comment form (`// FuncName does X.`), explaining not just the signature
   but *why it works the way it does* where that's non-obvious. Compare:

   - Bad: `// Next returns the next backend.`
   - Good: `// Next returns the next backend using round-robin selection: a
     shared, atomically-incremented index cycling through the alive
     backends list. Atomic rather than mutex-guarded because the only
     shared state is a single integer — see ARCHITECTURE.md's concurrency
     section for why that distinction matters here.`

3. **Inline comments at every non-obvious decision point**, especially:
   - Anywhere the code deviates from the "obvious" approach (e.g. why you're
     buffering instead of streaming, why a deadline is reset where it is).
   - Anywhere a protocol detail is being handled (e.g. the line in your
     chunked-decoder that strips the trailing CRLF after each chunk — say
     *why* it's there, not just what it does).
   - Anywhere a known limitation or simplification is being made (e.g. "we
     don't implement Vary-based cache keys here — see GLOSSARY.md").

## Required, per phase

Each phase, once complete, gets a short `NOTES.md` alongside its code (e.g.
`internal/balancer/NOTES.md`) covering:
- What this phase's code does, in plain language, as if explaining it to
  someone who's read the phase doc but hasn't seen the code yet.
- Any design decisions made that weren't fully pinned down by the phase doc
  (and why you made that call).
- Any known gaps or simplifications, explicitly listed — not silently
  omitted.

This is in addition to, not a replacement for, the in-code doc comments —
`NOTES.md` is the "tour guide" view, doc comments are the "reading the actual
exhibit" view.

## Required, top-level

- The project's own `README.md` (the proxy's, not this guide repo's) must let
  a stranger clone it, run it against real backends, and understand what
  they're looking at — setup steps, config format, how to run each phase's
  tests, and a short "how this works" section that links back to the
  relevant concepts (a one-paragraph summary per major subsystem is enough;
  it should point at the code, not duplicate the phase docs verbatim).
- Any deliberate simplification or out-of-scope item (everything in
  `00-MANIFESTO.md`'s non-goals list, plus anything noted as a "stretch
  goal" in the phase docs) should be listed explicitly in this README under
  a "Known Limitations" section — a lab is honest about what it didn't cover,
  not silent about it.

## What "good" looks like vs what to avoid

**Good:**
```go
// readChunkedBody reads an HTTP/1.1 chunked-encoded body (RFC 7230 §4.1).
// Each chunk is prefixed by its size in hex followed by CRLF, and the body
// terminates with a zero-length chunk. We read chunk-by-chunk into a single
// buffer rather than streaming, trading memory efficiency for simplicity —
// see ARCHITECTURE.md's "Data flow decisions" section for why that's an
// acceptable tradeoff at this stage of the project.
func readChunkedBody(r *bufio.Reader) ([]byte, error) {
    ...
}
```

**Avoid:**
```go
// reads chunked body
func readChunkedBody(r *bufio.Reader) ([]byte, error) {
    ...
}
```

The second isn't wrong, it's just useless — it restates the function name
without adding anything a reader didn't already know.

## Self-check before calling any phase "documented"

- [ ] Could someone who has read the phase doc but never seen this code
  understand *how* it works (not just *that* it works) from the comments
  alone?
- [ ] Does every exported identifier have a doc comment that would survive
  being run through `go doc`?
- [ ] Are protocol-level decisions (framing, header handling, TLS config,
  etc) explained with a "why," not just present in the code with no comment?
- [ ] Is every known simplification/gap stated somewhere (`NOTES.md`,
  top-level README, or both) rather than silently absent?

If any of these is "no," the phase's code is done but its documentation
isn't — and per `09-AGENT-RULES.md`, that means the phase isn't done.
