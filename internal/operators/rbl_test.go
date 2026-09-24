// Copyright 2022 Juan Pablo Tosso and the OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

//go:build !tinygo

package operators

import (
	"context"
	"io"
	"log"
	"net"
	"testing"
	"testing/synctest"
	"time"

	"github.com/foxcpp/go-mockdns"

	"github.com/ad3n/coraza/v3/experimental/plugins/plugintypes"
	"github.com/ad3n/coraza/v3/internal/corazawaf"
)

type testLogger struct{ t *testing.T }

func (l *testLogger) Printf(format string, v ...any) {
	l.t.Helper()
	l.t.Logf(format, v...)
}

func TestRbl(t *testing.T) {
	opts := plugintypes.OperatorOptions{
		Arguments: "xbl.spamhaus.org",
	}
	op, err := newRBL(opts)
	if err != nil {
		t.Fatal("Cannot init rbl operator")
	}

	logger := &testLogger{t}

	srv, err := mockdns.NewServerWithLogger(map[string]mockdns.Zone{
		"valid_no_txt.xbl.spamhaus.org.": {
			A: []string{"1.2.3.4"},
		},
		"valid_txt.xbl.spamhaus.org.": {
			A:   []string{"1.2.3.5"},
			TXT: []string{"not blocked"},
		},
		"blocked.xbl.spamhaus.org.": {
			A:   []string{"1.2.3.6"},
			TXT: []string{"blocked"},
		},
	}, logger, false)
	if err != nil {
		t.Fatalf("Cannot start mockdns server: %v", err)
	}
	defer srv.Close()

	srv.PatchNet(op.(*rbl).resolver)
	defer mockdns.UnpatchNet(op.(*rbl).resolver)

	t.Run("Valid hostname with no TXT record", func(t *testing.T) {
		if op.Evaluate(nil, "valid_no_txt") {
			t.Errorf("Unexpected result for valid hostname with no TXT record")
		}
	})

	t.Run("Valid hostname with TXT record", func(t *testing.T) {
		tx := corazawaf.NewWAF().NewTransaction()
		if !op.Evaluate(tx, "valid_txt") {
			t.Errorf("Unexpected result for valid hostname")
		}
		if want, have := "not blocked", tx.Variables().TX().Get("httpbl_msg")[0]; want != have {
			t.Errorf("Unexpected result for valid hostname: want %q, have %q", want, have)
		}
	})

	t.Run("Invalid hostname", func(t *testing.T) {
		if op.Evaluate(nil, "invalid") {
			t.Errorf("Unexpected result for invalid hostname")
		}
	})

	t.Run("Blocked hostname", func(t *testing.T) {
		tx := corazawaf.NewWAF().NewTransaction()
		tx.Capture = true
		if !op.Evaluate(tx, "blocked") {
			t.Fatal("Unexpected result for blocked hostname")
		}
		t.Log(tx.Variables().TX().Get("httpbl_msg"))
		if want, have := "blocked", tx.Variables().TX().Get("httpbl_msg")[0]; want != have {
			t.Errorf("Unexpected result for valid hostname: want %q, have %q", want, have)
		}

		if want, have := "blocked", tx.Variables().TX().Get("0")[0]; want != have {
			t.Errorf("Unexpected capture for blocked hostname: want %q, have %q", want, have)
		}
	})

	t.Run("Timeout releases DNS worker", func(t *testing.T) {
		if op.Evaluate(nil, "invalid") {
			t.Fatal("unexpected match while initializing the DNS resolver")
		}

		synctest.Test(t, func(t *testing.T) {
			op, err := newRBL(plugintypes.OperatorOptions{Arguments: "timeout.invalid"})
			if err != nil {
				t.Fatal(err)
			}

			op.(*rbl).resolver = &net.Resolver{
				PreferGo: true,
				Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
					<-ctx.Done()
					return nil, ctx.Err()
				},
			}

			for range 3 {
				started := time.Now()
				if op.Evaluate(nil, "timeout") {
					t.Fatal("unexpected match after DNS timeout")
				}

				if elapsed := time.Since(started); elapsed != timeout {
					t.Fatalf("timeout changed: want %s, got %s", timeout, elapsed)
				}

				synctest.Wait()
			}
		})
	})

	t.Run("Concurrent results stay isolated", func(t *testing.T) {
		waf := corazawaf.NewWAF()
		t.Cleanup(func() {
			if err := waf.Close(); err != nil {
				t.Error(err)
			}
		})

		for _, tc := range []struct {
			input   string
			message string
			matched bool
		}{
			{input: "valid_txt", message: "not blocked", matched: true},
			{input: "blocked", message: "blocked", matched: true},
			{input: "valid_no_txt", message: "unchanged"},
			{input: "invalid", message: "unchanged"},
		} {
			t.Run(tc.input, func(t *testing.T) {
				t.Parallel()
				tx := waf.NewTransaction()
				t.Cleanup(func() {
					if err := tx.Close(); err != nil {
						t.Error(err)
					}
				})
				tx.Capture = true
				for range 16 {
					tx.Variables().TX().Set("httpbl_msg", []string{"unchanged"})
					tx.Variables().TX().Set("0", []string{"unchanged"})
					if got := op.Evaluate(tx, tc.input); got != tc.matched {
						t.Fatalf("unexpected match: want %t, got %t", tc.matched, got)
					}

					for _, key := range []string{"httpbl_msg", "0"} {
						if got := tx.Variables().TX().Get(key)[0]; got != tc.message {
							t.Fatalf("unexpected %s: want %q, got %q", key, tc.message, got)
						}
					}
				}
			})
		}
	})
}

func BenchmarkRBL(b *testing.B) {
	srv, err := mockdns.NewServerWithLogger(map[string]mockdns.Zone{
		"hit.bench.invalid.": {
			A:   []string{"127.0.0.2"},
			TXT: []string{"listed"},
		},
	}, log.New(io.Discard, "", 0), false)
	if err != nil {
		b.Fatal(err)
	}
	defer srv.Close()

	op, err := newRBL(plugintypes.OperatorOptions{Arguments: "bench.invalid"})
	if err != nil {
		b.Fatal(err)
	}

	resolver := &net.Resolver{}
	srv.PatchNet(resolver)
	op.(*rbl).resolver = resolver
	waf := corazawaf.NewWAF()
	defer waf.Close()

	for _, tc := range []struct {
		name    string
		input   string
		matched bool
	}{
		{name: "LookupTXT", input: "hit", matched: true},
		{name: "InvalidHostname", input: "invalid..host"},
	} {
		b.Run(tc.name, func(b *testing.B) {
			tx := waf.NewTransaction()
			defer tx.Close()

			b.ReportAllocs()
			for b.Loop() {
				if got := op.Evaluate(tx, tc.input); got != tc.matched {
					b.Fatalf("unexpected match: want %t, got %t", tc.matched, got)
				}
			}
		})
	}
}
