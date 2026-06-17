// Copyright (c) 2026 Doug Stewart

// Package featuretoken provides HMAC-signed, user-bound feature licensing tokens.
//
// A token encodes a 32-bit feature bitfield and a UTC expiry timestamp, signed
// with HMAC-SHA256. Tokens are bound to a specific user ID at creation time;
// the same user ID must be presented to decode and verify the token. The user ID
// is mixed into the HMAC computation but is never stored in the token itself,
// making it impossible to extract from the token alone.
//
// # Token Binary Layout
//
// Each token is a base64-encoded byte sequence with the following structure:
//
//	Offset  Size   Description
//	------  ----   -----------
//	0       4B     Feature bitfield (uint32, big-endian)
//	4       8B     Expiry timestamp (int64 Unix epoch, big-endian)
//	12      8B     Random nonce (ensures token uniqueness)
//	20      32B    HMAC-SHA256 signature
//	------  ----
//	Total:  52 bytes (encoded to 72 base64 characters)
//
// # Security Properties
//
//   - Tamper-proof: Any modification to the payload invalidates the HMAC signature.
//   - User-bound: The HMAC is computed over the user ID concatenated with the payload,
//     so a token created for one user cannot be verified by another.
//   - Unique: An 8-byte cryptographic random nonce ensures that identical inputs
//     (same user, features, and expiry) produce different tokens each time.
//   - Timing-safe: Signature comparison uses crypto/hmac.Equal to prevent timing attacks.
//
// # Quick Start
//
//	// 1. Define your features as a map of bit position (0-31) to name.
//	features := map[int]string{
//	    0: "DarkMode",
//	    1: "BetaAPI",
//	    2: "ExportCSV",
//	}
//
//	// 2. Create a TokenManager with your secret key and feature map.
//	tm, err := featuretoken.New([]byte("my-secret-key"), features)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// 3. Create a token for a user with specific features enabled until a date.
//	//    Bits 0 and 2 → DarkMode + ExportCSV.
//	var bits uint32 = (1 << 0) | (1 << 2)
//	expiry := time.Date(2026, 12, 31, 23, 59, 0, 0, time.UTC)
//	token, err := tm.CreateToken("user@example.com", bits, expiry)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(token) // base64-encoded string
//
//	// 4. Decode the token (requires the same user ID).
//	features, exp, err := tm.DecodeToken("user@example.com", token)
//	if err != nil {
//	    log.Fatal(err) // fails if wrong user or tampered token
//	}
//	fmt.Println(features, exp)
//
//	// 5. Resolve bitfield to human-readable feature names.
//	names := tm.EnabledFeatures(features)
//	fmt.Println(names) // ["DarkMode", "ExportCSV"]
package featuretoken

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

// tokenPayloadSize is the byte length of the token payload before the HMAC
// signature: 4 (features) + 8 (expiry) + 8 (nonce) = 20 bytes.
const tokenPayloadSize = 20

// hmacSize is the byte length of an HMAC-SHA256 signature.
const hmacSize = 32

// totalTokenSize is the expected byte length of a complete decoded token:
// payload (20) + HMAC signature (32) = 52 bytes.
const totalTokenSize = tokenPayloadSize + hmacSize

// TokenManager holds the configuration needed to create and decode tokens.
// It stores a copy of the secret key and the feature name mapping. Create
// one using [New] and reuse it for all token operations.
type TokenManager struct {
	// secretKey is the HMAC-SHA256 signing key. A defensive copy is stored
	// so that external modifications to the original slice do not affect
	// token operations.
	secretKey []byte

	// featureNames maps bit positions (0-31) to human-readable feature names.
	// Only positions present in this map are included when resolving names
	// via [EnabledFeatures].
	featureNames map[int]string
}

// New creates and returns a configured TokenManager.
//
// Parameters:
//   - secretKey: The HMAC-SHA256 signing key. Must not be empty. A defensive
//     copy is stored internally so the caller may safely modify the original
//     slice after calling New.
//   - featureNames: A map of bit positions (0-31) to human-readable feature
//     names. Each key must be in the range [0, 31]. The map may contain fewer
//     than 32 entries; unmapped positions are simply unnamed.
//
// Returns:
//   - *TokenManager: A ready-to-use token manager, or nil on error.
//   - error: Non-nil if secretKey is empty or any feature position is out of range.
//
// Example:
//
//	features := map[int]string{
//	    0: "DarkMode",
//	    1: "BetaAPI",
//	    2: "ExportCSV",
//	}
//	tm, err := featuretoken.New([]byte("my-secret-key"), features)
//	if err != nil {
//	    log.Fatal(err)
//	}
func New(secretKey []byte, featureNames map[int]string) (*TokenManager, error) {
	// Validate that the secret key is not empty. An empty key would produce
	// trivially forgeable HMAC signatures.
	if len(secretKey) == 0 {
		return nil, errors.New("secret key must not be empty")
	}

	// Validate that all feature positions are within the 32-bit range.
	for pos := range featureNames {
		if pos < 0 || pos > 31 {
			return nil, fmt.Errorf("feature position %d out of range 0-31", pos)
		}
	}

	// Make a defensive copy of the secret key so external changes to the
	// caller's slice do not affect our HMAC computations.
	key := make([]byte, len(secretKey))
	copy(key, secretKey)

	return &TokenManager{
		secretKey:    key,
		featureNames: featureNames,
	}, nil
}

// CreateToken generates a signed, base64-encoded token bound to a specific user.
//
// The token encodes the given feature bitfield and expiry time, and is signed
// with HMAC-SHA256 using the configured secret key. The user ID is mixed into
// the HMAC computation but is not stored in the token, so the same user ID
// must be provided later to [DecodeToken] for verification.
//
// An 8-byte cryptographic random nonce is included in each token to ensure
// that identical inputs produce unique tokens every time.
//
// Parameters:
//   - userID:   The unique identifier of the user this token is issued to.
//   - features: A 32-bit bitfield where each set bit enables the corresponding feature.
//   - expiry:   The UTC date and time after which this token is considered expired.
//
// Returns:
//   - string: The base64-encoded token string, suitable for storage or transmission.
//   - error:  Always nil; retained for interface compatibility.
//
// Example:
//
//	// Enable features at bit positions 0 and 2 (DarkMode + ExportCSV).
//	var bits uint32 = (1 << 0) | (1 << 2)
//	expiry := time.Date(2026, 12, 31, 23, 59, 0, 0, time.UTC)
//	token, err := tm.CreateToken("user@example.com", bits, expiry)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(token)
func (tm *TokenManager) CreateToken(userID string, features uint32, expiry time.Time) (string, error) {
	// Build the binary payload:
	//   Bytes  0-3:   Feature bitfield as big-endian uint32
	//   Bytes  4-11:  Expiry as big-endian int64 Unix timestamp (UTC)
	//   Bytes 12-19:  Cryptographic random nonce for uniqueness
	payload := make([]byte, tokenPayloadSize)
	binary.BigEndian.PutUint32(payload[0:4], features)
	binary.BigEndian.PutUint64(payload[4:12], uint64(expiry.UTC().Unix()))

	// Fill the nonce portion with cryptographically secure random bytes.
	// This ensures every token is unique even with identical inputs.
	// rand.Read never returns an error (Go 1.20+); it crashes on failure.
	rand.Read(payload[12:20])

	// Compute HMAC-SHA256 over the user ID followed by the payload.
	// Writing the user ID first binds the token to this specific user;
	// a different user ID will produce a completely different signature.
	mac := hmac.New(sha256.New, tm.secretKey)
	mac.Write([]byte(userID))
	mac.Write(payload)
	signature := mac.Sum(nil)

	// Concatenate payload + signature and encode as base64 for safe
	// storage and transmission.
	token := append(payload, signature...)
	return base64.StdEncoding.EncodeToString(token), nil
}

// DecodeToken verifies and decodes a token for the given user ID.
//
// The token's HMAC signature is recomputed using the provided user ID and
// compared (in constant time) against the signature stored in the token.
// If the user ID does not match the one used during creation, or if the
// token has been tampered with, verification fails and an error is returned.
//
// Parameters:
//   - userID:   The unique identifier of the user presenting this token.
//   - tokenStr: The base64-encoded token string as returned by [CreateToken].
//
// Returns:
//   - features: The 32-bit feature bitfield encoded in the token.
//   - expiry:   The UTC expiry time encoded in the token.
//   - err:      Non-nil if the token is malformed, the base64 encoding is
//     invalid, the signature does not match, or the user ID is wrong.
//     Note: the error intentionally does not distinguish between a wrong
//     user ID and a tampered token, to avoid leaking information.
//
// Example:
//
//	features, expiry, err := tm.DecodeToken("user@example.com", tokenStr)
//	if err != nil {
//	    log.Fatal(err) // invalid token or wrong user
//	}
//	if time.Now().UTC().After(expiry) {
//	    fmt.Println("Token has expired")
//	}
//	fmt.Printf("Features: %032b\n", features)
func (tm *TokenManager) DecodeToken(userID string, tokenStr string) (features uint32, expiry time.Time, err error) {
	// Decode the base64 token string back into raw bytes.
	raw, err := base64.StdEncoding.DecodeString(tokenStr)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("invalid base64: %w", err)
	}

	// Verify the decoded length matches the expected total token size.
	if len(raw) != totalTokenSize {
		return 0, time.Time{}, fmt.Errorf("invalid token length: expected %d bytes, got %d", totalTokenSize, len(raw))
	}

	// Split the raw bytes into payload and signature portions.
	payload := raw[:tokenPayloadSize]
	signature := raw[tokenPayloadSize:]

	// Recompute the expected HMAC-SHA256 signature using the provided user ID
	// and the payload from the token. The user ID must match the one used
	// during creation or the signatures will differ.
	mac := hmac.New(sha256.New, tm.secretKey)
	mac.Write([]byte(userID))
	mac.Write(payload)
	expectedSig := mac.Sum(nil)

	// Compare signatures using hmac.Equal for constant-time comparison.
	// This prevents timing side-channel attacks that could be used to
	// forge signatures byte-by-byte.
	if !hmac.Equal(signature, expectedSig) {
		return 0, time.Time{}, fmt.Errorf("token verification failed: invalid token or wrong user ID")
	}

	// Extract the feature bitfield from bytes 0-3 (big-endian uint32).
	features = binary.BigEndian.Uint32(payload[0:4])

	// Extract the expiry timestamp from bytes 4-11 (big-endian int64 Unix epoch)
	// and convert to a time.Time in UTC.
	unixExpiry := int64(binary.BigEndian.Uint64(payload[4:12]))

	return features, time.Unix(unixExpiry, 0).UTC(), nil
}

// EnabledFeatures returns the human-readable names of all features that are
// enabled (bit set to 1) in the given bitfield.
//
// Only bit positions that were registered in the feature names map passed to
// [New] are included in the result. Bits that are set but have no corresponding
// name in the map are silently skipped. The returned names are ordered from
// the lowest bit position to the highest.
//
// Parameters:
//   - bits: A 32-bit feature bitfield, typically obtained from [DecodeToken].
//
// Returns:
//   - []string: A slice of feature names for each set bit that has a configured
//     name. Returns nil if no named features are enabled.
//
// Example:
//
//	// Assuming bit 0 = "DarkMode", bit 2 = "ExportCSV":
//	var bits uint32 = (1 << 0) | (1 << 2)
//	names := tm.EnabledFeatures(bits)
//	// names = ["DarkMode", "ExportCSV"]
func (tm *TokenManager) EnabledFeatures(bits uint32) []string {
	var enabled []string

	// Walk each of the 32 bit positions from lowest to highest.
	for i := 0; i < 32; i++ {
		// Check if bit at position i is set in the bitfield.
		if bits&(1<<uint(i)) != 0 {
			// Only include this feature if a name was configured for this position.
			if name, ok := tm.featureNames[i]; ok {
				enabled = append(enabled, name)
			}
		}
	}
	return enabled
}
