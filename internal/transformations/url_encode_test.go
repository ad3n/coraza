// Copyright 2022 Juan Pablo Tosso and the OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package transformations

import (
	"strings"
	"testing"
)

func TestEncode(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			input: "",
			want:  "",
		},
		{
			input: "helloWorld",
			want:  "helloWorld",
		},
		{
			input: "hello world",
			want:  "hello+world",
		},
		{
			input: "https://www.coraza.io",
			want:  "https%3a%2f%2fwww%2ecoraza%2eio",
		},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.input, func(t *testing.T) {
			have, changed, err := urlEncode(tt.input)
			if err != nil {
				t.Error(err)
			}
			if tt.input == tt.want && changed || tt.input != tt.want && !changed {
				t.Errorf("input %q, have %q with changed %t", tt.input, have, changed)
			}
			if have != tt.want {
				t.Errorf("have %q, want %q", have, tt.want)
			}
		})
	}
}

func BenchmarkURLEncode(b *testing.B) {
	tests := []string{
		" !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}",
		"ÀÁÂÃÄÅÆÇÈÉÊËÌÍÎÏÐÑÒÓÔÕÖ×ØÙÚÛÜÝÞßàáâãäåæçèéêëìíîïðñòóôõö÷øùúûüýþÿ",
		"~", //nolint:staticcheck
		"Test Case",
	}
	tests = append(tests, "helloWorld0123*", "", strings.Repeat("abcXYZ012*", 32), strings.Repeat("abcXYZ012*", 32)+" ")
	for _, tt := range tests {
		b.Run(tt, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_, _, err := urlEncode(tt)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
