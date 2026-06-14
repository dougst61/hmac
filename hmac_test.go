// Copyright (c) 2026 Doug Stewart

package main

import (
	"testing"
	"time"
)

func TestParseExpiry(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{
			name:  "date only defaults to 23:59 UTC",
			input: "2026-12-31",
			want:  time.Date(2026, 12, 31, 23, 59, 0, 0, time.UTC),
		},
		{
			name:  "date and time",
			input: "2026-12-31 14:30",
			want:  time.Date(2026, 12, 31, 14, 30, 0, 0, time.UTC),
		},
		{
			name:  "midnight",
			input: "2026-01-01 00:00",
			want:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "date only start of year",
			input: "2026-01-01",
			want:  time.Date(2026, 1, 1, 23, 59, 0, 0, time.UTC),
		},
		{
			name:    "wrong date order",
			input:   "31-12-2026",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "not a date",
			input:   "notadate",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseExpiry(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseExpiry(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseExpiry(%q) unexpected error: %v", tt.input, err)
			}
			if !got.Equal(tt.want) {
				t.Errorf("parseExpiry(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
