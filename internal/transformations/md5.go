// Copyright 2022 Juan Pablo Tosso and the OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package transformations

import (
	"crypto/fips140"
	"crypto/md5"
	"errors"

	"github.com/ad3n/coraza/v3/internal/sync"
)

const emptyMD5 = "\xd4\x1d\x8c\xd9\x8f\x00\xb2\x04\xe9\x80\x09\x98\xec\xf8\x42\x7e"

var (
	errMD5NotAvailableFIPS = errors.New("md5 transformation is unavailable in FIPS 140-3 mode")
	md5Pool                = sync.NewPool(func() any {
		return &digestState{hash: md5.New()}
	})
)

func md5T(data string) (string, bool, error) {
	if fips140.Enabled() {
		return data, false, errMD5NotAvailableFIPS
	}

	if len(data) == 0 {
		return emptyMD5, true, nil
	}

	return pooledDigest(data, md5Pool), true, nil
}
