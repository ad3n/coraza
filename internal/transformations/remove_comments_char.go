// Copyright 2022 Juan Pablo Tosso and the OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package transformations

import (
	"strings"

	stringsutil "github.com/ad3n/coraza/v3/internal/strings"
)

func removeCommentsChar(value string) (string, bool, error) {
	inputLen := len(value)
	start := strings.IndexAny(value, "/*<-#")
	if start < 0 {
		return value, false, nil
	}

	for start < inputLen {
		suffix := value[start:]
		if suffix[0] == '#' || strings.HasPrefix(suffix, "/*") || strings.HasPrefix(suffix, "*/") || strings.HasPrefix(suffix, "<!--") || strings.HasPrefix(suffix, "--") {
			break
		}

		start++
	}

	if start == inputLen {
		return value, false, nil
	}

	res := make([]byte, start, inputLen)
	copy(res, value[:start])
	for i := start; i < inputLen; {
		switch {
		case value[i] == '/' && (i+1 < inputLen) && value[i+1] == '*':
			i += 2
		case value[i] == '*' && (i+1 < inputLen) && value[i+1] == '/':
			i += 2
		case value[i] == '<' &&
			(i+1 < inputLen) &&
			value[i+1] == '!' &&
			(i+2 < inputLen) &&
			value[i+2] == '-' &&
			(i+3 < inputLen) &&
			value[i+3] == '-':
			i += 4
		case value[i] == '-' &&
			(i+1 < inputLen) && value[i+1] == '-' &&
			(i+2 < inputLen) && value[i+2] == '>':
			i += 3
		case value[i] == '-' && (i+1 < inputLen) && value[i+1] == '-':
			i += 2
		case value[i] == '#':
			i += 1
		default:
			res = append(res, value[i])
			i += 1
		}
	}

	return stringsutil.WrapUnsafe(res), true, nil
}
