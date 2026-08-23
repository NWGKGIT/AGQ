# Containerized build and test environment for the AGQ headless daemon.
#
# The desktop app cannot be containerized usefully - it needs a display
# server, GTK/WebKit (Linux) or WebView2 (Windows), and reads /proc of the
# host to discover language servers. The daemon shares all of its logic
# through the same Go packages, so container CI of the daemon covers the
# monitor core; desktop shells are built natively per platform.
#
# Build the daemon image:   docker build -t agq-daemon .
# Run the test suite:       docker build --target test -t agq-test .
#
# Note: process discovery reads the host's /proc, so a containerized daemon
# only sees language servers if run with host PID namespace:
#   docker run --pid=host --network=host -v agq-data:/root/.agq agq-daemon

FROM golang:1.26 AS base
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
COPY monitor/ monitor/

# Test stage: `docker build --target test .` fails the build on test failure.
FROM base AS test
RUN go vet ./... && go test ./...

FROM base AS build
RUN CGO_ENABLED=0 go build -trimpath -o /out/agq-daemon ./cmd/agq-daemon

FROM gcr.io/distroless/static-debian12 AS runtime
COPY --from=build /out/agq-daemon /usr/local/bin/agq-daemon
EXPOSE 7432
ENTRYPOINT ["/usr/local/bin/agq-daemon"]
