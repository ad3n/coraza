// Copyright 2022 Juan Pablo Tosso and the OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package transformations

import "testing"

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
		{
			input: "*-_.+/?%\x00\xff",
			want:  "*%2d%5f%2e%2b%2f%3f%25%00%ff",
		},
	}

	for _, tt := range tests {
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
	tests = append(tests, "helloWorld0123*", "")

	for _, input := range tests {
		b.Run(input, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, _, err := urlEncode(input); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
