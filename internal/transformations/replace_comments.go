// Copyright 2022 Juan Pablo Tosso and the OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package transformations

func replaceComments(data string) (string, bool, error) {
	transformedData, changed := doReplaceComments(data)
	return transformedData, changed, nil
}

func doReplaceComments(value string) (string, bool) {
	var i, j int
	incomment := false
	changed := false

	input := []byte(value)
	inputLen := len(input)
	for i < inputLen {
		switch {
		case !incomment:
			switch {
			case (input[i] == '/') && (i+1 < inputLen) && (input[i+1] == '*'):
				incomment = true
				changed = true
				i += 2
			default:
				input[j] = input[i]
				i++
				j++
			}
		default:
			switch {
			case (input[i] == '*') && (i+1 < inputLen) && (input[i+1] == '/'):
				incomment = false
				i += 2
				input[j] = ' '
				j++
			default:
				i++
			}
		}
	}

	if incomment {
		input[j] = ' '
		j++
	}

	return string(input[0:j]), changed
}
