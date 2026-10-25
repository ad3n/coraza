// Copyright 2026 The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package corazawaf

import (
	"errors"
	"io"
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

	state := w.bodyCopyPool.Get().(*bodyCopyState)
	size := min(limit, int64(len(state.buffer)))
	buffer := state.buffer[:size:size]
	state.reader = io.LimitedReader{R: src, N: limit}
	defer func() {
		state.reader = io.LimitedReader{}
		clear(buffer)
		w.bodyCopyPool.Put(state)
	}()

	written, err := io.CopyBuffer(dst, &state.reader, buffer)
	if written == limit {
		return written, nil
	}

	if written < limit && err == nil {
		return written, io.EOF
	}

	return written, err
}
