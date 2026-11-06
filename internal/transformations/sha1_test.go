// Copyright 2022 Juan Pablo Tosso and the OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package transformations

import (
	"crypto/fips140"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"testing"
)

var sha1Vectors = []struct {
	input      string
	wantDigest string // hex-encoded
}{
	{"", "da39a3ee5e6b4b0d3255bfef95601890afd80709"},
	{"1234", "7110eda4d09e062aa5e4a390b0a572ac0d2c0220"},
	{"transformation", "1b3ad029ca2fb8fa799a27606993485cf5094ead"},
	{strings.Repeat("a", 1023), "4aa47ffaf3b2dd2f0241fe22200fca88980f88ab"},
	{strings.Repeat("a", 1024), "8eca554631df9ead14510e1a70ae48c70f9b9384"},
	{strings.Repeat("a", 1025), "1009de2ef6183bd6d8c08f1ed8df4847e71f7acb"},
	{strings.Repeat("a", 8192), "2727756cfee3fbfe24bf5650123fd7743d7b3465"},
}

// TestSHA1 asserts the digest outside FIPS mode and, inside it (any GODEBUG=fips140 setting),
// that sha1T reports an error on every input.
func TestSHA1(t *testing.T) {
	for _, tc := range sha1Vectors {
		t.Run(strconv.Itoa(len(tc.input)), func(t *testing.T) {
			t.Parallel()
			out, changed, err := sha1T(tc.input)

			if fips140.Enabled() {
				if !errors.Is(err, errSHA1NotAvailableFIPS) {
					t.Fatalf("want %v, got %v", errSHA1NotAvailableFIPS, err)
				}
				if changed {
					t.Error("expected the transformation to report an unchanged value")
				}
				if out != tc.input {
					t.Errorf("expected the input to be passed through untransformed, got %q", out)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !changed {
				t.Error("expected the transformation to report a changed value")
			}
			for range 8 {
				again, _, err := sha1T("different input")
				if err != nil {
					t.Fatal(err)
				}

				if got := hex.EncodeToString([]byte(again)); got != "8cfb4fa10d57bd023eb66c693d1d56980759d932" {
					t.Fatalf("digest after reuse: %s", got)
				}
			}

			if got := hex.EncodeToString([]byte(out)); got != tc.wantDigest {
				t.Errorf("want %s, got %s", tc.wantDigest, got)
			}
		})
	}
}

func BenchmarkSHA1(b *testing.B) {
	if fips140.Enabled() {
		b.SkipNow()
	}
	tests := []string{
		"",
		"1234567890",
		strings.Repeat("a", 1024),
		strings.Repeat("a", 8192),
	}
	for _, tc := range tests {
		tt := tc
		b.Run(strconv.Itoa(len(tt)), func(b *testing.B) {
			for b.Loop() {
				if _, _, err := sha1T(tt); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
