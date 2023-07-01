# Build the manager binary
FROM golang:1.19.10-alpine3.18 as builder

RUN apk update
RUN apk add git gcc g++ curl

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download
ARG CRYPTO_LIB
ENV GOEXPERIMENT=${CRYPTO_LIB:+boringcrypto}
# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY pkg/ pkg/
COPY controllers/ controllers/

# Build

RUN if [ ${CRYPTO_LIB} ]; \
    then \
      CGO_ENABLED=1 GOOS=linux GOARCH=amd64 GO111MODULE=on go build  -ldflags "-linkmode=external -extldflags=-static" -a -o manager main.go ;\
    else \
      CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o manager main.go ;\
    fi

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER nonroot:nonroot

ENTRYPOINT ["/manager"]
