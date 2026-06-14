// Copyright (c) 2026 Doug Stewart

package featuretoken

import (
	"strings"
	"testing"
	"time"
)

var testFeatures = map[int]string{
	0: "Alpha",
	1: "Beta",
	2: "Gamma",
}

func newTestManager(t *testing.T) *TokenManager {
	t.Helper()
	tm, err := New([]byte("test-secret-key"), testFeatures)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	return tm
}

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		key         []byte
		features    map[int]string
		wantErr     bool
		errContains string
	}{
		{
			name:     "valid",
			key:      []byte("key"),
			features: map[int]string{0: "F1"},
		},
		{
			name:        "empty key",
			key:         []byte{},
			features:    map[int]string{0: "F1"},
			wantErr:     true,
			errContains: "secret key must not be empty",
		},
		{
			name:        "nil key",
			key:         nil,
			features:    map[int]string{0: "F1"},
			wantErr:     true,
			errContains: "secret key must not be empty",
		},
		{
			name:        "feature position too high",
			key:         []byte("key"),
			features:    map[int]string{32: "F1"},
			wantErr:     true,
			errContains: "out of range",
		},
		{
			name:        "feature position negative",
			key:         []byte("key"),
			features:    map[int]string{-1: "F1"},
			wantErr:     true,
			errContains: "out of range",
		},
		{
			name:     "all 32 positions",
			key:      []byte("key"),
			features: func() map[int]string {
				m := make(map[int]string, 32)
				for i := 0; i < 32; i++ {
					m[i] = "F"
				}
				return m
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm, err := New(tt.key, tt.features)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tm == nil {
				t.Fatal("expected non-nil TokenManager")
			}
		})
	}
}

func TestCreateDecodeToken_roundtrip(t *testing.T) {
	tm := newTestManager(t)
	expiry := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	var bits uint32 = (1 << 0) | (1 << 2) // Alpha + Gamma

	token, err := tm.CreateToken("user1", bits, expiry)
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}
	if token == "" {
		t.Fatal("CreateToken returned empty string")
	}

	gotBits, gotExpiry, err := tm.DecodeToken("user1", token)
	if err != nil {
		t.Fatalf("DecodeToken: %v", err)
	}
	if gotBits != bits {
		t.Errorf("features: got %032b, want %032b", gotBits, bits)
	}
	if !gotExpiry.Equal(expiry) {
		t.Errorf("expiry: got %v, want %v", gotExpiry, expiry)
	}
}

func TestCreateToken_uniqueness(t *testing.T) {
	tm := newTestManager(t)
	expiry := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)

	t1, err := tm.CreateToken("user1", 1, expiry)
	if err != nil {
		t.Fatalf("CreateToken first call: %v", err)
	}
	t2, err := tm.CreateToken("user1", 1, expiry)
	if err != nil {
		t.Fatalf("CreateToken second call: %v", err)
	}
	if t1 == t2 {
		t.Error("identical inputs produced identical tokens; nonce is not working")
	}
}

func TestDecodeToken_errors(t *testing.T) {
	tm := newTestManager(t)
	expiry := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)

	validToken, err := tm.CreateToken("user1", 1, expiry)
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	// Corrupt one character in the middle of the token to produce a different
	// but still valid-length base64 string.
	corrupted := []byte(validToken)
	if corrupted[10] == 'A' {
		corrupted[10] = 'B'
	} else {
		corrupted[10] = 'A'
	}

	tests := []struct {
		name   string
		userID string
		token  string
	}{
		{"wrong user", "user2", validToken},
		{"invalid base64", "user1", "not-valid-base64!!!"},
		{"wrong length", "user1", "aGVsbG8="},
		{"tampered payload", "user1", string(corrupted)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := tm.DecodeToken(tt.userID, tt.token)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestEnabledFeatures(t *testing.T) {
	tm := newTestManager(t)

	tests := []struct {
		name string
		bits uint32
		want []string
	}{
		{
			name: "none enabled",
			bits: 0,
			want: nil,
		},
		{
			name: "bit 0 only",
			bits: 1 << 0,
			want: []string{"Alpha"},
		},
		{
			name: "bits 0 and 2",
			bits: (1 << 0) | (1 << 2),
			want: []string{"Alpha", "Gamma"},
		},
		{
			name: "bit 1 only",
			bits: 1 << 1,
			want: []string{"Beta"},
		},
		{
			name: "unmapped bit ignored",
			bits: 1 << 31,
			want: nil,
		},
		{
			name: "all named bits",
			bits: (1 << 0) | (1 << 1) | (1 << 2),
			want: []string{"Alpha", "Beta", "Gamma"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tm.EnabledFeatures(tt.bits)
			if len(got) != len(tt.want) {
				t.Fatalf("EnabledFeatures(%032b) = %v, want %v", tt.bits, got, tt.want)
			}
			for i, name := range got {
				if name != tt.want[i] {
					t.Errorf("result[%d] = %q, want %q", i, name, tt.want[i])
				}
			}
		})
	}
}
