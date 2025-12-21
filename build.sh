#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "$0")" && pwd)
FRONT_DIR="$ROOT_DIR/pages"
DIST_DIR="$ROOT_DIR/dist"
BACK_PUBLIC="$DIST_DIR/public"

log() {
  printf "[build] %s\n" "$1"
}

log "构建前端 (pages)"
cd "$FRONT_DIR"
pnpm install
pnpm run build

log "同步前端静态资源到 public"
cd "$ROOT_DIR"
rm -rf "$BACK_PUBLIC"
mkdir -p "$BACK_PUBLIC"
cp -r "$FRONT_DIR/dist"/* "$BACK_PUBLIC"/

log "构建后端 (Go)"
mkdir -p "$DIST_DIR"
GOFLAGS=${GOFLAGS:-""}
GOOS=${GOOS:-$(go env GOOS)}
GOARCH=${GOARCH:-$(go env GOARCH)}
GOFLAGS="$GOFLAGS" go mod tidy
GOFLAGS="$GOFLAGS" GOOS=$GOOS GOARCH=$GOARCH go build -o "$DIST_DIR/linkit" ./cmd

log "完成，二进制输出在 $DIST_DIR/linkit"
