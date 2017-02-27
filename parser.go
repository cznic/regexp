// Copyright 2017 The Regexp Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package regexp

import (
	"fmt"
	"path"
	"runtime"
	"unicode"
	"unicode/utf8"
)

const (
	eof = -(iota + 1)
	pastEOF
	bot
)

const (
	flagI = 1 << iota // Case-insensitive (default false).
	flagM             // Multi-line mode: ^ and $ match begin/end line in addition to begin/end text (default false).
	flagS             // Let . match \n (default false).
)

var (
	noOpt bool
)

type parser struct {
	c         rune
	pos       int
	sz        int
	re        *Regexp
	src       string
	flags     int
	flagStack []int
}

func newParser(src string, re *Regexp) *parser { return &parser{re: re, src: src} }

func (p *parser) reset() {
	p.pos = 0
	p.sz = 0
}

func (p *parser) todo() {
	_, fn, fl, _ := runtime.Caller(1)
	panic(fmt.Sprintf("%s:%d: `%s`:%d %q TODO\n", path.Base(fn), fl, p.src, p.pos, string(p.c))) //TODOOK
}

func (p *parser) n() rune {
	s := p.src
	if n := p.pos + p.sz; n < len(s) {
		p.pos = n
		p.c, p.sz = utf8.DecodeRuneInString(s[n:])
		return p.c
	}

	p.pos += p.sz
	p.sz = 0
	p.c = eof
	return eof
}

func (p *parser) patch(s, t int) { p.re.prog[s].patch(t) }

func (p *parser) parse() (_ *Regexp, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("error parsing regexp: %v", e)
		}
		p.re = nil
	}()

	p.n()
	in, out := p.expr(true)
	p.re.start = in
	p.re.accept = p.re.addState(instr{kind: opAccept})
	p.patch(out, p.re.accept)
	switch p.c {
	case eof:
		// ok
	default:
		panic(fmt.Sprintf("unexpected %c: `%s`", p.c, p.src))
	}
	find := p.re.addState(instr{kind: opDot})
	p.re.start1 = p.re.addState(instr{kind: opSplit, out: p.re.start, out1: find})
	p.patch(find, p.re.start1)
	re := p.re
	re.groups++
	p.re = nil
	return re.optimize(), nil
}

func (p *parser) expr(capturingGroup bool) (in, out int) {
	n := 2 * p.re.groups
	for in, out = p.term(capturingGroup); ; {
		switch p.c {
		case eof, ')':
			if capturingGroup {
				in = p.re.addState(instr{kind: opSave, arg: n, out: in})
				o := p.re.addState(instr{kind: opSave, arg: n + 1})
				p.patch(out, o)
				out = o
			}
			return in, out
		case '|':
			p.n()
			i, o := p.term(capturingGroup)
			a := p.re.addState(instr{kind: opSplit, out: in, out1: i})
			b := p.re.addState(instr{kind: opNop})
			p.patch(out, b)
			p.patch(o, b)
			in, out = a, b
		default:
			panic("internal error")
		}
	}
}

func (p *parser) term(capturingGroup bool) (in, out int) {
	for in, out = p.factor(capturingGroup); ; {
		switch p.c {
		case eof, ')', '|':
			return in, out
		}

		i, o := p.factor(capturingGroup)
		p.patch(out, i)
		out = o
	}
}

func (p *parser) factor(capturingGroup bool) (in, out int) {
	pos0 := p.pos
	switch p.c {
	case eof, ')', '|':
		in := p.re.addState(instr{kind: opNop})
		return in, in
	case '.':
		p.n()
		switch {
		case p.flags&flagS != 0:
			in = p.re.addState(instr{kind: opDotNL})
		default:
			in = p.re.addState(instr{kind: opDot})
		}
		out = in
	case '^':
		p.n()
		in = p.re.addState(instr{kind: opAssert, arg: assertBOT})
		out = in
	case '$':
		p.n()
		in = p.re.addState(instr{kind: opAssertEOT})
		out = in
	case '(':
		p.n()
		nm := ""
		restore := false
		switch p.c {
		case '?':
			capturingGroup = false
			switch p.n() {
			case 'P':
				switch p.n() {
				case '<':
					for {
						r := p.n()
						if r == '>' {
							break
						}

						nm += string(r)
					}
					capturingGroup = true
				default:
					p.todo()
				}
			default:
				restore = p.parseFlags()
			}
			fallthrough
		default:
			if capturingGroup {
				p.re.groups++
				p.re.groupNames = append(p.re.groupNames, nm)
			}
			in, out = p.expr(capturingGroup)
			if p.c == ')' {
				p.n()
				if restore {
					p.popFlags()
				}
				break
			}

			panic(fmt.Sprintf("missing closing ): `%s`", p.src))
		}
	case '[':
		p.n()
		in, out = p.set()
	case '\\':
		p.n()
		switch r := p.esc(); {
		case r < 0:
			in = p.re.addState(instr{kind: opAssert, arg: int(-r)})
		default:
			in = p.re.addState(instr{kind: opChar, arg: int(r)})
		}
		out = in
	case '*':
		panic("missing argument to repetition operator: `*`")
	case '+':
		panic("missing argument to repetition operator: `+`")
	case '?':
		panic("missing argument to repetition operator: `?`")
	case '{':
		p.todo()
	default:
		in = p.re.addState(instr{kind: opChar, arg: int(p.c)})
		out = in
		p.n()
	}

	for {
		switch p.c {
		case '*':
			switch p.n() {
			case '*':
				panic("invalid nested repetition operator: `**`")
			case '+':
				panic("invalid nested repetition operator: `*+`")
			}

			in, out = p.star(in, out, false) //TODO
		case '+':
			p.n()
			in, out = p.plus(in, out, false) //TODO
		case '?':
			p.n()
			in, out = p.opt(in, out, false) //TODO
		case '{':
			pos := p.pos
			p.n()
			n := p.num()
			comma := false
			m := -1
			if p.c == ',' {
				p.n()
				comma = true
				if p.c != '}' {
					m = p.num()
				}
			}
			if p.c != '}' {
				p.todo()
			}
			p.n()
			if n > maxRepCount || m >= 0 && (m < n || m > maxRepCount) {
				panic(fmt.Sprintf("invalid repeat count: `%s`", p.src[pos:p.pos]))
			}
			switch {
			case comma && m < 0: // {n,}
				switch n {
				case 0: // factor*
					in, out = p.star(in, out, false) //TODO
				case 1: // factor+
					in, out = p.plus(in, out, false) //TODO
				default:
					in, out = p.min(in, out, n, p.src[pos0:pos])
				}
			case comma: // {n,m}
				if m == 0 {
					in = p.re.addState(instr{kind: opNop})
					out = in
					break
				}

				if n == 0 {
					in, out = p.opt(in, out, false) //TODO
					in, out = p.max(in, out, n, m-1, p.src[pos0:pos])
					break
				}

				in, out = p.max(in, out, n, m, p.src[pos0:pos])
			default: // {n}
				switch n {
				case 0:
					in = p.re.addState(instr{kind: opNop})
					out = in
				case 1:
					// nop
				default:
					in, out = p.count(in, out, n, p.src[pos0:pos])
				}
			}
		default:
			return in, out
		}
	}
}

func (p *parser) pushFlags(newFlags int) {
	p.flagStack = append(p.flagStack, p.flags)
	p.flags = newFlags
}

func (p *parser) popFlags() {
	n := len(p.flagStack) - 1
	p.flags = p.flagStack[n]
	p.flagStack = p.flagStack[:n]
}

func (p *parser) parseFlags() (restore bool) {
	flags := p.flags
	minus := false
	for {
		switch p.c {
		case eof:
		case ')':
			p.flags = flags
			return false
		case '-':
			p.n()
			minus = true
		case 'i':
			p.n()
			switch {
			case minus:
				flags &^= flagI
			default:
				flags |= flagI
			}
		case 'm':
			p.n()
			switch {
			case minus:
				flags &^= flagM
			default:
				flags |= flagM
			}
		case 's':
			p.n()
			switch {
			case minus:
				flags &^= flagS
			default:
				flags |= flagS
			}
		case ':':
			p.n()
			p.pushFlags(flags)
			return true
		default:
			p.todo()
			return false
		}
	}
}

func (p *parser) max(in, out, n, m int, src string) (int, int) {
	q := newParser(src, p.re)
	for i := 0; i < n-1; i++ {
		q.n()
		a, b := q.factor(false)
		p.patch(out, a)
		out = b
		q.reset()
	}
	for i := 0; i < m-n; i++ {
		q.n()
		a, b := q.factor(false)
		a, b = p.opt(a, b, false) //TODO
		p.patch(out, a)
		out = b
		q.reset()
	}
	return in, out
}

func (p *parser) min(in, out, n int, src string) (int, int) {
	q := newParser(src, p.re)
	for i := 0; i < n-1; i++ {
		q.n()
		a, b := q.factor(false)
		if i == n-2 {
			a, b = p.plus(a, b, false) //TODO
		}
		p.patch(out, a)
		out = b
		q.reset()
	}
	return in, out
}

func (p *parser) count(in, out, n int, src string) (int, int) {
	q := newParser(src, p.re)
	for i := 0; i < n-1; i++ {
		q.n()
		a, b := q.factor(false)
		p.patch(out, a)
		out = b
		q.reset()
	}
	return in, out
}

func (p *parser) star(in, out int, nonGreedy bool) (int, int) {
	//
	//   /¯¯¯¯¯¯¯¯¯¯¯¯¯¯¯¯¯¯¯¯¯¯¯¯¯¯¯↘
	// (a)--->(in)-X-(out)--->(b)-e->(c)
	//           ↖____________/
	//
	c := p.re.addState(instr{kind: opNop})
	split := instr{kind: opSplit, out: in, out1: c}
	if nonGreedy {
		split.out, split.out1 = split.out1, split.out
	}
	b := p.re.addState(split)
	p.patch(out, b)
	a := p.re.addState(instr{kind: opSplit, out: in, out1: c})
	return a, c
}

func (p *parser) plus(in, out int, nonGreedy bool) (int, int) {
	//
	// (in)-X-(out)--->(a)-ε->(b)
	//    ↖____________/
	//
	b := p.re.addState(instr{kind: opNop})
	split := instr{kind: opSplit, out: in, out1: b}
	if nonGreedy {
		split.out, split.out1 = split.out1, split.out
	}
	a := p.re.addState(split)
	p.patch(out, a)
	return in, b
}

func (p *parser) opt(in, out int, nonGreedy bool) (int, int) {
	//
	//   /¯¯¯¯¯¯¯¯¯¯¯¯¯¯¯¯¯¯¯¯↘
	// (a)--->(in)-X-(out)--->(b)
	//
	b := p.re.addState(instr{kind: opNop})
	p.patch(out, b)
	split := instr{kind: opSplit, out: in, out1: b}
	if nonGreedy {
		split.out, split.out1 = split.out1, split.out
	}
	a := p.re.addState(split)
	return a, b
}

func (p *parser) num() (n int) {
	if p.c < '0' && p.c > '9' {
		p.todo()
	}
	for p.c >= '0' && p.c <= '9' {
		n = 10*n + int(p.c) - '0'
		if n < 0 {
			p.todo()
		}
		p.n()
	}
	return n
}

func (p *parser) set() (in, out int) {
	pos0 := p.pos - len("[")
	lo := len(p.re.regs)
	kind := opCharClass
	r := p.c
	if r == '^' {
		r = p.n()
		kind = opNotCharClass
	}
	for first := true; ; first = false {
	again:
		var r2 rune
		switch r {
		case eof:
			panic(fmt.Sprintf("missing closing ]: `%s`", p.src[pos0:]))
		case ']':
			if !first {
				p.n()
				in = p.re.addState(instr{kind: kind, arg: lo, arg2: len(p.re.regs)})
				return in, in
			}

			r2 = p.n()
		case '\\':
			p.n()
			r = p.esc()
			if r < 0 {
				p.re.regs = append(p.re.regs, int(r), 0)
				r2 = p.c
				goto again
			}
			r2 = p.c
		default:
			r2 = p.n()
		}

		// r is a single set member or possibly start of a range if followed by -.
		switch r2 {
		case eof:
			r = r2
		case ']':
			p.re.regs = append(p.re.regs, int(r), int(r))
			r = r2
		case '-':
			r4 := rune(-1)
			switch r3 := p.n(); r3 {
			case eof:
				r = r3
			case ']':
				p.re.regs = append(p.re.regs, int(r), int(r), '-', '-')
				r = r3
			case '\\':
				p.n()
				r3 = p.esc() //TODO handle r3 < 0
				r4 = p.c
				fallthrough
			default:
				if r > r3 {
					panic(fmt.Sprintf("invalid character class range: `%s-%s`", string(r), string(r3)))
				}

				p.re.regs = append(p.re.regs, int(r), int(r3))
				if r4 < 0 {
					r4 = p.n()
				}
				r = r4
			}
		default:
			p.re.regs = append(p.re.regs, int(r), int(r))
			r = r2
		}
	}
}

func (p *parser) esc() rune {
	switch c := p.c; c {
	case eof:
		panic("trailing backslash at end of expression: ``")
	case 'a':
		p.n()
		return '\a'
	case 'A':
		p.n()
		return -assertBOT
	case 'b':
		p.n()
		return -assertB
	case 'B':
		p.n()
		return -assertNotB
	case 'd':
		p.n()
		return -assertD
	case 'D':
		p.n()
		return -assertNotD
	case 'f':
		p.n()
		return '\f'
	case 'n':
		p.n()
		return '\n'
	case 'r':
		p.n()
		return '\r'
	case 's':
		p.n()
		return -assertS
	case 'S':
		p.n()
		return -assertNotS
	case 't':
		p.n()
		return '\t'
	case 'v':
		p.n()
		return '\v'
	case 'w':
		p.n()
		return -assertW
	case 'W':
		p.n()
		return -assertNotW
	case 'x':
		p.n()
		return p.escX()
	case 'z':
		p.n()
		return -assertEOT
	case '$', '^', '+', '<', '=', '>', '|', '~', '`':
		p.n()
		return c
	default:
		r := p.c
		if !unicode.IsPunct(r) {
			panic(fmt.Sprintf("invalid escape sequence: `\\%c`", r))
		}

		p.n()
		return r
	}
}

func (p *parser) escX() rune {
	switch p.c {
	case eof:
		panic("invalid escape sequence: `\\x`")
	default:
		p.todo()
		panic("TODO261")
	}
}
