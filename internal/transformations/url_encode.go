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
	pos := 0
	for pos < inputLen {
		cc := input[pos]
		if cc != '*' && (cc < '0' || cc > '9') && (cc < 'A' || cc > 'Z') && (cc < 'a' || cc > 'z') {
			break
		}

		pos++
	}

	if pos == inputLen {
		return input, false
	}

	var d strings.Builder
	d.Grow(pos + (inputLen-pos)*3)
	d.WriteString(input[:pos])
	const c2xTable = "0123456789abcdef"

	for i := pos; i < inputLen; i++ {
		cc := input[i]

		switch cc {
		case ' ':
			d.WriteByte('+')
		default:
			switch {
			case (cc == 42) || ((cc >= 48) && (cc <= 57)) || ((cc >= 65) && (cc <= 90)) || ((cc >= 97) && (cc <= 122)):
				d.WriteByte(cc)
			default:
				d.Write([]byte{'%', c2xTable[cc>>4], c2xTable[cc&0x0f]})
			}
		}
	}

	return d.String(), true
}
