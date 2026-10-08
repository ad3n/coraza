// Copyright 2022 Juan Pablo Tosso and the OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

//go:build tinygo

package sync

func NewPool(new func() any) Pool {
	return &tinygoPool{
		new: new,
	}
}

// TinyGo is not concurrent, so we do not need a complicated implementation. We just want to reuse memory.
type tinygoPool struct {
	pool []any
	new  func() any
}

func (p *tinygoPool) Get() any {
	if len(p.pool) == 0 {
		return p.new()
	}

	last := len(p.pool) - 1
	x := p.pool[last]
	p.pool[last] = nil
	p.pool = p.pool[:last]
	return x
}

func (p *tinygoPool) Put(x any) {
	p.pool = append(p.pool, x)
}
