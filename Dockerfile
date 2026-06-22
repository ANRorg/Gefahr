# syntax=docker/dockerfile:1.7
FROM golang:1.25.11-alpine AS build
ARG VERSION=dev
ARG COMMIT=unknown
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" -o /out/goproxy ./cmd/goproxy
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/backend ./test/fixtures/backend

FROM gcr.io/distroless/static-debian12:nonroot AS runtime
COPY --from=build /out/goproxy /goproxy
COPY configs/proxy.example.yaml /etc/goproxy/proxy.yaml
USER nonroot:nonroot
EXPOSE 8080 9090
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 CMD ["/goproxy", "-healthcheck", "http://127.0.0.1:9090/readyz"]
ENTRYPOINT ["/goproxy"]
CMD ["-config", "/etc/goproxy/proxy.yaml"]

FROM gcr.io/distroless/static-debian12:nonroot AS fixture
COPY --from=build /out/backend /backend
USER nonroot:nonroot
ENTRYPOINT ["/backend"]
