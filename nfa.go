// Copyright 2017 The Regexp Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package regexp

import (
	"sync"
)

type instr struct {
	kind opcode
	arg  int
	arg2 int
	out  int
	out1 int // [Neg]Set: len(set)
}

func (s *instr) patch(t int) { s.out = t }

// Regexp is the representation of a compiled regular expression. A Regexp is
// safe for concurrent use by multiple goroutines.
type Regexp struct {
	accept     int
	complete   bool // Prefix is the whole re.
	groupNames []string
	groups     int
	longest    bool // See .Longest()
	longestMu  *sync.Mutex
	prefix     string // Any match must start with this literal.
	prog       []instr
	regs       []int
	src        string
	start      int // Full match.
	start1     int // Partial match.
}

func newRegexp(src string) *Regexp {
	return &Regexp{
		longestMu:  &sync.Mutex{},
		groupNames: []string{""},
		src:        src,
	}
}

func (re *Regexp) addState(s instr) int {
	if len(re.prog) > maxProg {
		panic("too many states")
	}

	re.prog = append(re.prog, s)
	return len(re.prog) - 1
}

func (re *Regexp) String() string { return re.src }

func (re *Regexp) route(s int) int {
	for re.prog[s].kind == opNop {
		s = re.prog[s].out
	}
	return s
}

func (re *Regexp) getPrefix() *Regexp {
	var r []rune
loop:
	for ip := re.start; ; {
		op := re.prog[ip]
		switch op.kind {
		case opAccept:
			re.prefix = string(r)
			re.complete = true
			return re
		case opChar:
			r = append(r, rune(op.arg))
		case
			opNop,
			opSave:

			// ok
		case
			opAssert,
			opCharClass,
			opDot,
			opDotNL,
			opNotCharClass,
			opSplit:

			break loop
		}
		ip = op.out
	}
	re.prefix = string(r)
	re.complete = false
	return re
}

func (re *Regexp) optimize() *Regexp {
	if noOpt {
		return re
	}

	for i := range re.prog {
		p := &re.prog[i]
		switch p.kind {
		case opAccept:
			// nop
		case
			opAssert,
			opChar,
			opCharClass,
			opDot,
			opDotNL,
			opNotCharClass,
			opNop,
			opSave:
			p.out = re.route(p.out)
		case
			opSplit:
			p.out = re.route(p.out)
			p.out1 = re.route(p.out1)
		default:
			panic("internal error")
		}
	}
	re.start = re.route(re.start)
	re.start1 = re.route(re.start1)
	return re
}

func (re *Regexp) reachable(in, out int) []int {
	set := newThreadList(len(re.prog))
	var f func(int)
	f = func(s int) {
		if set.has(s) {
			return
		}

		set.include(thread{pc: s})
		if s == out {
			return
		}

		switch p := &re.prog[s]; p.kind {
		case
			opAssert,
			opChar,
			opCharClass,
			opDot,
			opDotNL,
			opNotCharClass,
			opNop,
			opSave:
			f(p.out)
		case
			opSplit:
			f(p.out)
			f(p.out1)
		case opAccept:
			// nop
		default:
			panic("internal error")
		}
	}
	f(in)
	r := make([]int, len(set.dense))
	for i := range set.dense {
		r[i] = set.dense[i].pc
	}
	return r
}
