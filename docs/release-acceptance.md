# Release acceptance

Run the final gate from a clean worktree on the release commit.

```sh
make acceptance
make coverage
docker compose up --build -d
make load-check
docker compose stop --timeout 35
```

`make acceptance` verifies formatting, `go vet`, all unit tests under the race
detector, and the real-socket integration suite under the race detector. The
integration suite covers routing, balancing, caching, atomic reload publication,
rejected-reload retention, HTTP/2 frontend and upstream compatibility, and retry
after a real upstream connection failure.

`make coverage` verifies the repository coverage floor. The CI workflow enforces
the same 85% minimum after running race-enabled coverage.

The load check performs an unmeasured warm-up, records process metrics, sends a
concurrent cache-bypassing workload, closes idle client connections, and samples
the process again after a settling interval. Acceptance requires zero failed
requests and settled goroutine growth no greater than the configured threshold.
Treat sustained heap growth across repeated runs as a failure requiring
investigation; a single before/after heap delta is not proof of a leak because
Go retains heap spans for reuse.

After stopping the stack, verify that the proxy exits within the configured
shutdown timeout and that no Gefahr Compose containers remain. Record the command
output in the release or CI evidence rather than committing machine-specific
throughput and memory numbers to this document.

Pushing an annotated `v*` tag runs the release workflow. It creates the GitHub
release when necessary, uploads checksummed archives for supported operating
systems, publishes AMD64/ARM64 images to GHCR, generates SPDX SBOMs, and
creates GitHub artifact attestations for binaries and container images. Confirm
both workflow jobs complete before announcing the release.

Verify release integrity from a machine with GitHub CLI access:

```sh
gh attestation verify dist/gefahr_1.0.2_linux_amd64.tar.gz -R AnouarMohamed/Gefahr
gh attestation verify oci://ghcr.io/anouarmohamed/gefahr:v1.0.2 -R AnouarMohamed/Gefahr
```

For SBOM attestations, include the SPDX predicate type:

```sh
gh attestation verify dist/gefahr_1.0.2_linux_amd64.tar.gz \
  -R AnouarMohamed/Gefahr \
  --predicate-type https://spdx.dev/Document/v2.3
```
