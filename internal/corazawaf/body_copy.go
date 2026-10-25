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
	if src == nil && limit > 0 {
		return 0, errNilBodyReader
	}

	state := w.bodyCopyPool.Get().(*bodyCopyState)
	state.reader = io.LimitedReader{R: src, N: limit}
	defer func() {
		state.reader = io.LimitedReader{}
		clear(state.buffer[:])
		w.bodyCopyPool.Put(state)
	}()

	written, err := io.CopyBuffer(dst, &state.reader, state.buffer[:])
	if written == limit {
		return written, nil
	}

	if written < limit && err == nil {
		return written, io.EOF
	}

	return written, err
}
