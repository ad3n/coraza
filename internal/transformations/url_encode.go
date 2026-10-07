// Copyright 2022 Juan Pablo Tosso and the OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package transformations

import "strings"

func urlEncode(data string) (string, bool, error) {
	transformedData, changed := doURLEncode(data)
	return transformedData, changed, nil
}

func doURLEncode(input string) (string, bool) {
	inputLen := len(input)
	if inputLen == 0 {
		return "", false
	}

	first := 0
	for first < inputLen {
		c := input[first]
		if c != '*' && (c < '0' || c > '9') && (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') {
			break
		}

		first++
	}

	if first == inputLen {
		return input, false
	}

	leng := inputLen * 3
	var d strings.Builder
	d.Grow(leng)
	c2xTable := []byte("0123456789abcdef")

	d.WriteString(input[:first])

	for i := first; i < inputLen; i++ {
		cc := input[i]

		switch cc {
		case ' ':
			d.WriteByte('+')
		default:
			switch {
			case (cc == 42) || ((cc >= 48) && (cc <= 57)) || ((cc >= 65) && (cc <= 90)) || ((cc >= 97) && (cc <= 122)):
				d.WriteByte(cc)
			default:
				d.Write([]byte{'%', c2xTable[(cc&0xff)>>4], c2xTable[(cc&0xff)&0x0f]})
			}
		}
	}

	return d.String(), true
}
