// Copyright 2022 Juan Pablo Tosso and the OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package transformations

import (
	"github.com/ad3n/coraza/v3/internal/strings"
)

func urlDecode(data string) (string, bool, error) {
	for i := 0; i < len(data); i++ {
		if data[i] == '%' || data[i] == '+' {
			// TODO add error?
			return doURLDecode(data, []byte(data), i), true, nil
		}
	}
	return data, false, nil
}

// extracted from https://github.com/senghoo/modsecurity-go/blob/master/utils/urlencode.go
func doURLDecode(input string, d []byte, pos int) string {
	inputLen := len(d)
	i := pos
	c := pos

	for i < inputLen {
		switch input[i] {
		case '%':
			/* Character is a percent sign. */

			/* Are there enough bytes available? */
			switch {
			case i+2 < inputLen:
				c1 := input[i+1]
				c2 := input[i+2]
				switch {
				case strings.ValidHex(c1) && strings.ValidHex(c2):
					uni := strings.X2c(input[i+1:])

					d[c] = uni
					c++
					i += 3
				default:
					/* Not a valid encoding, skip this % */
					d[c] = input[i]
					c++
					i++
				}
			default:
				/* Not enough bytes available, copy the raw bytes. */
				d[c] = input[i]
				c++
				i++
			}
		default:
			/* Character is not a percent sign. */
			switch input[i] {
			case '+':
				d[c] = ' '
				c++
			default:
				d[c] = input[i]
				c++
			}

			i++
		}
	}

	return strings.WrapUnsafe(d[0:c])

}
