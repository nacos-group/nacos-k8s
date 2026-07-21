# ---------------------------------------------------------------------------
# Build the peer-finder binary for the target platform.
#
# Previously this image COPY-ed a prebuilt linux/amd64 peer-finder ELF checked
# into the repo (plugin/peer/peer-finder). That made nacos-peer-finder-plugin
# amd64-only: on arm64 nodes the cluster-mode init container fails to pull with
# "no match for platform in manifest" (ImagePullBackOff), so `global.mode=cluster`
# cannot come up on arm64 even though nacos-server itself is multi-arch.
#
# Compile the canonical, Apache-2.0 peer-finder (kmodules.xyz/peer-finder) for
# ${TARGETARCH} instead — the same TARGETARCH cross-compile pattern already used
# by operator/Dockerfile — so a single
#   docker buildx build --platform linux/amd64,linux/arm64 ...
# produces a multi-arch image. The CLI contract used by peer/plugin.sh
# (-on-start / -on-change / -service / -domain) is unchanged, so none of the
# shell scripts need edits.
# ---------------------------------------------------------------------------
FROM golang:1.22 AS builder
ARG TARGETARCH
ENV GOPROXY=https://goproxy.cn,https://goproxy.io,direct
WORKDIR /src
RUN git clone --depth 1 --branch v1.3.0 https://github.com/kmodules/peer-finder . \
 && CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} \
      go build -trimpath -ldflags="-s -w" -o /out/peer-finder .

FROM centos:7
LABEL maintainer="pader <huangmnlove@163.com>"

ADD ./peer/on-start.sh /
ADD ./peer/on-change.sh /
ADD ./peer/plugin.sh /
COPY --from=builder /out/peer-finder /peer-finder
ADD install.sh /
RUN chmod -c 755 /install.sh /on-start.sh /on-change.sh /plugin.sh /peer-finder
ENTRYPOINT ["/install.sh"]
