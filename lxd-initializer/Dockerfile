# Build the manager binary
ARG BUILDER_GOLANG_VERSION=1.24.5
# First stage: build the executable.
FROM --platform=$TARGETPLATFORM us-docker.pkg.dev/palette-images/build-base-images/golang:${BUILDER_GOLANG_VERSION}-alpine as toolchain

FROM toolchain as builder
WORKDIR /workspace

RUN apk update
RUN apk add git gcc g++ curl

# Install build dependencies
RUN apk add --no-cache git make

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Copy vendor directory (contains all dependencies)
COPY vendor/ vendor/

# Download dependencies (optional with vendor)
RUN go mod download

# Copy source code
COPY *.go ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o lxd-initializer .

# Create final image
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache lxc curl jq

# Copy the binary from the builder stage
COPY --from=builder /workspace/lxd-initializer /lxd-initializer

# Set the entrypoint
ENTRYPOINT ["/lxd-initializer"] 