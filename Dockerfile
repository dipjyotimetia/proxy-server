FROM golang:1.21.0 as builder

# Set the Current Working Directory inside the container
WORKDIR /app

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.* ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

# Build the Go app
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -a -o ./server .

# second stage
FROM debian:buster-slim

# Set the Current Working Directory inside the container
WORKDIR /app

# Install the necessary packages
RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Copy the binary from the builder stage
COPY --from=builder /app/server /app/server

# Run the binary program produced by `go install`
ENTRYPOINT ["/app/server"]
