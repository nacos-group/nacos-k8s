# Build the manager binary
FROM golang:1.16 as builder
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
ADD https://raw.githubusercontent.com/alibaba/nacos/develop/distribution/conf/mysql-schema.sql config/sql/nacos-mysql.sql
RUN sed -i 's/INSERT INTO/INSERT IGNORE INTO/' config/sql/nacos-mysql.sql

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} GO111MODULE=on go build  -a -v -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM registry.cn-hangzhou.aliyuncs.com/shenkonghui/distroless-static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
COPY --from=builder --chmod=644 /workspace/config/sql/nacos-mysql.sql config/sql/nacos-mysql.sql
USER nonroot:nonroot

ENTRYPOINT ["/manager"]
