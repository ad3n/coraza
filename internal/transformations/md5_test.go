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

var md5Vectors = []struct {
	input      string
	wantDigest string // hex-encoded
}{
	{"", "d41d8cd98f00b204e9800998ecf8427e"},
	{"1234", "81dc9bdb52d04dc20036dbd8313ed055"},
	{"transformation", "3935f8feee087b4547de27296ec777b9"},
	{strings.Repeat("a", 1023), "e3e210ff5dfc9da789cdbee262348db6"},
	{strings.Repeat("a", 1024), "c9a34cfc85d982698c6ac89f76071abd"},
	{strings.Repeat("a", 1025), "ae187e1febee2a150b64849c32d566ca"},
	{strings.Repeat("a", 8192), "221994040b14294bdf7fbc128e66633c"},
}

// TestMD5 asserts the digest outside FIPS mode and, inside it (any GODEBUG=fips140 setting),
// that md5T reports an error on every input.
func TestMD5(t *testing.T) {
	for _, tc := range md5Vectors {
		t.Run(strconv.Itoa(len(tc.input)), func(t *testing.T) {
			t.Parallel()
			out, changed, err := md5T(tc.input)

			if fips140.Enabled() {
				if !errors.Is(err, errMD5NotAvailableFIPS) {
					t.Fatalf("want %v, got %v", errMD5NotAvailableFIPS, err)
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
				again, _, err := md5T("different input")
				if err != nil {
					t.Fatal(err)
				}

				if got := hex.EncodeToString([]byte(again)); got != "ea6bd4dd6e31d1f60a6d57aab3fefe89" {
					t.Fatalf("digest after reuse: %s", got)
				}
			}

			if got := hex.EncodeToString([]byte(out)); got != tc.wantDigest {
				t.Errorf("want %s, got %s", tc.wantDigest, got)
			}
		})
	}
}

func BenchmarkMD5(b *testing.B) {
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
				if _, _, err := md5T(tt); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
