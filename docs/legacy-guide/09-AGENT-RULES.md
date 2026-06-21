# Rules for the Coding Agent (Codex)

Paste this whole file into the agent's context before asking it to write any
code for this project. It exists because a coding agent, left to its own
judgment, will take the fastest path to "it works" — which for a reverse
proxy means reaching for `net/http/httputil.ReverseProxy` and silently
defeating the entire point of this project. These rules close that loophole
and others like it.

## Hard constraints — violating any of these means redo the work

1. **Never use `net/http/httputil.ReverseProxy`, `http.Server` with a
   handler-based model for the proxy's own listener, or any third-party
   reverse-proxy library.** The HTTP parsing and proxying logic must be
   hand-written using `net`, `bufio`, `tls`, and `io` from the standard
   library. If you (the agent) find yourself about to import
   `net/http/httputil`, stop — that's the line.
2. **`net/http.Header` as a data type is fine.** It's just a
   `map[string][]string` with case-insensitive helpers. What's not fine is
   `http.ReadRequest` / `http.ReadResponse` / `http.Server.Serve` doing the
   parsing or connection-handling work for you.
3. **Do not skip ahead to a later phase's file structure before the current
   phase's acceptance criteria (in its own `0N-PHASE-*.md` file) are met.**
   If asked to "build the whole thing," still build and verify phase by
   phase, in order — don't generate all six phases' code in one shot with no
   checkpoint.
4. **Do not silently add scope** that isn't in `00-MANIFESTO.md`'s scope or
   a phase doc — no HTTP/2, no web UI, no Redis, no WebSocket proxying,
   unless explicitly asked. If you think something's missing, say so and ask
   rather than just adding it.
5. **Every phase must end with something runnable and testable**, matching
   that phase's acceptance criteria checklist — not just code that compiles.
   If you can't demonstrate the acceptance criteria pass, the phase isn't
   done, regardless of how complete the code looks.
6. **When something requires a judgment call the docs don't pin down**
   (e.g. exact header size limit, exact health-check interval), pick a
   reasonable value, but explicitly say what you picked and why — don't bury
   the decision silently in code with no comment.
7. **Documentation is part of the deliverable, not a polish pass.** This
   project is a lab/TP for understanding reverse proxies — code that works
   but isn't explained teaches nothing. Follow `10-DOCUMENTATION-STANDARDS.md`
   for every phase: file-level doc comments, doc comments on every exported
   identifier, inline "why" comments at non-obvious decisions, and a
   `NOTES.md` per phase. A phase with working code and thin comments is not
   a completed phase — treat it the same as failing acceptance criteria.

## Workflow to follow

1. Read `00-MANIFESTO.md`, `01-ARCHITECTURE.md`, and
   `10-DOCUMENTATION-STANDARDS.md` in full before writing anything.
2. Work through `02` → `07` in order. For each phase:
   - Re-read that phase's doc fully before coding it.
   - Build exactly what "What to build" describes — no more, no less.
   - Self-check against "Common mistakes" before declaring it done.
   - Write the tests described in the corresponding section of
     `08-TESTING.md`.
   - Run them. Show the output. Don't just claim it works.
   - Check off the relevant boxes in `CHECKLIST.md`.
3. Only move to the next phase after the current one's acceptance criteria
   are demonstrably met.

## If you (the human) notice the agent cutting corners

Common tells that it's taking a shortcut instead of doing the work:
- It imports `net/http/httputil` or builds the proxy as an `http.Handler`.
- It generates all the code for every phase in one response with no test
  output shown.
- It marks acceptance criteria boxes checked without having actually run
  anything.
- It quietly adds a dependency (a third-party load-balancer or cache
  library) instead of writing the ~50 lines of Go this project calls for.
- It "simplifies" chunked transfer encoding handling by just not
  implementing it, without flagging that as a gap.
- It writes working code with bare or absent comments, or skips the
  per-phase `NOTES.md` — "it works" isn't the bar here, "it's explained" is.

If you see any of these, stop it, point at the specific rule above, and have
it redo that piece.

## Why this matters

The entire value of this project is that *you* end up understanding how a
reverse proxy works, with code you can explain line by line. An agent that
"helpfully" shortcuts to a working binary using a library that does the job
for you produces something that runs identically from the outside but
teaches you nothing. Treat working-but-shortcut code as a failed deliverable,
not a fast success.
