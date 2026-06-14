# featuretoken

`featuretoken` is a command-line tool for issuing and verifying software feature
licenses. It lets you control which features are available to which users, and for
how long, without shipping separate builds or maintaining a license server.

You define up to 32 named features. When you want to grant a user access to some
subset of those features until a given date, you run the tool, answer a few
prompts, and it produces a short token string. The user gives that string to your
application, which calls the `featuretoken` package to verify it and check which
features are enabled.

Tokens are cryptographically signed with HMAC-SHA256 and bound to a specific user
ID, so they cannot be forged, transferred between users, or modified to enable
additional features or extend the expiry date. No network connection or central
server is required — verification is entirely local using a shared secret key.

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
