// Copyright 2022 Juan Pablo Tosso and the OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package transformations

import (
	"crypto/fips140"
	"crypto/sha1"
	"errors"

	"github.com/ad3n/coraza/v3/internal/sync"
)

const emptySHA1 = "\xda\x39\xa3\xee\x5e\x6b\x4b\x0d\x32\x55\xbf\xef\x95\x60\x18\x90\xaf\xd8\x07\x09"

var (
	errSHA1NotAvailableFIPS = errors.New("sha1 transformation is unavailable in FIPS 140-3 mode")
	sha1Pool                = sync.NewPool(func() any {
		return &digestState{hash: sha1.New()}
	})
)

func sha1T(data string) (string, bool, error) {
	if fips140.Enabled() {
		return data, false, errSHA1NotAvailableFIPS
	}

	if len(data) == 0 {
		return emptySHA1, true, nil
	}
	return pooledDigest(data, sha1Pool), true, nil
}
