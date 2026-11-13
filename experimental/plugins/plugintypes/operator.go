// Copyright 2022 Juan Pablo Tosso and the OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package plugintypes

import "io/fs"

// Memoizer caches the result of expensive function calls by key.
// Implementations must be safe for concurrent use.
type Memoizer interface {
	Do(key string, fn func() (any, error)) (any, error)
}

// OperatorOptions is used to store the options for a rule operator
type OperatorOptions struct {

	// Root is the root to resolve Path from.
	Root fs.FS

	// Memoizer caches expensive compilations (regex, aho-corasick).
	Memoizer Memoizer

	// Datasets contains input datasets or dictionaries.
	Datasets map[string][]string

	// Arguments stores the operator args.
	Arguments string

	// Path stores a list of possible data paths.
	Path []string

	// RxPreFilterEnabled controls whether the @rx operator uses
	// literal pre-filtering. Set by the SecRxPreFilter directive.
	RxPreFilterEnabled bool
}

// Operator interface is used to define rule @operators
type Operator interface {
	// Evaluate is used during the rule evaluation,
	// it returns true if the operator succeeded against
	// the input data for the transaction
	Evaluate(TransactionState, string) bool
}

type OperatorFactory func(options OperatorOptions) (Operator, error)
