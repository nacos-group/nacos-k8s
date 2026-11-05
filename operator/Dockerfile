# Build the manager binary

FROM golang:1.16 AS builder
ARG TARGETARCH

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
ENV GOPROXY=https://goproxy.cn,https://goproxy.io,direct
#RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY pkg/ pkg/
# Pinned to the official Nacos 2.4.3 schema (see config/sql/README.md).
COPY config/sql/nacos-mysql.sql config/sql/nacos-mysql.sql

# Build
# Keep compiler concurrency and GC memory use suitable for small build hosts.
RUN GOMAXPROCS=1 GOGC=50 CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH:-amd64} GO111MODULE=on go build -mod=readonly -p 1 -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM registry.cn-hangzhou.aliyuncs.com/shenkonghui/distroless-static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
COPY --from=builder /workspace/config/sql/nacos-mysql.sql config/sql/nacos-mysql.sql
USER nonroot:nonroot

ENTRYPOINT ["/manager"]
