# featuretoken

Interactive command-line tool for creating and decoding HMAC-signed feature
licensing tokens.

## Overview

Each token encodes a 32-bit feature bitfield and a UTC expiry timestamp, signed
with HMAC-SHA256. Tokens are bound to a specific user ID at creation time — the
same user ID must be provided to decode the token. The payload is not encrypted
but is tamper-proof; any modification invalidates the signature.

See [`doc/featuretoken.md`](doc/featuretoken.md) for full package documentation
including the token binary layout and security properties.

## Usage

    ./featuretoken

The tool presents an interactive menu:

- **Option 1 — Create a token**: Enter a user ID, select features by number,
  and provide an expiry date (`YYYY-MM-DD` or `YYYY-MM-DD HH:MM`). The tool
  outputs a base64 token string.

- **Option 2 — Decode a token**: Enter the user ID and paste the token. The
  tool verifies the signature and displays the enabled features, expiry date,
  and active/expired status.

## Configuration

Edit `hmac.go` to customize the tool before building:

| Variable       | Purpose                                                         |
|----------------|-----------------------------------------------------------------|
| `secretKey`    | HMAC-SHA256 signing key. Load from a secrets manager in production — do not hardcode in source. |
| `featureNames` | Maps bit positions (0–31) to human-readable feature names.     |

## Build

    make build    # build for current OS and architecture → build/featuretoken
    make all      # build all four release targets
    make test     # run all tests
    make lint     # run go vet
    make fmt      # run go fmt
    make clean    # remove build/ directory

Alternatively, `./build.sh` builds all four targets directly.

## Release targets

| Platform         | Output                              |
|------------------|-------------------------------------|
| macOS Apple Silicon | `build/featuretoken-darwin-arm64` |
| macOS Intel         | `build/featuretoken-darwin-amd64` |
| Linux AMD64         | `build/featuretoken-linux-amd64`  |
| Windows AMD64       | `build/featuretoken-windows-amd64.exe` |

---
Copyright (c) 2026 Doug Stewart
