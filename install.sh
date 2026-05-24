#!/bin/bash
set -e

GO_VERSION=1.22.4
GO_DIR=/usr/local

install_go() {
    ARCH=$(uname -m)
    case $ARCH in
        x86_64)  GOARCH=amd64 ;;
        aarch64) GOARCH=arm64 ;;
        armv7l)  GOARCH=armv6l ;;
        *) echo "Unsupported arch: $ARCH"; exit 1 ;;
    esac
    echo "Installing Go ${GO_VERSION} (${GOARCH})..."
    curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${GOARCH}.tar.gz" \
        | sudo tar -C "$GO_DIR" -xz
    export PATH=$PATH:$GO_DIR/go/bin
    echo "Go installed: $(go version)"
}

if ! command -v go >/dev/null 2>&1; then
    install_go
fi

echo "Building splitroute..."
go build -mod=vendor -o splitroute .

sudo install -m 755 splitroute /usr/local/bin/splitroute
echo "Installed: $(which splitroute)"
