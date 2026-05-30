# PicoClaw — build targets (Phase 1)
#
# The prebuilt picoclaw.exe / picoclaw-launcher.exe in this folder are the
# upstream release binaries; our from-scratch build outputs to bin/ to avoid
# clobbering them.

BIN      := bin
PKG      := ./cmd/picoclaw
BINARY   := $(BIN)/picoclaw-dev
LDFLAGS  := -s -w

.PHONY: all build run vet fmt test clean build-all

all: build

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY)$(EXT) $(PKG)

vet:
	go vet ./...

fmt:
	gofmt -w .

test:
	go test ./...

run: build
	$(BINARY)$(EXT) agent

clean:
	rm -rf $(BIN)

# Cross-architecture builds (single static binary per target).
build-all:
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN)/picoclaw-linux-amd64   $(PKG)
	GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BIN)/picoclaw-linux-arm64   $(PKG)
	GOOS=linux   GOARCH=arm   GOARM=6 go build -ldflags "$(LDFLAGS)" -o $(BIN)/picoclaw-linux-armv6 $(PKG)
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN)/picoclaw-windows-amd64.exe $(PKG)
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BIN)/picoclaw-darwin-arm64  $(PKG)

# Raspberry Pi Zero (ARMv6).
build-pi-zero:
	GOOS=linux GOARCH=arm GOARM=6 go build -ldflags "$(LDFLAGS)" -o $(BIN)/picoclaw-pi-zero $(PKG)
