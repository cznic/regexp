// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the GO-LICENSE file.

package regexp

import (
	"reflect"
	//TODO "regexp/syntax"
	"strings"
	"testing"
)

var goodRe = []string{
	``,
	`.`,
	`^.$`,
	`a`,
	`a*`,
	`a+`,
	`a?`,
	`a|b`,
	`a*|b*`,
	`(a*|b)(c*|d)`,
	`[a-z]`,
	`[a-abc-c\-\]\[]`,
	`[a-z]+`,
	`[abc]`,
	`[^1234]`,
	`[^\n]`,
	`\!\\`,
}

type stringError struct {
	re  string
	err string
}

var badRe = []stringError{
	{`*`, "missing argument to repetition operator: `*`"},
	{`+`, "missing argument to repetition operator: `+`"},
	{`?`, "missing argument to repetition operator: `?`"},
	{`(abc`, "missing closing ): `(abc`"},
	{`abc)`, "unexpected ): `abc)`"},
	{`x[a-z`, "missing closing ]: `[a-z`"},
	{`[z-a]`, "invalid character class range: `z-a`"},
	{`abc\`, "trailing backslash at end of expression"},
	{`a**`, "invalid nested repetition operator: `**`"},
	{`a*+`, "invalid nested repetition operator: `*+`"},
	{`\x`, "invalid escape sequence: `\\x`"},
}

func compileTest(t *testing.T, expr string, error string) *Regexp {
	re, err := Compile(expr)
	if error == "" && err != nil {
		t.Error("compiling `", expr, "`; unexpected error: ", err.Error())
	}
	if error != "" && err == nil {
		t.Error("compiling `", expr, "`; missing error")
	} else if error != "" && !strings.Contains(err.Error(), error) {
		t.Error("compiling `", expr, "`; wrong error: ", err.Error(), "; want ", error)
	}
	return re
}

func TestGoodCompile(t *testing.T) {
	for i := 0; i < len(goodRe); i++ {
		compileTest(t, goodRe[i], "")
	}
}

func TestBadCompile(t *testing.T) {
	for i := 0; i < len(badRe); i++ {
		compileTest(t, badRe[i].re, badRe[i].err)
	}
}

func matchTest(t *testing.T, test *FindTest) {
	re := compileTest(t, test.pat, "")
	if re == nil {
		return
	}
	m := re.MatchString(test.text)
	if m != (len(test.matches) > 0) {
		t.Errorf("MatchString failure on %s: %t should be %t", test, m, len(test.matches) > 0)
	}
	// now try bytes
	m = re.Match([]byte(test.text))
	if m != (len(test.matches) > 0) {
		t.Errorf("Match failure on %s: %t should be %t", test, m, len(test.matches) > 0)
	}
}

func TestMatch(t *testing.T) {
	for _, test := range findTests {
		matchTest(t, &test)
	}
}

func matchFunctionTest(t *testing.T, test *FindTest) {
	m, err := MatchString(test.pat, test.text)
	if err == nil {
		return
	}
	if m != (len(test.matches) > 0) {
		t.Errorf("Match failure on %s: %t should be %t", test, m, len(test.matches) > 0)
	}
}

func TestMatchFunction(t *testing.T) {
	for _, test := range findTests {
		matchFunctionTest(t, &test)
	}
}

func copyMatchTest(t *testing.T, test *FindTest) {
	re := compileTest(t, test.pat, "")
	if re == nil {
		return
	}
	m1 := re.MatchString(test.text)
	m2 := re.Copy().MatchString(test.text)
	if m1 != m2 {
		t.Errorf("Copied Regexp match failure on %s: original gave %t; copy gave %t; should be %t",
			test, m1, m2, len(test.matches) > 0)
	}
}

func TestCopyMatch(t *testing.T) {
	for _, test := range findTests {
		copyMatchTest(t, &test)
	}
}

//TODO type ReplaceTest struct {
//TODO 	pattern, replacement, input, output string
//TODO }
//TODO
//TODO var replaceTests = []ReplaceTest{
//TODO 	// Test empty input and/or replacement, with pattern that matches the empty string.
//TODO 	{"", "", "", ""},
//TODO 	{"", "x", "", "x"},
//TODO 	{"", "", "abc", "abc"},
//TODO 	{"", "x", "abc", "xaxbxcx"},
//TODO
//TODO 	// Test empty input and/or replacement, with pattern that does not match the empty string.
//TODO 	{"b", "", "", ""},
//TODO 	{"b", "x", "", ""},
//TODO 	{"b", "", "abc", "ac"},
//TODO 	{"b", "x", "abc", "axc"},
//TODO 	{"y", "", "", ""},
//TODO 	{"y", "x", "", ""},
//TODO 	{"y", "", "abc", "abc"},
//TODO 	{"y", "x", "abc", "abc"},
//TODO
//TODO 	// Multibyte characters -- verify that we don't try to match in the middle
//TODO 	// of a character.
//TODO 	{"[a-c]*", "x", "\u65e5", "x\u65e5x"},
//TODO 	{"[^\u65e5]", "x", "abc\u65e5def", "xxx\u65e5xxx"},
//TODO
//TODO 	// Start and end of a string.
//TODO 	{"^[a-c]*", "x", "abcdabc", "xdabc"},
//TODO 	{"[a-c]*$", "x", "abcdabc", "abcdx"},
//TODO 	{"^[a-c]*$", "x", "abcdabc", "abcdabc"},
//TODO 	{"^[a-c]*", "x", "abc", "x"},
//TODO 	{"[a-c]*$", "x", "abc", "x"},
//TODO 	{"^[a-c]*$", "x", "abc", "x"},
//TODO 	{"^[a-c]*", "x", "dabce", "xdabce"},
//TODO 	{"[a-c]*$", "x", "dabce", "dabcex"},
//TODO 	{"^[a-c]*$", "x", "dabce", "dabce"},
//TODO 	{"^[a-c]*", "x", "", "x"},
//TODO 	{"[a-c]*$", "x", "", "x"},
//TODO 	{"^[a-c]*$", "x", "", "x"},
//TODO
//TODO 	{"^[a-c]+", "x", "abcdabc", "xdabc"},
//TODO 	{"[a-c]+$", "x", "abcdabc", "abcdx"},
//TODO 	{"^[a-c]+$", "x", "abcdabc", "abcdabc"},
//TODO 	{"^[a-c]+", "x", "abc", "x"},
//TODO 	{"[a-c]+$", "x", "abc", "x"},
//TODO 	{"^[a-c]+$", "x", "abc", "x"},
//TODO 	{"^[a-c]+", "x", "dabce", "dabce"},
//TODO 	{"[a-c]+$", "x", "dabce", "dabce"},
//TODO 	{"^[a-c]+$", "x", "dabce", "dabce"},
//TODO 	{"^[a-c]+", "x", "", ""},
//TODO 	{"[a-c]+$", "x", "", ""},
//TODO 	{"^[a-c]+$", "x", "", ""},
//TODO
//TODO 	// Other cases.
//TODO 	{"abc", "def", "abcdefg", "defdefg"},
//TODO 	{"bc", "BC", "abcbcdcdedef", "aBCBCdcdedef"},
//TODO 	{"abc", "", "abcdabc", "d"},
//TODO 	{"x", "xXx", "xxxXxxx", "xXxxXxxXxXxXxxXxxXx"},
//TODO 	{"abc", "d", "", ""},
//TODO 	{"abc", "d", "abc", "d"},
//TODO 	{".+", "x", "abc", "x"},
//TODO 	{"[a-c]*", "x", "def", "xdxexfx"},
//TODO 	{"[a-c]+", "x", "abcbcdcdedef", "xdxdedef"},
//TODO 	{"[a-c]*", "x", "abcbcdcdedef", "xdxdxexdxexfx"},
//TODO
//TODO 	// Substitutions
//TODO 	{"a+", "($0)", "banana", "b(a)n(a)n(a)"},
//TODO 	{"a+", "(${0})", "banana", "b(a)n(a)n(a)"},
//TODO 	{"a+", "(${0})$0", "banana", "b(a)an(a)an(a)a"},
//TODO 	{"a+", "(${0})$0", "banana", "b(a)an(a)an(a)a"},
//TODO 	{"hello, (.+)", "goodbye, ${1}", "hello, world", "goodbye, world"},
//TODO 	{"hello, (.+)", "goodbye, $1x", "hello, world", "goodbye, "},
//TODO 	{"hello, (.+)", "goodbye, ${1}x", "hello, world", "goodbye, worldx"},
//TODO 	{"hello, (.+)", "<$0><$1><$2><$3>", "hello, world", "<hello, world><world><><>"},
//TODO 	{"hello, (?P<noun>.+)", "goodbye, $noun!", "hello, world", "goodbye, world!"},
//TODO 	{"hello, (?P<noun>.+)", "goodbye, ${noun}", "hello, world", "goodbye, world"},
//TODO 	{"(?P<x>hi)|(?P<x>bye)", "$x$x$x", "hi", "hihihi"},
//TODO 	{"(?P<x>hi)|(?P<x>bye)", "$x$x$x", "bye", "byebyebye"},
//TODO 	{"(?P<x>hi)|(?P<x>bye)", "$xyz", "hi", ""},
//TODO 	{"(?P<x>hi)|(?P<x>bye)", "${x}yz", "hi", "hiyz"},
//TODO 	{"(?P<x>hi)|(?P<x>bye)", "hello $$x", "hi", "hello $x"},
//TODO 	{"a+", "${oops", "aaa", "${oops"},
//TODO 	{"a+", "$$", "aaa", "$"},
//TODO 	{"a+", "$", "aaa", "$"},
//TODO
//TODO 	// Substitution when subexpression isn't found
//TODO 	{"(x)?", "$1", "123", "123"},
//TODO 	{"abc", "$1", "123", "123"},
//TODO
//TODO 	// Substitutions involving a (x){0}
//TODO 	{"(a)(b){0}(c)", ".$1|$3.", "xacxacx", "x.a|c.x.a|c.x"},
//TODO 	{"(a)(((b))){0}c", ".$1.", "xacxacx", "x.a.x.a.x"},
//TODO 	{"((a(b){0}){3}){5}(h)", "y caramb$2", "say aaaaaaaaaaaaaaaah", "say ay caramba"},
//TODO 	{"((a(b){0}){3}){5}h", "y caramb$2", "say aaaaaaaaaaaaaaaah", "say ay caramba"},
//TODO }
//TODO
//TODO var replaceLiteralTests = []ReplaceTest{
//TODO 	// Substitutions
//TODO 	{"a+", "($0)", "banana", "b($0)n($0)n($0)"},
//TODO 	{"a+", "(${0})", "banana", "b(${0})n(${0})n(${0})"},
//TODO 	{"a+", "(${0})$0", "banana", "b(${0})$0n(${0})$0n(${0})$0"},
//TODO 	{"a+", "(${0})$0", "banana", "b(${0})$0n(${0})$0n(${0})$0"},
//TODO 	{"hello, (.+)", "goodbye, ${1}", "hello, world", "goodbye, ${1}"},
//TODO 	{"hello, (?P<noun>.+)", "goodbye, $noun!", "hello, world", "goodbye, $noun!"},
//TODO 	{"hello, (?P<noun>.+)", "goodbye, ${noun}", "hello, world", "goodbye, ${noun}"},
//TODO 	{"(?P<x>hi)|(?P<x>bye)", "$x$x$x", "hi", "$x$x$x"},
//TODO 	{"(?P<x>hi)|(?P<x>bye)", "$x$x$x", "bye", "$x$x$x"},
//TODO 	{"(?P<x>hi)|(?P<x>bye)", "$xyz", "hi", "$xyz"},
//TODO 	{"(?P<x>hi)|(?P<x>bye)", "${x}yz", "hi", "${x}yz"},
//TODO 	{"(?P<x>hi)|(?P<x>bye)", "hello $$x", "hi", "hello $$x"},
//TODO 	{"a+", "${oops", "aaa", "${oops"},
//TODO 	{"a+", "$$", "aaa", "$$"},
//TODO 	{"a+", "$", "aaa", "$"},
//TODO }
//TODO
//TODO type ReplaceFuncTest struct {
//TODO 	pattern       string
//TODO 	replacement   func(string) string
//TODO 	input, output string
//TODO }
//TODO
//TODO var replaceFuncTests = []ReplaceFuncTest{
//TODO 	{"[a-c]", func(s string) string { return "x" + s + "y" }, "defabcdef", "defxayxbyxcydef"},
//TODO 	{"[a-c]+", func(s string) string { return "x" + s + "y" }, "defabcdef", "defxabcydef"},
//TODO 	{"[a-c]*", func(s string) string { return "x" + s + "y" }, "defabcdef", "xydxyexyfxabcydxyexyfxy"},
//TODO }
//TODO
//TODO func TestReplaceAll(t *testing.T) {
//TODO 	for _, tc := range replaceTests {
//TODO 		re, err := Compile(tc.pattern)
//TODO 		if err != nil {
//TODO 			t.Errorf("Unexpected error compiling %q: %v", tc.pattern, err)
//TODO 			continue
//TODO 		}
//TODO 		actual := re.ReplaceAllString(tc.input, tc.replacement)
//TODO 		if actual != tc.output {
//TODO 			t.Errorf("%q.ReplaceAllString(%q,%q) = %q; want %q",
//TODO 				tc.pattern, tc.input, tc.replacement, actual, tc.output)
//TODO 		}
//TODO 		// now try bytes
//TODO 		actual = string(re.ReplaceAll([]byte(tc.input), []byte(tc.replacement)))
//TODO 		if actual != tc.output {
//TODO 			t.Errorf("%q.ReplaceAll(%q,%q) = %q; want %q",
//TODO 				tc.pattern, tc.input, tc.replacement, actual, tc.output)
//TODO 		}
//TODO 	}
//TODO }
//TODO
//TODO func TestReplaceAllLiteral(t *testing.T) {
//TODO 	// Run ReplaceAll tests that do not have $ expansions.
//TODO 	for _, tc := range replaceTests {
//TODO 		if strings.Contains(tc.replacement, "$") {
//TODO 			continue
//TODO 		}
//TODO 		re, err := Compile(tc.pattern)
//TODO 		if err != nil {
//TODO 			t.Errorf("Unexpected error compiling %q: %v", tc.pattern, err)
//TODO 			continue
//TODO 		}
//TODO 		actual := re.ReplaceAllLiteralString(tc.input, tc.replacement)
//TODO 		if actual != tc.output {
//TODO 			t.Errorf("%q.ReplaceAllLiteralString(%q,%q) = %q; want %q",
//TODO 				tc.pattern, tc.input, tc.replacement, actual, tc.output)
//TODO 		}
//TODO 		// now try bytes
//TODO 		actual = string(re.ReplaceAllLiteral([]byte(tc.input), []byte(tc.replacement)))
//TODO 		if actual != tc.output {
//TODO 			t.Errorf("%q.ReplaceAllLiteral(%q,%q) = %q; want %q",
//TODO 				tc.pattern, tc.input, tc.replacement, actual, tc.output)
//TODO 		}
//TODO 	}
//TODO
//TODO 	// Run literal-specific tests.
//TODO 	for _, tc := range replaceLiteralTests {
//TODO 		re, err := Compile(tc.pattern)
//TODO 		if err != nil {
//TODO 			t.Errorf("Unexpected error compiling %q: %v", tc.pattern, err)
//TODO 			continue
//TODO 		}
//TODO 		actual := re.ReplaceAllLiteralString(tc.input, tc.replacement)
//TODO 		if actual != tc.output {
//TODO 			t.Errorf("%q.ReplaceAllLiteralString(%q,%q) = %q; want %q",
//TODO 				tc.pattern, tc.input, tc.replacement, actual, tc.output)
//TODO 		}
//TODO 		// now try bytes
//TODO 		actual = string(re.ReplaceAllLiteral([]byte(tc.input), []byte(tc.replacement)))
//TODO 		if actual != tc.output {
//TODO 			t.Errorf("%q.ReplaceAllLiteral(%q,%q) = %q; want %q",
//TODO 				tc.pattern, tc.input, tc.replacement, actual, tc.output)
//TODO 		}
//TODO 	}
//TODO }
//TODO
//TODO func TestReplaceAllFunc(t *testing.T) {
//TODO 	for _, tc := range replaceFuncTests {
//TODO 		re, err := Compile(tc.pattern)
//TODO 		if err != nil {
//TODO 			t.Errorf("Unexpected error compiling %q: %v", tc.pattern, err)
//TODO 			continue
//TODO 		}
//TODO 		actual := re.ReplaceAllStringFunc(tc.input, tc.replacement)
//TODO 		if actual != tc.output {
//TODO 			t.Errorf("%q.ReplaceFunc(%q,fn) = %q; want %q",
//TODO 				tc.pattern, tc.input, actual, tc.output)
//TODO 		}
//TODO 		// now try bytes
//TODO 		actual = string(re.ReplaceAllFunc([]byte(tc.input), func(s []byte) []byte { return []byte(tc.replacement(string(s))) }))
//TODO 		if actual != tc.output {
//TODO 			t.Errorf("%q.ReplaceFunc(%q,fn) = %q; want %q",
//TODO 				tc.pattern, tc.input, actual, tc.output)
//TODO 		}
//TODO 	}
//TODO }

type MetaTest struct {
	pattern, output, literal string
	isLiteral                bool
}

var metaTests = []MetaTest{
	{``, ``, ``, true},
	{`foo`, `foo`, `foo`, true},
	{`foo\.\$`, `foo\\\.\\\$`, `foo.$`, true}, // has meta but no operator
	{`foo.\$`, `foo\.\\\$`, `foo`, false},     // has escaped operators and real operators
	{`!@#$%^&*()_+-=[{]}\|,<.>/?~`, `!@#\$%\^&\*\(\)_\+-=\[\{\]\}\\\|,<\.>/\?~`, `!@#`, false},
}

//TODO var literalPrefixTests = []MetaTest{
//TODO 	// See golang.org/issue/11175.
//TODO 	// output is unused.
//TODO 	{`^0^0$`, ``, `0`, false},
//TODO 	{`^0^`, ``, ``, false},
//TODO 	{`^0$`, ``, `0`, true},
//TODO 	{`$0^`, ``, ``, false},
//TODO 	{`$0$`, ``, ``, false},
//TODO 	{`^^0$$`, ``, ``, false},
//TODO 	{`^$^$`, ``, ``, false},
//TODO 	{`$$0^^`, ``, ``, false},
//TODO }

func TestQuoteMeta(t *testing.T) {
	for _, tc := range metaTests {
		// Verify that QuoteMeta returns the expected string.
		quoted := QuoteMeta(tc.pattern)
		if quoted != tc.output {
			t.Errorf("QuoteMeta(`%s`) = `%s`; want `%s`",
				tc.pattern, quoted, tc.output)
			continue
		}

		//TODO 		// Verify that the quoted string is in fact treated as expected
		//TODO 		// by Compile -- i.e. that it matches the original, unquoted string.
		//TODO 		if tc.pattern != "" {
		//TODO 			re, err := Compile(quoted)
		//TODO 			if err != nil {
		//TODO 				t.Errorf("Unexpected error compiling QuoteMeta(`%s`): %v", tc.pattern, err)
		//TODO 				continue
		//TODO 			}
		//TODO 			src := "abc" + tc.pattern + "def"
		//TODO 			repl := "xyz"
		//TODO 			replaced := re.ReplaceAllString(src, repl)
		//TODO 			expected := "abcxyzdef"
		//TODO 			if replaced != expected {
		//TODO 				t.Errorf("QuoteMeta(`%s`).Replace(`%s`,`%s`) = `%s`; want `%s`",
		//TODO 					tc.pattern, src, repl, replaced, expected)
		//TODO 			}
		//TODO 		}
	}
}

//TODO func TestLiteralPrefix(t *testing.T) {
//TODO 	for _, tc := range append(metaTests, literalPrefixTests...) {
//TODO 		// Literal method needs to scan the pattern.
//TODO 		re := MustCompile(tc.pattern)
//TODO 		str, complete := re.LiteralPrefix()
//TODO 		if complete != tc.isLiteral {
//TODO 			t.Errorf("LiteralPrefix(`%s`) = %t; want %t", tc.pattern, complete, tc.isLiteral)
//TODO 		}
//TODO 		if str != tc.literal {
//TODO 			t.Errorf("LiteralPrefix(`%s`) = `%s`; want `%s`", tc.pattern, str, tc.literal)
//TODO 		}
//TODO 	}
//TODO }

type subexpCase struct {
	input string
	num   int
	names []string
}

var subexpCases = []subexpCase{
	{``, 0, nil},
	{`.*`, 0, nil},
	{`abba`, 0, nil},
	{`ab(b)a`, 1, []string{"", ""}},
	{`ab(.*)a`, 1, []string{"", ""}},
	{`(.*)ab(.*)a`, 2, []string{"", "", ""}},
	{`(.*)(ab)(.*)a`, 3, []string{"", "", "", ""}},
	{`(.*)((a)b)(.*)a`, 4, []string{"", "", "", "", ""}},
	{`(.*)(\(ab)(.*)a`, 3, []string{"", "", "", ""}},
	{`(.*)(\(a\)b)(.*)a`, 3, []string{"", "", "", ""}},
	{`(?P<foo>.*)(?P<bar>(a)b)(?P<foo>.*)a`, 4, []string{"", "foo", "bar", "", "foo"}},
}

func TestSubexp(t *testing.T) {
	for _, c := range subexpCases {
		re := MustCompile(c.input)
		n := re.NumSubexp()
		if n != c.num {
			t.Errorf("%q: NumSubexp = %d, want %d", c.input, n, c.num)
			continue
		}
		names := re.SubexpNames()
		if len(names) != 1+n {
			t.Errorf("%q: len(SubexpNames) = %d, want %d", c.input, len(names), n)
			continue
		}
		if c.names != nil {
			for i := 0; i < 1+n; i++ {
				if names[i] != c.names[i] {
					t.Errorf("%q: SubexpNames[%d] = %q, want %q", c.input, i, names[i], c.names[i])
				}
			}
		}
	}
}

var splitTests = []struct {
	s   string
	r   string
	n   int
	out []string
}{
	{"foo:and:bar", ":", -1, []string{"foo", "and", "bar"}},
	{"foo:and:bar", ":", 1, []string{"foo:and:bar"}},
	{"foo:and:bar", ":", 2, []string{"foo", "and:bar"}},
	{"foo:and:bar", "foo", -1, []string{"", ":and:bar"}},
	{"foo:and:bar", "bar", -1, []string{"foo:and:", ""}},
	{"foo:and:bar", "baz", -1, []string{"foo:and:bar"}},
	{"baabaab", "a", -1, []string{"b", "", "b", "", "b"}},
	{"baabaab", "a*", -1, []string{"b", "b", "b"}},
	{"baabaab", "ba*", -1, []string{"", "", "", ""}},
	{"foobar", "f*b*", -1, []string{"", "o", "o", "a", "r"}},
	{"foobar", "f+.*b+", -1, []string{"", "ar"}},
	{"foobooboar", "o{2}", -1, []string{"f", "b", "boar"}},
	{"a,b,c,d,e,f", ",", 3, []string{"a", "b", "c,d,e,f"}},
	{"a,b,c,d,e,f", ",", 0, nil},
	{",", ",", -1, []string{"", ""}},
	{",,,", ",", -1, []string{"", "", "", ""}},
	{"", ",", -1, []string{""}},
	{"", ".*", -1, []string{""}},
	{"", ".+", -1, []string{""}},
	{"", "", -1, []string{}},
	{"foobar", "", -1, []string{"f", "o", "o", "b", "a", "r"}},
	{"abaabaccadaaae", "a*", 5, []string{"", "b", "b", "c", "cadaaae"}},
	{":x:y:z:", ":", -1, []string{"", "x", "y", "z", ""}},
}

func TestSplit(t *testing.T) {
	for i, test := range splitTests {
		re, err := Compile(test.r)
		if err != nil {
			t.Errorf("#%d: %q: compile error: %s", i, test.r, err.Error())
			continue
		}

		split := re.Split(test.s, test.n)
		if !reflect.DeepEqual(split, test.out) {
			t.Errorf("#%d: %q: got %q; want %q", i, test.r, split, test.out)
		}

		if QuoteMeta(test.r) == test.r {
			strsplit := strings.SplitN(test.s, test.r, test.n)
			if !reflect.DeepEqual(split, strsplit) {
				t.Errorf("#%d: Split(%q, %q, %d): regexp vs strings mismatch\nregexp=%q\nstrings=%q", i, test.s, test.r, test.n, split, strsplit)
			}
		}
	}
}

//TODO // The following sequence of Match calls used to panic. See issue #12980.
//TODO func TestParseAndCompile(t *testing.T) {
//TODO 	expr := "a$"
//TODO 	s := "a\nb"
//TODO
//TODO 	for i, tc := range []struct {
//TODO 		reFlags  syntax.Flags
//TODO 		expMatch bool
//TODO 	}{
//TODO 		{syntax.Perl | syntax.OneLine, false},
//TODO 		{syntax.Perl &^ syntax.OneLine, true},
//TODO 	} {
//TODO 		parsed, err := syntax.Parse(expr, tc.reFlags)
//TODO 		if err != nil {
//TODO 			t.Fatalf("%d: parse: %v", i, err)
//TODO 		}
//TODO 		re, err := Compile(parsed.String())
//TODO 		if err != nil {
//TODO 			t.Fatalf("%d: compile: %v", i, err)
//TODO 		}
//TODO 		if match := re.MatchString(s); match != tc.expMatch {
//TODO 			t.Errorf("%d: %q.MatchString(%q)=%t; expected=%t", i, re, s, match, tc.expMatch)
//TODO 		}
//TODO 	}
//TODO }

//TODO // Check that one-pass cutoff does trigger.
//TODO func TestOnePassCutoff(t *testing.T) {
//TODO 	re, err := syntax.Parse(`^x{1,1000}y{1,1000}$`, syntax.Perl)
//TODO 	if err != nil {
//TODO 		t.Fatalf("parse: %v", err)
//TODO 	}
//TODO 	p, err := syntax.Compile(re.Simplify())
//TODO 	if err != nil {
//TODO 		t.Fatalf("compile: %v", err)
//TODO 	}
//TODO 	if compileOnePass(p) != notOnePass {
//TODO 		t.Fatalf("makeOnePass succeeded; wanted notOnePass")
//TODO 	}
//TODO }

// Check that the same machine can be used with the standard matcher
// and then the backtracker when there are no captures.
func TestSwitchBacktrack(t *testing.T) {
	re := MustCompile(`a|b`)
	long := make([]byte, maxBacktrackVector+1)

	// The following sequence of Match calls used to panic. See issue #10319.
	re.Match(long)     // triggers standard matcher
	re.Match(long[:1]) // triggers backtracker
}

func BenchmarkFind(b *testing.B) {
	b.StopTimer()
	re := MustCompile("a+b+")
	wantSubs := "aaabb"
	s := []byte("acbb" + wantSubs + "dd")
	b.StartTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		subs := re.Find(s)
		if string(subs) != wantSubs {
			b.Fatalf("Find(%q) = %q; want %q", s, subs, wantSubs)
		}
	}
}

func BenchmarkFindString(b *testing.B) {
	b.StopTimer()
	re := MustCompile("a+b+")
	wantSubs := "aaabb"
	s := "acbb" + wantSubs + "dd"
	b.StartTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		subs := re.FindString(s)
		if subs != wantSubs {
			b.Fatalf("FindString(%q) = %q; want %q", s, subs, wantSubs)
		}
	}
}

func BenchmarkFindSubmatch(b *testing.B) {
	b.StopTimer()
	re := MustCompile("a(a+b+)b")
	wantSubs := "aaabb"
	s := []byte("acbb" + wantSubs + "dd")
	b.StartTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		subs := re.FindSubmatch(s)
		if string(subs[0]) != wantSubs {
			b.Fatalf("FindSubmatch(%q)[0] = %q; want %q", s, subs[0], wantSubs)
		}
		if string(subs[1]) != "aab" {
			b.Fatalf("FindSubmatch(%q)[1] = %q; want %q", s, subs[1], "aab")
		}
	}
}

func BenchmarkFindStringSubmatch(b *testing.B) {
	b.StopTimer()
	re := MustCompile("a(a+b+)b")
	wantSubs := "aaabb"
	s := "acbb" + wantSubs + "dd"
	b.StartTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		subs := re.FindStringSubmatch(s)
		if subs[0] != wantSubs {
			b.Fatalf("FindStringSubmatch(%q)[0] = %q; want %q", s, subs[0], wantSubs)
		}
		if subs[1] != "aab" {
			b.Fatalf("FindStringSubmatch(%q)[1] = %q; want %q", s, subs[1], "aab")
		}
	}
}

func BenchmarkLiteral(b *testing.B) {
	x := strings.Repeat("x", 50) + "y"
	b.StopTimer()
	re := MustCompile("y")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if !re.MatchString(x) {
			b.Fatalf("no match!")
		}
	}
}

func BenchmarkNotLiteral(b *testing.B) {
	x := strings.Repeat("x", 50) + "y"
	b.StopTimer()
	re := MustCompile(".y")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if !re.MatchString(x) {
			b.Fatalf("no match!")
		}
	}
}

func BenchmarkMatchClass(b *testing.B) {
	b.StopTimer()
	x := strings.Repeat("xxxx", 20) + "w"
	re := MustCompile("[abcdw]")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if !re.MatchString(x) {
			b.Fatalf("no match!")
		}
	}
}

func BenchmarkMatchClass_InRange(b *testing.B) {
	b.StopTimer()
	// 'b' is between 'a' and 'c', so the charclass
	// range checking is no help here.
	x := strings.Repeat("bbbb", 20) + "c"
	re := MustCompile("[ac]")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if !re.MatchString(x) {
			b.Fatalf("no match!")
		}
	}
}

//TODO func BenchmarkReplaceAll(b *testing.B) {
//TODO 	x := "abcdefghijklmnopqrstuvwxyz"
//TODO 	b.StopTimer()
//TODO 	re := MustCompile("[cjrw]")
//TODO 	b.StartTimer()
//TODO 	for i := 0; i < b.N; i++ {
//TODO 		re.ReplaceAllString(x, "")
//TODO 	}
//TODO }

func BenchmarkAnchoredLiteralShortNonMatch(b *testing.B) {
	b.StopTimer()
	x := []byte("abcdefghijklmnopqrstuvwxyz")
	re := MustCompile("^zbc(d|e)")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		re.Match(x)
	}
}

func BenchmarkAnchoredLiteralLongNonMatch(b *testing.B) {
	b.StopTimer()
	x := []byte("abcdefghijklmnopqrstuvwxyz")
	for i := 0; i < 15; i++ {
		x = append(x, x...)
	}
	re := MustCompile("^zbc(d|e)")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		re.Match(x)
	}
}

func BenchmarkAnchoredShortMatch(b *testing.B) {
	b.StopTimer()
	x := []byte("abcdefghijklmnopqrstuvwxyz")
	re := MustCompile("^.bc(d|e)")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		re.Match(x)
	}
}

func BenchmarkAnchoredLongMatch(b *testing.B) {
	b.StopTimer()
	x := []byte("abcdefghijklmnopqrstuvwxyz")
	for i := 0; i < 15; i++ {
		x = append(x, x...)
	}
	re := MustCompile("^.bc(d|e)")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		re.Match(x)
	}
}

func BenchmarkOnePassShortA(b *testing.B) {
	b.StopTimer()
	x := []byte("abcddddddeeeededd")
	re := MustCompile("^.bc(d|e)*$")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		re.Match(x)
	}
}

func BenchmarkNotOnePassShortA(b *testing.B) {
	b.StopTimer()
	x := []byte("abcddddddeeeededd")
	re := MustCompile(".bc(d|e)*$")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		re.Match(x)
	}
}

func BenchmarkOnePassShortB(b *testing.B) {
	b.StopTimer()
	x := []byte("abcddddddeeeededd")
	re := MustCompile("^.bc(?:d|e)*$")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		re.Match(x)
	}
}

func BenchmarkNotOnePassShortB(b *testing.B) {
	b.StopTimer()
	x := []byte("abcddddddeeeededd")
	re := MustCompile(".bc(?:d|e)*$")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		re.Match(x)
	}
}

func BenchmarkOnePassLongPrefix(b *testing.B) {
	b.StopTimer()
	x := []byte("abcdefghijklmnopqrstuvwxyz")
	re := MustCompile("^abcdefghijklmnopqrstuvwxyz.*$")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		re.Match(x)
	}
}

func BenchmarkOnePassLongNotPrefix(b *testing.B) {
	b.StopTimer()
	x := []byte("abcdefghijklmnopqrstuvwxyz")
	re := MustCompile("^.bcdefghijklmnopqrstuvwxyz.*$")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		re.Match(x)
	}
}

func BenchmarkMatchParallelShared(b *testing.B) {
	x := []byte("this is a long line that contains foo bar baz")
	re := MustCompile("foo (ba+r)? baz")
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			re.Match(x)
		}
	})
}

func BenchmarkMatchParallelCopied(b *testing.B) {
	x := []byte("this is a long line that contains foo bar baz")
	re := MustCompile("foo (ba+r)? baz")
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		re := re.Copy()
		for pb.Next() {
			re.Match(x)
		}
	})
}

var sink string

func BenchmarkQuoteMetaAll(b *testing.B) {
	s := string(specialBytes)
	b.SetBytes(int64(len(s)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sink = QuoteMeta(s)
	}
}

func BenchmarkQuoteMetaNone(b *testing.B) {
	s := "abcdefghijklmnopqrstuvwxyz"
	b.SetBytes(int64(len(s)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sink = QuoteMeta(s)
	}
}
