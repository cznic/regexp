// Copyright 2017 The Regexp Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package regexp

import (
	"flag"
	"fmt"
	"os"
	"path"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func caller(s string, va ...interface{}) {
	if s == "" {
		s = strings.Repeat("%v ", len(va))
	}
	_, fn, fl, _ := runtime.Caller(2)
	fmt.Fprintf(os.Stderr, "# caller: %s:%d: ", path.Base(fn), fl)
	fmt.Fprintf(os.Stderr, s, va...)
	fmt.Fprintln(os.Stderr)
	_, fn, fl, _ = runtime.Caller(1)
	fmt.Fprintf(os.Stderr, "# \tcallee: %s:%d: ", path.Base(fn), fl)
	fmt.Fprintln(os.Stderr)
	os.Stderr.Sync()
}

func dbg(s string, va ...interface{}) {
	if s == "" {
		s = strings.Repeat("%v ", len(va))
	}
	_, fn, fl, _ := runtime.Caller(1)
	fmt.Fprintf(os.Stderr, "# dbg %s:%d: ", path.Base(fn), fl)
	fmt.Fprintf(os.Stderr, s, va...)
	fmt.Fprintln(os.Stderr)
	os.Stderr.Sync()
}

func TODO(...interface{}) string { //TODOOK
	_, fn, fl, _ := runtime.Caller(1)
	return fmt.Sprintf("# TODO: %s:%d:\n", path.Base(fn), fl) //TODOOK
}

func use(...interface{}) {}

func init() {
	use(caller, dbg, TODO) //TODOOK
}

// ============================================================================

var (
	oCase = flag.Int("case", -1, "")

	goodRe2 = []string{
		`()`,
		`(a*|b)(c*|d)`,
		`(a|)`,
		`(a|b)`,
		`(|)`,
		`(|b)`,
		`.*a`,
		`.`,
		`[--z-a]`,
		`[-]`,
		`[-a-]`,
		`[-a-a-]`,
		`[-a-a-z]`,
		`[-a\-a-z]`,
		`[-a]`,
		`[.]`,
		`[A-\]]`,
		`[\\]`,
		`[\a-\]]`,
		`[\a-\f\]]`,
		`[\a-\f]`,
		`[\a-]`,
		`[\a-a\]]`,
		`[\a-z]`,
		`[\a\-\]]`,
		`[\a\-\f\]]`,
		`[\a\-\f]`,
		`[\a\-]`,
		`[\a\-z\]]`,
		`[\a\-z]`,
		`[\a\]]`,
		`[\a]`,
		`[]]`,
		`[^--z-a]`,
		`[^-]`,
		`[^-a-]`,
		`[^-a-a-]`,
		`[^-a-a-z]`,
		`[^-a-a\-z]`,
		`[^-a]`,
		`[^1234]`,
		`[^\n]`,
		`[^^1234]`,
		`[^^\n]`,
		`[^a-abc-c\-\]\[]`,
		`[^a-z]+`,
		`[^a-z]`,
		`[^abc]`,
		`[a-\{\]]`,
		`[a-\{]`,
		`[a-]`,
		`[a-abc-c\-\]\[]`,
		`[a-z\]]`,
		`[a-z]+`,
		`[a-z]`,
		`[a\--z]`,
		`[a\-\]]`,
		`[a\-\{\]]`,
		`[a\-\{]`,
		`[a\-]`,
		`[a\-z\]]`,
		`[a\-z]`,
		`[a\]]`,
		`[a]`,
		`[ab]`,
		`[abc]`,
		`\!\\`,
		`^.$`,
		``,
		`a*`,
		`a*b`,
		`a*|b*`,
		`a+`,
		`a?`,
		`a`,
		`ab*`,
		`ab`,
		`ab|c`,
		`ab|cd`,
		`a{1000,1000}`,
		`a{1000,}`,
		`a{1000}`,
		`a|b`,
		`a|bc`,
		`a|bc|c`,
		`a|b|c`,
		`|`,
	}

	badRe2 = []string{
		`(abc`,
		`*`,
		`+`,
		`?`,
		`[---z-a]`,
		`[-z-a]`,
		`[^---z-a]`,
		`[^-z-a]`,
		`[^z-a]`,
		`[`,
		`[a-`,
		`[a-z`,
		`[a`,
		`[z-a]`,
		`\x`,
		`a(b`,
		`a**`,
		`a*+`,
		`abc)`,
		`abc\`,
		`a{1000,1001}`,
		`a{1001,1000}`,
		`a{1001,}`,
		`a{1001}`,
		`a{2,1}`,
		`x[^a-z`,
		`x[a-z`,
	}

	simpleTests = []struct {
		re, src string
	}{

		{`$*`, ``},
		{`$`, ``},
		{`$`, `a`},
		{`$a`, `$`},
		{`$a`, `$a`},
		{`$a`, ``},
		{`$a`, `a`},
		{`.*a.{3}bc`, ``},
		{`.*a.{3}bc`, `axaybzbc`},
		{`.*a.{3}bc`, `axaybzbd`},
		{`.*a.{3}bc`, `axxbc`},
		{`.*a.{3}bc`, `axxxbc`},
		{`.`, "\n"},
		{`.`, ``},
		{`.`, `a`},
		{`.`, `b`},
		{`.`, `c`},
		{`.a`, ``},
		{`.a`, `a`},
		{`.a`, `aa`},
		{`.a`, `aaa`},
		{`.a`, `aab`},
		{`.a`, `ab`},
		{`.a`, `b`},
		{`[-]`, `-`},
		{`[-]`, ``},
		{`[-]`, `a`},
		{`[-]`, `b`},
		{`[-a]`, `-`},
		{`[-a]`, ``},
		{`[-a]`, `a`},
		{`[-a]`, `b`},
		{`[0-9a-f]+`, `13f`},
		{`[0-9a-f]+`, `PQ`},
		{`[0-9a-f]+`, `x13fz`},
		{`[0-9a-f]`, `13f`},
		{`[0-9a-f]`, `PQ`},
		{`[0-9a-f]`, ``},
		{`[0-9a-f]`, `x13fz`},
		{`[^-]`, `-`},
		{`[^-]`, ``},
		{`[^-]`, `a`},
		{`[^-]`, `b`},
		{`[^-a]`, `-`},
		{`[^-a]`, ``},
		{`[^-a]`, `a`},
		{`[^-a]`, `b`},
		{`[^0-9a-f]+`, `13f`},
		{`[^0-9a-f]+`, `PQ`},
		{`[^0-9a-f]+`, `x13fz`},
		{`[^0-9a-f]`, `13f`},
		{`[^0-9a-f]`, `PQ`},
		{`[^0-9a-f]`, ``},
		{`[^0-9a-f]`, `x13fz`},
		{`[^a-]`, `-`},
		{`[^a-]`, ``},
		{`[^a-]`, `a`},
		{`[^a-]`, `b`},
		{`[a-]`, `-`},
		{`[a-]`, ``},
		{`[a-]`, `a`},
		{`[a-]`, `b`},
		{`\$`, ``},
		{`\$`, `a`},
		{`\$a`, `$`},
		{`\$a`, `$a`},
		{`\$a`, ``},
		{`\$a`, `a`},
		{`\*`, `*`},
		{`\*`, ``},
		{`\*`, `a`},
		{`\^`, ``},
		{`\^a$`, `a`},
		{`\^a$`, `b`},
		{`\^a$`, `ba`},
		{`\^a$`, `bac`},
		{`\^a\$`, `a`},
		{`\^a\$`, `b`},
		{`\^a\$`, `ba`},
		{`\^a\$`, `bac`},
		{`\^a`, `a`},
		{`\^a`, `aa`},
		{`\^a`, `aaa`},
		{`\^a`, `ab`},
		{`\^a`, `b`},
		{`\^a`, `ba`},
		{`\^a`, `bac`},
		{`^*`, ``},
		{`^[a-z]+\[[0-9]+\]$`, `Job[48]`},
		{`^[a-z]+\[[0-9]+\]$`, `adam[23]`},
		{`^[a-z]+\[[0-9]+\]$`, `eve[7]`},
		{`^[a-z]+\[[0-9]+\]$`, `snakey`},
		{`^`, ``},
		{`^a$`, `a`},
		{`^a$`, `b`},
		{`^a$`, `ba`},
		{`^a$`, `bac`},
		{`^a\$`, `a`},
		{`^a\$`, `b`},
		{`^a\$`, `ba`},
		{`^a\$`, `bac`},
		{`^a`, `a`},
		{`^a`, `aa`},
		{`^a`, `aaa`},
		{`^a`, `ab`},
		{`^a`, `b`},
		{`^a`, `ba`},
		{`^a`, `bac`},
		{`a$`, `a`},
		{`a$`, `ab`},
		{`a$`, `b`},
		{`a$`, `ba`},
		{`a$`, `bac`},
		{`a*$`, ``},
		{`a*$`, `a`},
		{`a*$`, `aa`},
		{`a*$`, `ab`},
		{`a*\$`, ``},
		{`a*\$`, `a`},
		{`a*\$`, `aa`},
		{`a*\$`, `ab`},
		{`a*`, ``},
		{`a*`, `a`},
		{`a*`, `aa`},
		{`a*`, `aaa`},
		{`a*`, `aaab`},
		{`a*`, `aab`},
		{`a*`, `ab`},
		{`a*`, `b`},
		{`a*`, `baac`},
		{`a*`, `bac`},
		{`a*b$`, ``},
		{`a*b$`, `a`},
		{`a*b$`, `aab`},
		{`a*b$`, `aac`},
		{`a*b$`, `ab`},
		{`a*b$`, `ac`},
		{`a*b$`, `b`},
		{`a*b$`, `c`},
		{`a*b\$`, ``},
		{`a*b\$`, `a`},
		{`a*b\$`, `aab`},
		{`a*b\$`, `aac`},
		{`a*b\$`, `ab`},
		{`a*b\$`, `ac`},
		{`a*b\$`, `b`},
		{`a*b\$`, `c`},
		{`a*b`, ``},
		{`a*b`, `a`},
		{`a*b`, `aa`},
		{`a*b`, `aab`},
		{`a*b`, `aac`},
		{`a*b`, `ab`},
		{`a*b`, `ac`},
		{`a*b`, `b`},
		{`a*b`, `c`},
		{`a+`, ``},
		{`a+`, `a`},
		{`a+`, `aa`},
		{`a+`, `aab`},
		{`a+`, `b`},
		{`a.`, "a\n"},
		{`a.`, ``},
		{`a.`, `a`},
		{`a.`, `aa`},
		{`a.`, `ab`},
		{`a.`, `ac`},
		{`a.`, `b`},
		{`a?`, ``},
		{`a?`, `a`},
		{`a?`, `aa`},
		{`a?`, `ab`},
		{`a?`, `c`},
		{`a?b`, ``},
		{`a?b`, `a`},
		{`a?b`, `ab`},
		{`a?b`, `b`},
		{`a?b`, `ba`},
		{`aX{3}c`, `aXXaXXXc`},
		{`aX{3}c`, `aXXaXc`},
		{`a\$`, `a`},
		{`a\$`, `ab`},
		{`a\$`, `b`},
		{`a\$`, `ba`},
		{`a\$`, `bac`},
		{`a\^`, ``},
		{`a\^`, `a^`},
		{`a\^`, `a^a`},
		{`a\^`, `a^b`},
		{`a\^`, `a`},
		{`a\^`, `b^`},
		{`a\^`, `b`},
		{`a^`, ``},
		{`a^`, `a^`},
		{`a^`, `a^a`},
		{`a^`, `a^b`},
		{`a^`, `a`},
		{`a^`, `b^`},
		{`a^`, `b`},
		{`a`, ``},
		{`a`, `a`},
		{`a`, `ab`},
		{`a`, `b`},
		{`a`, `ba`},
		{`a`, `bac`},
		{`ab+`, ``},
		{`ab+`, `a`},
		{`ab+`, `ab`},
		{`ab+`, `abb`},
		{`ab+`, `abbc`},
		{`ab+`, `abc`},
		{`ab+c`, ``},
		{`ab+c`, `a`},
		{`ab+c`, `abbc`},
		{`ab+c`, `abbd`},
		{`ab+c`, `abc`},
		{`ab+c`, `abd`},
		{`ab+c`, `ac`},
		{`ab+c`, `adc`},
		{`ab`, `aa`},
		{`ab`, `ab`},
		{`ab`, `ac`},
		{`a{0,0}`, ``},
		{`a{0,0}`, `a`},
		{`a{0,0}`, `aac`},
		{`a{0,0}`, `ac`},
		{`a{0,0}`, `b`},
		{`a{0,0}`, `ba`},
		{`a{0,0}`, `bac`},
		{`a{0,0}`, `bc`},
		{`a{0,0}`, `c`},
		{`a{0,1}`, ``},
		{`a{0,1}`, `a`},
		{`a{0,1}`, `aac`},
		{`a{0,1}`, `ac`},
		{`a{0,1}`, `b`},
		{`a{0,1}`, `ba`},
		{`a{0,1}`, `bac`},
		{`a{0,1}`, `bc`},
		{`a{0,1}`, `c`},
		{`a{0,2}`, ``},
		{`a{0,2}`, `a`},
		{`a{0,2}`, `aa`},
		{`a{0,2}`, `aaa`},
		{`a{0,2}`, `aaac`},
		{`a{0,2}`, `aac`},
		{`a{0,2}`, `ac`},
		{`a{0,2}`, `b`},
		{`a{0,2}`, `ba`},
		{`a{0,2}`, `baa`},
		{`a{0,2}`, `baaa`},
		{`a{0,2}`, `baaac`},
		{`a{0,2}`, `baac`},
		{`a{0,2}`, `bac`},
		{`a{0,2}`, `bc`},
		{`a{0,2}`, `c`},
		{`a{0,}`, ``},
		{`a{0,}`, `a`},
		{`a{0,}`, `aac`},
		{`a{0,}`, `ac`},
		{`a{0,}`, `b`},
		{`a{0,}`, `ba`},
		{`a{0,}`, `bac`},
		{`a{0,}`, `bc`},
		{`a{0,}`, `c`},
		{`a{0}`, ``},
		{`a{0}`, `a`},
		{`a{0}`, `aac`},
		{`a{0}`, `ac`},
		{`a{0}`, `b`},
		{`a{0}`, `ba`},
		{`a{0}`, `bac`},
		{`a{0}`, `bc`},
		{`a{0}`, `c`},
		{`a{1,1}`, ``},
		{`a{1,1}`, `a`},
		{`a{1,1}`, `aa`},
		{`a{1,1}`, `aaa`},
		{`a{1,1}`, `aaac`},
		{`a{1,1}`, `aac`},
		{`a{1,1}`, `ac`},
		{`a{1,1}`, `b`},
		{`a{1,1}`, `ba`},
		{`a{1,1}`, `baa`},
		{`a{1,1}`, `baaa`},
		{`a{1,1}`, `baaac`},
		{`a{1,1}`, `baac`},
		{`a{1,1}`, `bac`},
		{`a{1,1}`, `bc`},
		{`a{1,1}`, `c`},
		{`a{1,2}`, ``},
		{`a{1,2}`, `a`},
		{`a{1,2}`, `aa`},
		{`a{1,2}`, `aaa`},
		{`a{1,2}`, `aaac`},
		{`a{1,2}`, `aac`},
		{`a{1,2}`, `ac`},
		{`a{1,2}`, `b`},
		{`a{1,2}`, `ba`},
		{`a{1,2}`, `baa`},
		{`a{1,2}`, `baaa`},
		{`a{1,2}`, `baaac`},
		{`a{1,2}`, `baac`},
		{`a{1,2}`, `bac`},
		{`a{1,2}`, `bc`},
		{`a{1,2}`, `c`},
		{`a{1,}`, ``},
		{`a{1,}`, `a`},
		{`a{1,}`, `aac`},
		{`a{1,}`, `ac`},
		{`a{1,}`, `b`},
		{`a{1,}`, `ba`},
		{`a{1,}`, `bac`},
		{`a{1,}`, `bc`},
		{`a{1,}`, `c`},
		{`a{1}`, ``},
		{`a{1}`, `a`},
		{`a{1}`, `aac`},
		{`a{1}`, `ac`},
		{`a{1}`, `b`},
		{`a{1}`, `ba`},
		{`a{1}`, `bac`},
		{`a{1}`, `bc`},
		{`a{1}`, `c`},
		{`a{2,2}`, ``},
		{`a{2,2}`, `a`},
		{`a{2,2}`, `aa`},
		{`a{2,2}`, `aaa`},
		{`a{2,2}`, `aaac`},
		{`a{2,2}`, `aac`},
		{`a{2,2}`, `ac`},
		{`a{2,2}`, `b`},
		{`a{2,2}`, `ba`},
		{`a{2,2}`, `baa`},
		{`a{2,2}`, `baaa`},
		{`a{2,2}`, `baaac`},
		{`a{2,2}`, `baac`},
		{`a{2,2}`, `bac`},
		{`a{2,2}`, `bc`},
		{`a{2,2}`, `c`},
		{`a{2,3}`, ``},
		{`a{2,3}`, `a`},
		{`a{2,3}`, `aa`},
		{`a{2,3}`, `aaa`},
		{`a{2,3}`, `aaaa`},
		{`a{2,3}`, `aaaac`},
		{`a{2,3}`, `aaac`},
		{`a{2,3}`, `aac`},
		{`a{2,3}`, `ac`},
		{`a{2,3}`, `b`},
		{`a{2,3}`, `ba`},
		{`a{2,3}`, `baa`},
		{`a{2,3}`, `baaa`},
		{`a{2,3}`, `baaaa`},
		{`a{2,3}`, `baaaac`},
		{`a{2,3}`, `baaac`},
		{`a{2,3}`, `baac`},
		{`a{2,3}`, `bac`},
		{`a{2,3}`, `bc`},
		{`a{2,3}`, `c`},
		{`a{2,}`, ``},
		{`a{2,}`, `a`},
		{`a{2,}`, `aa`},
		{`a{2,}`, `aaa`},
		{`a{2,}`, `aaac`},
		{`a{2,}`, `aac`},
		{`a{2,}`, `ac`},
		{`a{2,}`, `b`},
		{`a{2,}`, `ba`},
		{`a{2,}`, `baa`},
		{`a{2,}`, `baaa`},
		{`a{2,}`, `baaac`},
		{`a{2,}`, `baac`},
		{`a{2,}`, `bac`},
		{`a{2,}`, `bc`},
		{`a{2,}`, `c`},
		{`a{2}`, ``},
		{`a{2}`, `a`},
		{`a{2}`, `aa`},
		{`a{2}`, `aaa`},
		{`a{2}`, `aaac`},
		{`a{2}`, `aac`},
		{`a{2}`, `ac`},
		{`a{2}`, `b`},
		{`a{2}`, `ba`},
		{`a{2}`, `baa`},
		{`a{2}`, `baaa`},
		{`a{2}`, `baaac`},
		{`a{2}`, `baac`},
		{`a{2}`, `bac`},
		{`a{2}`, `bc`},
		{`a{2}`, `c`},
		{`a|\^b`, ``},
		{`a|\^b`, `a`},
		{`a|\^b`, `b`},
		{`a|\^b`, `c`},
		{`a|^b`, ``},
		{`a|^b`, `a`},
		{`a|^b`, `b`},
		{`a|^b`, `c`},
		{`bar.*`, `seafood`},
		{`bc`, `ac`},
		{`bc`, `bc`},
		{`bc`, `cc`},
		{`foo.*`, `seafood`},
	}
)

func TestGoodCompile2(t *testing.T) {
	for i := 0; i < len(goodRe2); i++ {
		compileTest(t, goodRe2[i], "")
	}
}

func TestBadCompile2(t *testing.T) {
	for _, re := range badRe2 {
		_, err := regexp.Compile(re)
		if err == nil {
			panic("internal error")
		}

		compileTest(t, re, err.Error())
	}
}

func BenchmarkCompileGood(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, v := range goodRe2 {
			if _, err := regexp.Compile(v); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkCompileGoodNew(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, v := range goodRe2 {
			_, err := Compile(v)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func (re *Regexp) str(start int) string {
	if start != re.start && start != re.start1 {
		panic("internal error")
	}

	sa := make([]string, len(re.prog))
	m := map[int]struct{}{}
	var f func(int)
	f = func(s int) {
		if _, ok := m[s]; ok {
			return
		}

		m[s] = struct{}{}
		p := "  "
		if s == start {
			p = "->"
		}
		neg := ""
		switch state := &re.prog[s]; state.kind {
		case opAccept:
			if re.accept != s {
				panic("internal error")
			}
			sa[s] = fmt.Sprintf("%s %d\taccept\t", p, s)
		case opAssert:
			ss, ok := assertString[state.arg]
			if !ok {
				panic(state.arg)
			}
			sa[s] = fmt.Sprintf("%s %d\t%s\t%d", p, s, ss, state.out)
			f(state.out)
		case opAssertEOT:
			sa[s] = fmt.Sprintf("%s %d\t$\t%d", p, s, state.out)
			f(state.out)
		case opDot:
			sa[s] = fmt.Sprintf("%s %d\t.\t%d", p, s, state.out)
			f(state.out)
		case opDotNL:
			sa[s] = fmt.Sprintf("%s %d\t.NL\t%d", p, s, state.out)
			f(state.out)
		case opNop:
			sa[s] = fmt.Sprintf("%s %d\tempty\t%d", p, s, state.out)
			f(state.out)
		case opChar:
			sa[s] = fmt.Sprintf("%s %d\tchar\t%q, %d", p, s, state.arg, state.out)
			f(state.out)
		case opSave:
			sa[s] = fmt.Sprintf("%s %d\tsave\t%d, %d", p, s, state.arg, state.out)
			f(state.out)
		case opSplit:
			sa[s] = fmt.Sprintf("%s %d\tsplit\t%d, %d", p, s, state.out, state.out1)
			f(state.out)
			f(state.out1)
		case opNotCharClass:
			neg = "^"
			fallthrough
		case opCharClass:
			var a []string
			for i := state.arg; i < state.arg2; i += 2 {
				l := re.regs[i]
				if l < 0 {
					a = append(a, fmt.Sprintf("\\%s", assertString[-l]))
					continue
				}

				h := re.regs[i+1]
				if l == h {
					switch l {
					case '^', '-', '\\':
						a = append(a, fmt.Sprintf("\\%s", string(l)))
					case '\n':
						a = append(a, "\\n")
					default:
						a = append(a, regexp.QuoteMeta(string(l)))
					}
					continue
				}

				a = append(a, fmt.Sprintf("%s-%s", regexp.QuoteMeta(string(l)), regexp.QuoteMeta(string(h))))
			}
			sa[s] = fmt.Sprintf("%s %d\t%s[%s]\t%d", p, s, neg, strings.Join(a, ""), state.out)
			f(state.out)
		default:
			panic(state.kind)
		}
	}
	f(start)
	a := make([]int, 0, len(m))
	for pc := range m {
		a = append(a, pc)
	}
	sort.Ints(a)
	sa0 := sa
	sa = sa[:0]
	for _, v := range a {
		sa = append(sa, sa0[v])
	}
	return strings.Join(sa, "\n")
}

func (re *Regexp) checkInvariants(s int) {
	for _, state := range re.reachable(s, -1) {
		ps := &re.prog[state]
		switch ps.kind {
		case
			opAccept,
			opAssert,
			opAssertEOT,
			opChar,
			opCharClass,
			opDot,
			opDotNL,
			opNotCharClass,
			opSave,
			opSplit:
			// nop
		case opNop:
			if noOpt {
				break
			}

			panic("internal error")
		default:
			panic(ps.kind)
		}
	}
}

func (re *Regexp) fullMatch(s string) bool {
	return newVM(re, strings.NewReader(s)).fullMatch()
}

func TestFullMatch(t *testing.T) {
	o := *oCase
	for i, v := range simpleTests {
		if o >= 0 && i != o {
			continue
		}

		s := v.re
		if !strings.HasPrefix(s, "^") {
			s = "^" + s
		}
		if !strings.HasSuffix(s, "$") {
			s = s + "$"
		}
		re2, err := regexp.Compile(s)
		if err != nil {
			t.Errorf("%d: `%s`: %s", i, s, err)
		}

		re3, err := regexp.CompilePOSIX(s)
		if err != nil {
			t.Errorf("%d: `%s`: %s", i, s, err)
		}

		re, err := Compile(s)
		if err != nil {
			t.Error(i, err)
			return
		}

		re.checkInvariants(re.start)
		if o >= 0 {
			t.Logf("[%d]\n`%s` `%s`\n%s", i, s, v.src, re.str(re.start))
		}

		var g, e bool
		if g, e = re.fullMatch(v.src), re2.MatchString(v.src); g != e {
			t.Errorf("%d: `%s` `%s` got %v exp %v", i, s, v.src, g, e)
		}

		if e = re3.MatchString(v.src); g != e {
			t.Errorf("%d: `%s` `%s` got %v exp %v (POSIX)", i, s, v.src, g, e)
		}
	}
}

func TestMatchString(t *testing.T) {
	o := *oCase
	for i, v := range simpleTests {
		if o >= 0 && i != o {
			continue
		}

		s := v.re
		re2, err := regexp.Compile(s)
		if err != nil {
			t.Errorf("%d: `%s`: %s", i, s, err)
		}

		re3, err := regexp.CompilePOSIX(s)
		if err != nil {
			t.Errorf("%d: `%s`: %s", i, s, err)
		}

		re, err := Compile(s)
		if err != nil {
			t.Error(i, err)
			return
		}

		re.checkInvariants(re.start1)
		if o >= 0 {
			t.Logf("[%d]\n`%s` `%s`\n%s", i, s, v.src, re.str(re.start1))
		}

		var g, e bool
		if g, e = re.MatchString(v.src), re2.MatchString(v.src); g != e {
			t.Errorf("%d: `%s` `%s` got %v exp %v", i, s, v.src, g, e)
		}

		if e = re3.MatchString(v.src); g != e {
			t.Errorf("%d: `%s` `%s` got %v exp %v (POSIX)", i, s, v.src, g, e)
		}
	}
}

func BenchmarkCompileSimple(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, v := range simpleTests {
			_, err := regexp.Compile(v.re)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkCompileSimplePOSIX(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, v := range simpleTests {
			_, err := regexp.CompilePOSIX(v.re)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkCompileSimpleNew(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, v := range simpleTests {
			_, err := Compile(v.re)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkMatchFullSimple(b *testing.B) {
	a := make([]*regexp.Regexp, len(simpleTests))
	for i, v := range simpleTests {
		s := v.re
		if !strings.HasPrefix(s, "^") {
			s = "^" + s
		}
		if !strings.HasSuffix(s, "$") {
			s = s + "$"
		}
		var err error
		if a[i], err = regexp.Compile(s); err != nil {
			b.Fatal(err)
			return
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for i, v := range simpleTests {
			a[i].MatchString(v.src)
		}
	}
}

func BenchmarkMatchFullSimplePOSIX(b *testing.B) {
	a := make([]*regexp.Regexp, len(simpleTests))
	for i, v := range simpleTests {
		s := v.re
		if !strings.HasPrefix(s, "^") {
			s = "^" + s
		}
		if !strings.HasSuffix(s, "$") {
			s = s + "$"
		}
		var err error
		if a[i], err = regexp.CompilePOSIX(s); err != nil {
			b.Fatal(err)
			return
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for i, v := range simpleTests {
			a[i].MatchString(v.src)
		}
	}
}

func BenchmarkMatchFullSimpleNew(b *testing.B) {
	a := make([]*Regexp, len(simpleTests))
	for i, v := range simpleTests {
		var err error
		if a[i], err = Compile(v.re); err != nil {
			b.Fatal(err)
			return
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for i, v := range simpleTests {
			a[i].fullMatch(v.src)
		}
	}
}

func BenchmarkMatchStringSimple(b *testing.B) {
	a := make([]*regexp.Regexp, len(simpleTests))
	for i, v := range simpleTests {
		var err error
		if a[i], err = regexp.Compile(v.re); err != nil {
			b.Fatal(err)
			return
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for i, v := range simpleTests {
			a[i].MatchString(v.src)
		}
	}
}

func BenchmarkMatchStringSimplePOSIX(b *testing.B) {
	a := make([]*regexp.Regexp, len(simpleTests))
	for i, v := range simpleTests {
		var err error
		if a[i], err = regexp.CompilePOSIX(v.re); err != nil {
			b.Fatal(err)
			return
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for i, v := range simpleTests {
			a[i].MatchString(v.src)
		}
	}
}

func BenchmarkMatchStringSimpleNew(b *testing.B) {
	a := make([]*Regexp, len(simpleTests))
	for i, v := range simpleTests {
		var err error
		if a[i], err = Compile(v.re); err != nil {
			b.Fatal(err)
			return
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for i, v := range simpleTests {
			a[i].MatchString(v.src)
		}
	}
}

type regexper interface {
	MatchString(string) bool
}

const benchmarkCountRe = `.*a.{%d}b`

var benchmarkCountStr = strings.Repeat("a", 1e5)

func benchmarkCount0(b *testing.B, re regexper) {
	b.SetBytes(1e5)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if re.MatchString(benchmarkCountStr) {
			b.Fatal()
		}
	}
}

func benchmarkCount(b *testing.B, src string) {
	benchmarkCount0(b, regexp.MustCompile(src))
}

func benchmarkCountPOSIX(b *testing.B, src string) {
	benchmarkCount0(b, regexp.MustCompilePOSIX(src))
}

func benchmarkCountNew(b *testing.B, src string) {
	benchmarkCount0(b, MustCompile(src))
}

func BenchmarkCount_2_1e5(b *testing.B) {
	benchmarkCount(b, fmt.Sprintf(benchmarkCountRe, 2))
}

func BenchmarkCountPOSIX_2_1e5(b *testing.B) {
	benchmarkCountPOSIX(b, fmt.Sprintf(benchmarkCountRe, 2))
}

func BenchmarkCountNew_2_1e5(b *testing.B) {
	benchmarkCountNew(b, fmt.Sprintf(benchmarkCountRe, 2))
}

func BenchmarkCoun_256_1e5(b *testing.B) {
	benchmarkCount(b, fmt.Sprintf(benchmarkCountRe, 256))
}

func BenchmarkCountPOSIX_256_1e5(b *testing.B) {
	benchmarkCountPOSIX(b, fmt.Sprintf(benchmarkCountRe, 256))
}

func BenchmarkCountNew_256_1e5(b *testing.B) {
	benchmarkCountNew(b, fmt.Sprintf(benchmarkCountRe, 256))
}
