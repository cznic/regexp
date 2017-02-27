// Copyright 2017 The Regexp Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package regexp

import (
	"io"
)

type submatches struct {
	sub []int
}

func (s *submatches) update(i, pos, nsub int) submatches {
	sub := make([]int, nsub)
	for i := range sub {
		sub[i] = -1
	}
	copy(sub, s.sub)
	sub[i] = pos
	s.sub = sub
	return *s
}

type thread struct {
	pc    int
	saved submatches
}

type threadList struct {
	dense  []thread
	sparse []int
	len    int
	match  bool
}

func newThreadList(len int) *threadList {
	return &threadList{
		dense:  make([]thread, len),
		sparse: make([]int, len),
	}
}

func (l *threadList) include(t thread) *thread {
	l.dense[l.len] = t
	p := &l.dense[l.len]
	l.sparse[t.pc] = l.len
	l.len++
	return p
}

func (l *threadList) has(pc int) bool {
	i := l.sparse[pc]
	return i < l.len && l.dense[i].pc == pc
}

type vm struct {
	re     *Regexp
	r      io.RuneReader
	saved  []int
	pos    int
	sz     int
	last   rune
	c      rune
	first  bool
	closed bool
}

func newVM(re *Regexp, r io.RuneReader) *vm {
	vm := &vm{
		re: re,
		r:  r,
	}
	vm.c, vm.sz = vm.readRune()
	vm.last = bot
	vm.pos = 0
	vm.first = true
	return vm
}

func (vm *vm) readRune() (rune, int) {
	if vm.closed {
		return pastEOF, 0
	}

	r, sz, err := vm.r.ReadRune()
	if err != nil {
		r = eof
		sz = 0
		vm.closed = true
	}
	return r, sz
}

func (vm *vm) next() rune {
	vm.last = vm.c
	vm.pos += vm.sz
	vm.c, vm.sz = vm.readRune()
	return vm.c
}

func (vm *vm) fullMatch() bool {
	a := vm.find()
	return (vm.c == eof || vm.c == pastEOF) && a != nil && a[0] == 0 && a[1] == vm.pos
}

func (vm *vm) match() bool {
	clist := newThreadList(len(vm.re.prog))
	nlist := newThreadList(len(vm.re.prog))
	vm.addThread(clist, thread{pc: vm.re.start1}, 0)
	for vm.first = false; !clist.match && clist.len != 0; clist, nlist = nlist, clist {
		vm.step(clist, nlist)
		vm.next()
	}
	return clist.match
}

func (vm *vm) find() []int {
	clist := newThreadList(len(vm.re.prog))
	nlist := newThreadList(len(vm.re.prog))
	vm.saved = nil
	vm.addThread(clist, thread{pc: vm.re.start1}, vm.pos)
	for vm.first = false; clist.len != 0; clist, nlist = nlist, clist {
		vm.step(clist, nlist)
		if vm.c != eof && clist.match && !nlist.match {
			break
		}

		if vm.saved != nil && vm.saved[0] == vm.saved[1] {
			vm.next()
			break
		}

		vm.next()
	}
	return vm.saved
}

func (vm *vm) step(clist *threadList, nlist *threadList) {
	nlist.len = 0
	nlist.match = false
	for i := 0; i < clist.len; i++ {
		if nlist.match {
			break
		}

		t := &clist.dense[i]
		switch op := &vm.re.prog[t.pc]; op.kind {
		case opAccept:
			// nop
		case opAssert:
			switch op.arg {
			case assertB, assertNotB, assertBOT, assertEOT:
				if asserts[op.arg](vm.first, vm.last, vm.c) {
					vm.addThread(nlist, thread{op.out, t.saved}, vm.pos)
				}
			}
		case opAssertEOT:
			if vm.c == eof {
				vm.addThread(nlist, thread{op.out, t.saved}, vm.pos)
			}
		case opChar:
			if vm.c == rune(op.arg) {
				vm.addThread(nlist, thread{op.out, t.saved}, vm.pos+vm.sz)
			}
		case opCharClass:
			if vm.set(vm.re.regs[op.arg:op.arg2]) {
				vm.addThread(nlist, thread{op.out, t.saved}, vm.pos+vm.sz)
			}
		case opDot:
			if vm.c != '\n' && vm.c != eof {
				vm.addThread(nlist, thread{op.out, t.saved}, vm.pos+vm.sz)
			}
		case opDotNL:
			if vm.c != eof {
				vm.addThread(nlist, thread{op.out, t.saved}, vm.pos+vm.sz)
			}
		case opNop:
			if noOpt {
				vm.addThread(nlist, thread{op.out, t.saved}, vm.pos)
				break
			}

			panic("internal error")
		case opNotCharClass:
			if !vm.set(vm.re.regs[op.arg:op.arg2]) && vm.c != eof {
				vm.addThread(nlist, thread{op.out, t.saved}, vm.pos+vm.sz)
			}
		case opSave:
			// nop
		case opSplit:
			// nop
		default:
			panic(op.kind)
		}
	}
}

func (vm *vm) addThread(list *threadList, t thread, pos int) {
	if list.has(t.pc) {
		return
	}

	list.include(t)
	switch op := &vm.re.prog[t.pc]; op.kind {
	case opAccept:
		if sub := t.saved.sub; vm.saved == nil || vm.saved[0] == sub[0] {
			vm.saved = sub
		}
		list.match = true
	case opAssert:
		if asserts[op.arg](vm.first, vm.last, vm.c) {
			vm.addThread(list, thread{op.out, t.saved}, pos)
		}
	case opAssertEOT:
		// nop
	case opChar:
		// nop
	case opCharClass:
		// nop
	case opDot:
		// nop
	case opDotNL:
		// nop
	case opNop:
		if noOpt {
			vm.addThread(list, thread{op.out, t.saved}, pos)
			break
		}

		panic("internal error")
	case opNotCharClass:
		// nop
	case opSave:
		vm.addThread(list, thread{op.out, t.saved.update(op.arg, pos, 2*vm.re.groups)}, pos)
	case opSplit:
		vm.addThread(list, thread{op.out, t.saved}, pos)
		vm.addThread(list, thread{op.out1, t.saved}, pos)
	default:
		panic(op.kind)
	}
}

func (vm *vm) set(ranges []int) bool {
	for i := 0; i < len(ranges); i += 2 {
		lo := ranges[i]
		if lo < 0 {
			if asserts[-lo](vm.first, vm.last, vm.c) {
				return true
			}

			continue
		}

		if r := vm.c; r >= rune(lo) && r <= rune(ranges[i+1]) {
			return true
		}
	}
	return false
}

const (
	_ = iota // Values must be non-zero.
	assertB
	assertBOT
	assertD
	assertEOT
	assertNotB
	assertNotD
	assertNotS
	assertNotW
	assertS
	assertW
)

var (
	asserts = map[int]func(bool, rune, rune) bool{
		assertB:    isB,
		assertBOT:  isBOT,
		assertD:    isD,
		assertEOT:  isEOT,
		assertNotB: isNotB,
		assertNotD: isNotD,
		assertNotS: isNotS,
		assertNotW: isNotW,
		assertS:    isS,
		assertW:    isW,
	}

	assertString = map[int]string{
		assertB:    "\\b",
		assertBOT:  "\\A",
		assertD:    "\\d",
		assertEOT:  "\\z",
		assertNotB: "\\B",
		assertNotD: "\\D",
		assertNotS: "\\S",
		assertNotW: "\\W",
		assertS:    "\\s",
		assertW:    "\\w",
	}
)

func isB(_ bool, last, c rune) (r bool) {
	return isW(false, -1, last) && (isEOT(false, -1, c) || isNotW(false, -1, c)) ||
		isW(false, -1, c) && (last == bot || isNotW(false, -1, last))
}

func isNotB(first bool, last, c rune) bool {
	return !isB(false, last, c)
}

func isBOT(first bool, _, _ rune) bool { return first }

func isD(_ bool, _, c rune) bool    { return c >= '0' && c <= '9' }
func isNotD(_ bool, _, c rune) bool { return !isD(false, -1, c) }

func isEOT(_ bool, _, c rune) bool { return c == eof }

func isS(_ bool, _, c rune) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\f' || c == '\r'
}
func isNotS(_ bool, _, c rune) bool { return !isS(false, -1, c) }

func isW(_ bool, _, c rune) bool {
	return c >= '0' && c <= '9' || c >= 'A' && c <= 'Z' || c >= 'c' && c <= 'z' || c == '_'
}

func isNotW(_ bool, _, c rune) bool { return !isW(false, -1, c) }
