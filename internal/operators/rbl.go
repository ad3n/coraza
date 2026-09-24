// Copyright 2022 Juan Pablo Tosso and the OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

//go:build !tinygo && !coraza.disabled_operators.rbl

package operators

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/ad3n/coraza/v3/experimental/plugins/plugintypes"
	"github.com/ad3n/coraza/v3/internal/sync"
)

const timeout = 500 * time.Millisecond

// Description:
// Looks up the input IP address in the specified RBL (Real-time Block List) service.
// Performs DNS lookups to check if the IP is listed. Sets TX.httpbl_msg variable with
// the response text if found. Has a 500ms timeout for DNS queries.
//
// Arguments:
// RBL hostname to query (e.g., "sbl-xbl.spamhaus.org").
//
// Returns:
// true if the IP address is found in the RBL, false otherwise or on timeout
//
// Example:
// ```seclang
// # Check IP against Spamhaus blocklist
// SecRule REMOTE_ADDR "@rbl sbl-xbl.spamhaus.org" "id:183,deny,log,msg:'IP found in RBL'"
//
// # Multiple RBL checks
// SecRule REMOTE_ADDR "@rbl dnsbl.example.com" "id:184,deny"
// ```
type rbl struct {
	service    string
	resolver   *net.Resolver
	resultPool sync.Pool
}

type rblResult struct {
	txt     []string
	matched bool
}

var _ plugintypes.Operator = (*rbl)(nil)

func newRBL(options plugintypes.OperatorOptions) (plugintypes.Operator, error) {
	data := options.Arguments

	return &rbl{
		service:  data,
		resolver: net.DefaultResolver,
		resultPool: sync.NewPool(func() any {
			return make(chan rblResult, 1)
		}),
	}, nil
}

// https://github.com/mrichman/godnsbl
// https://github.com/SpiderLabs/ModSecurity/blob/b66224853b4e9d30e0a44d16b29d5ed3842a6b11/src/operators/rbl.cc
func (o *rbl) Evaluate(tx plugintypes.TransactionState, ipAddr string) bool {
	// TODO validate address
	resC := o.resultPool.Get().(chan rblResult)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	addr := fmt.Sprintf("%s.%s", ipAddr, o.service)
	go func(ctx context.Context) {
		res, err := o.resolver.LookupHost(ctx, addr)
		if err != nil {
			resC <- rblResult{}
			return
		}

		var txt []string
		if len(res) > 0 {
			txt, err = o.resolver.LookupTXT(ctx, addr)
			if err != nil {
				resC <- rblResult{}
				return
			}
		}

		resC <- rblResult{txt: txt, matched: true}
	}(ctx)

	select {
	case res := <-resC:
		o.resultPool.Put(resC)
		if res.matched && len(res.txt) > 0 {
			status := res.txt[0]
			tx.Variables().TX().Set("httpbl_msg", []string{status})
			tx.CaptureField(0, status)
		}

		return res.matched
	case <-time.After(timeout):
		return false
	}
}

func init() {
	Register("rbl", newRBL)
}
