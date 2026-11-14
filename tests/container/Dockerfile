FROM node:22-bookworm-slim AS node
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates make git gcc libc6-dev libgl1-mesa-dev libx11-dev libxcursor-dev libxi-dev libxinerama-dev libxrandr-dev libxxf86vm-dev python3 chromium && rm -rf /var/lib/apt/lists/*
COPY --from=node /usr/local/bin/node /usr/local/bin/node
WORKDIR /src
ENV PATH=/go/bin:$PATH GOPATH=/gopath GOCACHE=/tmp/go-cache NODE_PATH=/node_modules GOFLAGS=-buildvcs=false
