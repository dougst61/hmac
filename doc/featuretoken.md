# featuretoken Package Documentation

## Overview

The `featuretoken` package provides a secure, reusable library for creating and
decoding HMAC-signed, user-bound feature licensing tokens in Go.

Each token encodes:

- A **32-bit feature bitfield** where each bit represents a toggleable feature.
- A **UTC expiry timestamp** after which the token is no longer valid.
- An **8-byte random nonce** that ensures every generated token is unique.

Tokens are **bound to a specific user ID** at creation time. The user ID is
mixed into the HMAC signature but is never stored in the token itself. This
means:

- You **must** provide the correct user ID to decode the token.
- There is **no way** to extract the user ID from the token alone.
- A token created for one user **cannot** be used by another.

---

## Installation

The package lives under the `featuretoken/` subdirectory of this project. To
use it in another Go project, import it by its module path:

```go
import "hmac.go/featuretoken"
```

If you are using this in a separate module, add a `replace` directive in your
`go.mod` or publish the module to a Go module registry.

---

## API Reference

### Types

#### `TokenManager`

```go
type TokenManager struct {
    // unexported fields
}
```

`TokenManager` is the central type for all token operations. It holds the HMAC
secret key and the feature name mapping. Create one with `New()` and reuse it
throughout your application.

---

### Functions

#### `New`

```go
func New(secretKey []byte, featureNames map[int]string) (*TokenManager, error)
```

Creates and returns a configured `TokenManager`.

**Parameters:**

| Name           | Type              | Description                                                             |
|----------------|-------------------|-------------------------------------------------------------------------|
| `secretKey`    | `[]byte`          | The HMAC-SHA256 signing key. Must not be empty. A defensive copy is stored internally. |
| `featureNames` | `map[int]string`  | Maps bit positions (0-31) to human-readable feature names. Not all 32 positions need to be populated. |

**Returns:**

| Type             | Description                                                    |
|------------------|----------------------------------------------------------------|
| `*TokenManager`  | A ready-to-use token manager, or `nil` on error.               |
| `error`          | Non-nil if `secretKey` is empty or any feature position is outside 0-31. |

**Example:**

```go
features := map[int]string{
    0: "DarkMode",
    1: "BetaAPI",
    2: "ExportCSV",
    3: "AdvancedAnalytics",
}

tm, err := featuretoken.New([]byte("my-secret-key"), features)
if err != nil {
    log.Fatal(err)
}
```

**Error conditions:**

- `"secret key must not be empty"` — The `secretKey` slice is nil or zero-length.
- `"feature position X out of range 0-31"` — A key in `featureNames` is negative or greater than 31.

---

#### `CreateToken`

```go
func (tm *TokenManager) CreateToken(userID string, features uint32, expiry time.Time) (string, error)
```

Generates a signed, base64-encoded token bound to the specified user.

**Parameters:**

| Name       | Type        | Description                                                           |
|------------|-------------|-----------------------------------------------------------------------|
| `userID`   | `string`    | The unique identifier of the user this token is issued to. Mixed into the HMAC signature; not stored in the token. |
| `features` | `uint32`    | A 32-bit bitfield where each set bit enables the corresponding feature (bit 0 = feature at position 0, etc.). |
| `expiry`   | `time.Time` | The date and time (in UTC) after which this token should be considered expired. |

**Returns:**

| Type     | Description                                                        |
|----------|--------------------------------------------------------------------|
| `string` | The base64-encoded token string (72 characters), suitable for storage or transmission. |
| `error`  | Non-nil only if the cryptographic random number generator fails (extremely rare). |

**Example:**

```go
// Enable DarkMode (bit 0) and ExportCSV (bit 2).
var bits uint32 = (1 << 0) | (1 << 2)
expiry := time.Date(2026, 12, 31, 23, 59, 0, 0, time.UTC)

token, err := tm.CreateToken("user@example.com", bits, expiry)
if err != nil {
    log.Fatal(err)
}

fmt.Println(token)
// Output: AAAABQAAAABrNuxE... (base64 string, unique each time)
```

**Notes:**

- Every call produces a **unique token** even with identical inputs, due to the random nonce.
- The `expiry` time is stored as a Unix timestamp, so precision is to the second.

---

#### `DecodeToken`

```go
func (tm *TokenManager) DecodeToken(userID string, tokenStr string) (features uint32, expiry time.Time, err error)
```

Verifies and decodes a previously created token.

**Parameters:**

| Name       | Type     | Description                                                                |
|------------|----------|----------------------------------------------------------------------------|
| `userID`   | `string` | The unique identifier of the user presenting this token. Must match the user ID used during creation. |
| `tokenStr` | `string` | The base64-encoded token string as returned by `CreateToken`.              |

**Returns:**

| Name       | Type        | Description                                                     |
|------------|-------------|-----------------------------------------------------------------|
| `features` | `uint32`    | The 32-bit feature bitfield encoded in the token.               |
| `expiry`   | `time.Time` | The UTC expiry time encoded in the token.                       |
| `err`      | `error`     | Non-nil if the token is invalid (see error conditions below).   |

**Example:**

```go
features, expiry, err := tm.DecodeToken("user@example.com", tokenStr)
if err != nil {
    fmt.Println("Token invalid:", err)
    return
}

// Check if the token has expired.
if time.Now().UTC().After(expiry) {
    fmt.Println("Token has expired as of", expiry)
    return
}

// Token is valid — check which features are enabled.
fmt.Printf("Features bitfield: %032b\n", features)
fmt.Printf("Expires: %s\n", expiry.Format("2006-01-02 15:04"))
```

**Error conditions:**

- `"invalid base64: ..."` — The token string is not valid base64.
- `"invalid token length: expected 52 bytes, got N"` — The decoded token is the wrong size.
- `"token verification failed: invalid token or wrong user ID"` — The HMAC signature
  does not match. This occurs when:
  - The wrong `userID` was provided.
  - The token has been modified or corrupted.
  - A different secret key was used to create the token.

> **Security note:** The error message intentionally does not distinguish between
> a wrong user ID and a tampered token, to avoid leaking information to an attacker.

---

#### `EnabledFeatures`

```go
func (tm *TokenManager) EnabledFeatures(bits uint32) []string
```

Resolves a feature bitfield into human-readable names.

**Parameters:**

| Name   | Type     | Description                                                          |
|--------|----------|----------------------------------------------------------------------|
| `bits` | `uint32` | A 32-bit feature bitfield, typically obtained from `DecodeToken`.    |

**Returns:**

| Type       | Description                                                               |
|------------|---------------------------------------------------------------------------|
| `[]string` | Feature names for each set bit that has a configured name, ordered from lowest to highest bit position. Returns `nil` if no named features are set. |

**Example:**

```go
names := tm.EnabledFeatures(features)
for _, name := range names {
    fmt.Printf("  Enabled: %s\n", name)
}
// Output:
//   Enabled: DarkMode
//   Enabled: ExportCSV
```

**Notes:**

- Bits that are set but have **no configured name** (i.e., the bit position was
  not included in the `featureNames` map passed to `New`) are silently skipped.
- This allows you to use only a subset of the 32 available positions.

---

## Token Anatomy: How a Token is Built

This section provides a complete, step-by-step walkthrough of how a token is
created, what it looks like at each stage, and how it is verified during
decoding. We will use a concrete example throughout.

### Inputs

Suppose we want to create a token with the following inputs:

| Input        | Value                          |
|--------------|--------------------------------|
| Secret key   | `mysecretkey123`               |
| User ID      | `jdoe`                         |
| Features     | Feature1 + Feature3 (bits 0,2) |
| Expiry       | 2026-12-31 23:59 UTC           |

### Step 1: Build the Feature Bitfield (4 bytes)

The feature bitfield is a 32-bit unsigned integer where each bit position
corresponds to one feature. Bit 0 is the least significant bit.

To enable Feature1 (position 0) and Feature3 (position 2):

```
Bit position:  31 30 29 ... 3  2  1  0
Value:          0  0  0 ... 0  1  0  1
                                │     │
                                │     └── Bit 0: Feature1 (enabled)
                                └──────── Bit 2: Feature3 (enabled)

As uint32: 0x00000005  (decimal 5)
```

This uint32 value is encoded as **4 bytes in big-endian** (most significant byte
first) format:

```
Byte index:  [0]  [1]  [2]  [3]
Hex:          00   00   00   05
```

### Step 2: Encode the Expiry Timestamp (8 bytes)

The expiry date `2026-12-31 23:59:00 UTC` is converted to a Unix timestamp
(seconds since January 1, 1970 00:00:00 UTC):

```
2026-12-31 23:59:00 UTC = 1798761540 (Unix epoch seconds)
```

This int64 value is encoded as **8 bytes in big-endian** format:

```
Byte index:  [4]  [5]  [6]  [7]  [8]  [9] [10] [11]
Hex:          00   00   00   00   6B   36   EC   44
```

### Step 3: Generate the Random Nonce (8 bytes)

Eight bytes of cryptographically secure random data are generated using Go's
`crypto/rand.Read()`. This nonce serves one purpose: ensuring that every token
is unique, even when all other inputs are identical.

```
Byte index: [12] [13] [14] [15] [16] [17] [18] [19]
Hex:         a7   3f   c1   92   0e   4b   d8   71    (example — different every time)
```

**Why a nonce?** Without it, creating a token for the same user, features, and
expiry would always produce the exact same token. The nonce makes each token
unique, which is useful for:

- Distinguishing tokens issued at different times for the same parameters.
- Preventing trivial detection of duplicate issuances.
- Ensuring that revoking one token (by value) does not affect another with the
  same parameters.

### Step 4: Assemble the Payload (20 bytes)

The three components are concatenated into a single 20-byte payload:

```
+------------------+------------------+------------------+
|   Feature Bits   |  Expiry (Unix)   |      Nonce       |
|    4 bytes       |    8 bytes       |    8 bytes       |
|   (uint32 BE)    |   (int64 BE)     |   (random)       |
+------------------+------------------+------------------+

Offset:  0         4                  12                 20
Hex:     00000005  000000006B36EC44   A73FC1920E4BD871
```

This 20-byte payload is the data that gets signed and is also stored
(unencrypted) in the final token. Anyone who base64-decodes the token can
read these bytes directly — the payload is **not** encrypted.

### Step 5: Compute the HMAC-SHA256 Signature (32 bytes)

The signature is what makes the token tamper-proof and user-bound. It is
computed using HMAC-SHA256 with the secret key, over the **user ID followed
by the payload**:

```
HMAC input construction:

  ┌─────────────────────────────────────────────────────────┐
  │  userID bytes          │  payload bytes                  │
  │  "jdoe" (4 bytes)      │  (20 bytes from Step 4)         │
  │  6A 64 6F 65           │  00 00 00 05 00 ... D8 71       │
  └─────────────────────────────────────────────────────────┘
                              │
                              ▼
             HMAC-SHA256(secretKey = "mysecretkey123",
                         data     = userID + payload)
                              │
                              ▼
                   32-byte signature
```

**Why is the user ID written first?** The user ID is prepended to the HMAC
input so that the same payload signed for `jdoe` produces a completely
different signature than the same payload signed for `asmith`. This binds the
token to a specific user without storing the user ID in the token.

**What is NOT in the token:** The user ID and the secret key are both used to
compute the signature but are **not stored** anywhere in the token. This means:

- Looking at a token reveals the features and expiry date, but not who it
  belongs to.
- Verifying a token requires knowing both the secret key (held by the
  application) and the user ID (provided by the user at decode time).

### Step 6: Concatenate Payload + Signature (52 bytes)

The 20-byte payload and the 32-byte signature are concatenated into a single
52-byte binary blob:

```
+------------------+------------------+------------------+---------------------------+
|   Feature Bits   |  Expiry (Unix)   |      Nonce       |     HMAC-SHA256 Sig       |
|    4 bytes       |    8 bytes       |    8 bytes       |       32 bytes            |
|   (uint32 BE)    |   (int64 BE)     |   (random)       |                           |
+------------------+------------------+------------------+---------------------------+
|<------------- payload (20 bytes) ------------------>|<----- signature (32B) ----->|

                              Total: 52 bytes
```

### Step 7: Base64 Encode (72 characters)

The 52-byte binary blob is encoded using standard base64 encoding (RFC 4648)
to produce a printable ASCII string safe for storage, transmission, copy/paste,
and embedding in configuration files:

```
52 raw bytes  →  base64  →  72 characters

Example: AAAABQAAAABrNuxEpz/Bkg5L2HH... (72 chars total)
```

This final base64 string is the token that gets delivered to the user.

### Field Reference Table

| Field          | Offset | Size  | Encoding          | Description                                                       |
|----------------|--------|-------|-------------------|-------------------------------------------------------------------|
| Feature bits   | 0      | 4B    | uint32 big-endian | Each bit (0-31) enables the feature mapped to that position.      |
| Expiry         | 4      | 8B    | int64 big-endian  | UTC Unix timestamp — seconds since 1970-01-01 00:00:00 UTC.      |
| Nonce          | 12     | 8B    | random bytes      | Cryptographically random data; ensures token uniqueness.          |
| HMAC signature | 20     | 32B   | raw bytes         | HMAC-SHA256 computed over `userID + payload[0:20]`.               |
| **Total**      |        | **52B** |                 | Encodes to **72 base64 characters**.                              |

---

## How Token Verification Works

When a token is presented for decoding, the following steps are performed:

### Step 1: Base64 Decode

The token string is decoded from base64 back to 52 raw bytes. If the string
is not valid base64, or the decoded length is not exactly 52 bytes, the token
is rejected immediately.

### Step 2: Split Payload and Signature

The 52 bytes are split into two parts:

```
payload   = bytes[0:20]    (feature bits + expiry + nonce)
signature = bytes[20:52]   (the HMAC-SHA256 stored in the token)
```

### Step 3: Recompute the Expected Signature

Using the **same secret key** and the **user ID provided at decode time**, the
verifier computes what the HMAC signature *should* be:

```
expected = HMAC-SHA256(secretKey, userID + payload)
```

### Step 4: Constant-Time Signature Comparison

The recomputed signature is compared to the signature stored in the token using
Go's `crypto/hmac.Equal()` function. This function always takes the same amount
of time regardless of how many bytes match, preventing timing side-channel
attacks.

If the signatures match, the token is valid. If they do not match, the token
is rejected. **The error message is intentionally identical** regardless of
whether the failure was caused by:

- A wrong user ID
- A tampered or corrupted token
- A token created with a different secret key

This ambiguity is a deliberate security measure — see "Why Verification Errors
Are Ambiguous" below.

### Step 5: Extract Fields

If verification succeeds, the payload fields are extracted:

```
features = uint32(payload[0:4])    →  the 32-bit feature bitfield
expiry   = int64(payload[4:12])    →  Unix timestamp, converted to time.Time in UTC
nonce    = payload[12:20]          →  (discarded — only needed for uniqueness)
```

The features bitfield and expiry time are returned to the caller. The nonce
is not returned because it has no application-level meaning.

---

## Security Properties

### Tamper-Proof

The HMAC-SHA256 signature covers the entire 20-byte payload. Any modification
to the feature bitfield, expiry time, or nonce will cause signature verification
to fail. It is computationally infeasible to modify the payload and produce a
valid signature without knowing the secret key.

For example, an attacker who tries to:

- **Enable additional features** by flipping bits in bytes 0-3 → the signature
  won't match.
- **Extend the expiry date** by changing bytes 4-11 → the signature won't match.
- **Reuse the nonce** from another token with a different payload → the
  signature won't match.

### User-Bound

The HMAC is computed over the **user ID concatenated with the payload**:

```
HMAC-SHA256(secretKey, userID || payload)
```

This means:

- A token created for `user-A` **cannot** be decoded by `user-B`.
- The user ID is **not stored** in the token — it cannot be extracted by
  inspecting the token.
- An attacker who obtains a token cannot determine which user it belongs to.

### Unique Tokens

Every token includes an 8-byte (64-bit) cryptographic random nonce. This ensures
that even if you create two tokens with the **exact same** user ID, features, and
expiry, the resulting tokens will be different. This prevents replay detection
issues and makes each token distinguishable.

### Timing-Safe Comparison

Signature verification uses Go's `crypto/hmac.Equal()` function, which performs
a constant-time comparison. This prevents timing side-channel attacks where an
attacker could measure response times to guess the expected signature
byte-by-byte.

### Why Verification Errors Are Ambiguous

When token verification fails, the error returned is always:

```
token verification failed: invalid token or wrong user ID
```

This is deliberate. If the system returned different errors for "wrong user ID"
vs. "bad signature", an attacker who possesses a valid token could:

1. Try different user IDs against the token.
2. Observe when the error message changes from "wrong user ID" to "bad
   signature".
3. Conclude that the user ID which produced the "bad signature" error is the
   correct user — confirming which user the token belongs to.

By returning the same error for all failure modes, this information leakage
is prevented. The three indistinguishable failure cases are:

| Failure                        | What Happens Internally                          |
|--------------------------------|--------------------------------------------------|
| Wrong user ID                  | Different user ID → different HMAC → mismatch    |
| Tampered token                 | Modified payload → different HMAC → mismatch     |
| Different secret key           | Different key → different HMAC → mismatch        |

---

## Complete Integration Example

Below is a full working example showing how to integrate the `featuretoken`
package into your own application:

```go
package main

import (
    "fmt"
    "log"
    "time"

    "hmac.go/featuretoken"
)

func main() {
    // Step 1: Define your application's features.
    // Map bit positions (0-31) to descriptive names.
    features := map[int]string{
        0: "DarkMode",
        1: "BetaAPI",
        2: "ExportCSV",
        3: "AdvancedAnalytics",
        4: "PrioritySupport",
    }

    // Step 2: Initialize the TokenManager with your secret key.
    // In production, load this key from secure storage.
    tm, err := featuretoken.New([]byte("your-secret-key-here"), features)
    if err != nil {
        log.Fatalf("Failed to initialize TokenManager: %v", err)
    }

    // Step 3: Create a token for a specific user.
    // Enable DarkMode (0), ExportCSV (2), and PrioritySupport (4).
    var bits uint32 = (1 << 0) | (1 << 2) | (1 << 4)
    expiry := time.Date(2026, 12, 31, 23, 59, 0, 0, time.UTC)

    token, err := tm.CreateToken("user-12345", bits, expiry)
    if err != nil {
        log.Fatalf("Failed to create token: %v", err)
    }

    fmt.Println("Generated token:", token)

    // Step 4: Later, decode the token when the user presents it.
    decodedBits, decodedExpiry, err := tm.DecodeToken("user-12345", token)
    if err != nil {
        log.Fatalf("Token verification failed: %v", err)
    }

    // Step 5: Check expiry.
    if time.Now().UTC().After(decodedExpiry) {
        fmt.Println("Token has expired!")
        return
    }

    // Step 6: Check which features are enabled.
    enabledNames := tm.EnabledFeatures(decodedBits)
    fmt.Println("Active features:")
    for _, name := range enabledNames {
        fmt.Printf("  - %s\n", name)
    }
    fmt.Printf("Valid until: %s UTC\n", decodedExpiry.Format("2006-01-02 15:04"))

    // Step 7: Check a specific feature by bit position.
    if decodedBits&(1<<2) != 0 {
        fmt.Println("ExportCSV is enabled for this user!")
    }
}
```

**Expected output:**

```
Generated token: AAAAFQAAAA... (unique base64 string)
Active features:
  - DarkMode
  - ExportCSV
  - PrioritySupport
Valid until: 2026-12-31 23:59 UTC
ExportCSV is enabled for this user!
```

---

## Feature Bitfield Quick Reference

The feature bitfield is a standard 32-bit unsigned integer. Each bit position
(0 through 31) corresponds to one feature:

```
Bit:  31 30 29 28 27 26 25 24 23 22 21 20 19 18 17 16 15 14 13 12 11 10  9  8  7  6  5  4  3  2  1  0
       |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |
       F32 F31 ...                                                                          F3 F2 F1
```

### Setting bits (enabling features)

```go
var bits uint32

// Enable a single feature (e.g., feature at position 0):
bits |= 1 << 0

// Enable multiple features at once:
bits |= (1 << 0) | (1 << 2) | (1 << 4)

// Enable all 32 features:
bits = 0xFFFFFFFF
```

### Checking bits (querying features)

```go
// Check if feature at position 2 is enabled:
if bits & (1 << 2) != 0 {
    fmt.Println("Feature at position 2 is enabled")
}
```

### Clearing bits (disabling features)

```go
// Disable feature at position 2:
bits &^= 1 << 2
```

---

## Error Handling Guide

All errors returned by the package are standard Go `error` values. Here is a
reference of possible errors and their causes:

| Function       | Error Message                                              | Cause                                            |
|----------------|------------------------------------------------------------|--------------------------------------------------|
| `New`          | `"secret key must not be empty"`                           | Nil or zero-length `secretKey` slice.            |
| `New`          | `"feature position X out of range 0-31"`                   | A key in `featureNames` is < 0 or > 31.         |
| `CreateToken`  | `"failed to generate nonce: ..."`                          | System entropy source unavailable (very rare).   |
| `DecodeToken`  | `"invalid base64: ..."`                                    | Token string is not valid base64 encoding.       |
| `DecodeToken`  | `"invalid token length: expected 52 bytes, got N"`         | Decoded bytes are not the expected 52-byte size. |
| `DecodeToken`  | `"token verification failed: invalid token or wrong user ID"` | HMAC mismatch: wrong user, tampered token, or wrong key. |

---

---
Copyright (c) 2026 Doug Stewart
