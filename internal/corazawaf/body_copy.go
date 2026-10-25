// Copyright 2026 The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package corazawaf

import (
	"bytes"
	"errors"
	"io"
	"strings"
)

var errNilBodyReader = errors.New("nil body reader")

type bodyCopyState struct {
	reader io.LimitedReader
	buffer [32 * 1024]byte
}

func (w *WAF) copyBodyN(dst *BodyBuffer, src io.Reader, limit int64) (int64, error) {
	if limit <= 0 {
		return 0, nil
	}

	if src == nil {
		return 0, errNilBodyReader
	}

	available := limit
	switch reader := src.(type) {
	case *bytes.Reader:
		available = int64(reader.Len())
	case *bytes.Buffer:
		available = int64(reader.Len())
	case *strings.Reader:
		available = int64(reader.Len())
	}

	state := w.bodyCopyPool.Get().(*bodyCopyState)
	size := min(limit, max(1, available), int64(len(state.buffer)))
	buffer := state.buffer[:size:size]
	state.reader = io.LimitedReader{R: src, N: limit}
	defer func() {
		state.reader = io.LimitedReader{}
		clear(buffer)
		w.bodyCopyPool.Put(state)
	}()

	var (
		written int64
		err     error
	)
	for {
		n, readErr := state.reader.Read(buffer)
		if n > 0 {
			copied, writeErr := dst.Write(buffer[:n])
			written += int64(copied)
			if writeErr != nil {
				err = writeErr
				break
			}

			if copied != n {
				err = io.ErrShortWrite
				break
			}
		}

		if readErr != nil {
			if readErr != io.EOF {
				err = readErr
			}

			break
		}
	}

	if written == limit {
		return written, nil
	}

	if written < limit && err == nil {
		return written, io.EOF
	}

	return written, err
}
