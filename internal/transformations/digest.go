// Copyright 2026 Juan Pablo Tosso and the OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package transformations

import (
	"hash"

	stringsutil "github.com/ad3n/coraza/v3/internal/strings"
	"github.com/ad3n/coraza/v3/internal/sync"
)

type digestState struct {
	hash   hash.Hash
	buffer [1024]byte
}

func pooledDigest(data string, pool sync.Pool) string {
	state := pool.Get().(*digestState)
	used := min(len(data), len(state.buffer))
	defer func() {
		clear(state.buffer[:used])
		state.hash.Reset()
		pool.Put(state)
	}()

	for len(data) > 0 {
		n := copy(state.buffer[:], data)
		_, _ = state.hash.Write(state.buffer[:n])
		data = data[n:]
	}

	return stringsutil.WrapUnsafe(state.hash.Sum(nil))
}
