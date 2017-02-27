// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the GO-LICENSE file.

package regexp

//TODO import (
//TODO 	"bufio"
//TODO 	"compress/bzip2"
//TODO 	"fmt"
//TODO 	"internal/testenv"
//TODO 	"io"
//TODO 	"os"
//TODO 	"path/filepath"
//TODO 	"regexp/syntax"
//TODO 	"strconv"
//TODO 	"strings"
//TODO 	"testing"
//TODO 	"unicode/utf8"
//TODO )
//TODO
//TODO // TestRE2 tests this package's regexp API against test cases
//TODO // considered during RE2's exhaustive tests, which run all possible
//TODO // regexps over a given set of atoms and operators, up to a given
//TODO // complexity, over all possible strings over a given alphabet,
//TODO // up to a given size. Rather than try to link with RE2, we read a
//TODO // log file containing the test cases and the expected matches.
//TODO // The log file, re2-exhaustive.txt, is generated by running 'make log'
//TODO // in the open source RE2 distribution https://github.com/google/re2/.
//TODO //
//TODO // The test file format is a sequence of stanzas like:
//TODO //
//TODO //	strings
//TODO //	"abc"
//TODO //	"123x"
//TODO //	regexps
//TODO //	"[a-z]+"
//TODO //	0-3;0-3
//TODO //	-;-
//TODO //	"([0-9])([0-9])([0-9])"
//TODO //	-;-
//TODO //	-;0-3 0-1 1-2 2-3
//TODO //
//TODO // The stanza begins by defining a set of strings, quoted
//TODO // using Go double-quote syntax, one per line. Then the
//TODO // regexps section gives a sequence of regexps to run on
//TODO // the strings. In the block that follows a regexp, each line
//TODO // gives the semicolon-separated match results of running
//TODO // the regexp on the corresponding string.
//TODO // Each match result is either a single -, meaning no match, or a
//TODO // space-separated sequence of pairs giving the match and
//TODO // submatch indices. An unmatched subexpression formats
//TODO // its pair as a single - (not illustrated above).  For now
//TODO // each regexp run produces two match results, one for a
//TODO // ``full match'' that restricts the regexp to matching the entire
//TODO // string or nothing, and one for a ``partial match'' that gives
//TODO // the leftmost first match found in the string.
//TODO //
//TODO // Lines beginning with # are comments. Lines beginning with
//TODO // a capital letter are test names printed during RE2's test suite
//TODO // and are echoed into t but otherwise ignored.
//TODO //
//TODO // At time of writing, re2-exhaustive.txt is 59 MB but compresses to 385 kB,
//TODO // so we store re2-exhaustive.txt.bz2 in the repository and decompress it on the fly.
//TODO //
//TODO func TestRE2Search(t *testing.T) {
//TODO 	testRE2(t, "testdata/re2-search.txt")
//TODO }
//TODO
//TODO func testRE2(t *testing.T, file string) {
//TODO 	f, err := os.Open(file)
//TODO 	if err != nil {
//TODO 		t.Fatal(err)
//TODO 	}
//TODO 	defer f.Close()
//TODO 	var txt io.Reader
//TODO 	if strings.HasSuffix(file, ".bz2") {
//TODO 		z := bzip2.NewReader(f)
//TODO 		txt = z
//TODO 		file = file[:len(file)-len(".bz2")] // for error messages
//TODO 	} else {
//TODO 		txt = f
//TODO 	}
//TODO 	lineno := 0
//TODO 	scanner := bufio.NewScanner(txt)
//TODO 	var (
//TODO 		str       []string
//TODO 		input     []string
//TODO 		inStrings bool
//TODO 		re        *Regexp
//TODO 		refull    *Regexp
//TODO 		nfail     int
//TODO 		ncase     int
//TODO 	)
//TODO 	for lineno := 1; scanner.Scan(); lineno++ {
//TODO 		line := scanner.Text()
//TODO 		switch {
//TODO 		case line == "":
//TODO 			t.Fatalf("%s:%d: unexpected blank line", file, lineno)
//TODO 		case line[0] == '#':
//TODO 			continue
//TODO 		case 'A' <= line[0] && line[0] <= 'Z':
//TODO 			// Test name.
//TODO 			t.Logf("%s\n", line)
//TODO 			continue
//TODO 		case line == "strings":
//TODO 			str = str[:0]
//TODO 			inStrings = true
//TODO 		case line == "regexps":
//TODO 			inStrings = false
//TODO 		case line[0] == '"':
//TODO 			q, err := strconv.Unquote(line)
//TODO 			if err != nil {
//TODO 				// Fatal because we'll get out of sync.
//TODO 				t.Fatalf("%s:%d: unquote %s: %v", file, lineno, line, err)
//TODO 			}
//TODO 			if inStrings {
//TODO 				str = append(str, q)
//TODO 				continue
//TODO 			}
//TODO 			// Is a regexp.
//TODO 			if len(input) != 0 {
//TODO 				t.Fatalf("%s:%d: out of sync: have %d strings left before %#q", file, lineno, len(input), q)
//TODO 			}
//TODO 			re, err = tryCompile(q)
//TODO 			if err != nil {
//TODO 				if err.Error() == "error parsing regexp: invalid escape sequence: `\\C`" {
//TODO 					// We don't and likely never will support \C; keep going.
//TODO 					continue
//TODO 				}
//TODO 				t.Errorf("%s:%d: compile %#q: %v", file, lineno, q, err)
//TODO 				if nfail++; nfail >= 100 {
//TODO 					t.Fatalf("stopping after %d errors", nfail)
//TODO 				}
//TODO 				continue
//TODO 			}
//TODO 			full := `\A(?:` + q + `)\z`
//TODO 			refull, err = tryCompile(full)
//TODO 			if err != nil {
//TODO 				// Fatal because q worked, so this should always work.
//TODO 				t.Fatalf("%s:%d: compile full %#q: %v", file, lineno, full, err)
//TODO 			}
//TODO 			input = str
//TODO 		case line[0] == '-' || '0' <= line[0] && line[0] <= '9':
//TODO 			// A sequence of match results.
//TODO 			ncase++
//TODO 			if re == nil {
//TODO 				// Failed to compile: skip results.
//TODO 				continue
//TODO 			}
//TODO 			if len(input) == 0 {
//TODO 				t.Fatalf("%s:%d: out of sync: no input remaining", file, lineno)
//TODO 			}
//TODO 			var text string
//TODO 			text, input = input[0], input[1:]
//TODO 			if !isSingleBytes(text) && strings.Contains(re.String(), `\B`) {
//TODO 				// RE2's \B considers every byte position,
//TODO 				// so it sees 'not word boundary' in the
//TODO 				// middle of UTF-8 sequences. This package
//TODO 				// only considers the positions between runes,
//TODO 				// so it disagrees. Skip those cases.
//TODO 				continue
//TODO 			}
//TODO 			res := strings.Split(line, ";")
//TODO 			if len(res) != len(run) {
//TODO 				t.Fatalf("%s:%d: have %d test results, want %d", file, lineno, len(res), len(run))
//TODO 			}
//TODO 			for i := range res {
//TODO 				have, suffix := run[i](re, refull, text)
//TODO 				want := parseResult(t, file, lineno, res[i])
//TODO 				if !same(have, want) {
//TODO 					t.Errorf("%s:%d: %#q%s.FindSubmatchIndex(%#q) = %v, want %v", file, lineno, re, suffix, text, have, want)
//TODO 					if nfail++; nfail >= 100 {
//TODO 						t.Fatalf("stopping after %d errors", nfail)
//TODO 					}
//TODO 					continue
//TODO 				}
//TODO 				b, suffix := match[i](re, refull, text)
//TODO 				if b != (want != nil) {
//TODO 					t.Errorf("%s:%d: %#q%s.MatchString(%#q) = %v, want %v", file, lineno, re, suffix, text, b, !b)
//TODO 					if nfail++; nfail >= 100 {
//TODO 						t.Fatalf("stopping after %d errors", nfail)
//TODO 					}
//TODO 					continue
//TODO 				}
//TODO 			}
//TODO
//TODO 		default:
//TODO 			t.Fatalf("%s:%d: out of sync: %s\n", file, lineno, line)
//TODO 		}
//TODO 	}
//TODO 	if err := scanner.Err(); err != nil {
//TODO 		t.Fatalf("%s:%d: %v", file, lineno, err)
//TODO 	}
//TODO 	if len(input) != 0 {
//TODO 		t.Fatalf("%s:%d: out of sync: have %d strings left at EOF", file, lineno, len(input))
//TODO 	}
//TODO 	t.Logf("%d cases tested", ncase)
//TODO }
//TODO
//TODO var run = []func(*Regexp, *Regexp, string) ([]int, string){
//TODO 	runFull,
//TODO 	runPartial,
//TODO 	runFullLongest,
//TODO 	runPartialLongest,
//TODO }
//TODO
//TODO func runFull(re, refull *Regexp, text string) ([]int, string) {
//TODO 	refull.longest = false
//TODO 	return refull.FindStringSubmatchIndex(text), "[full]"
//TODO }
//TODO
//TODO func runPartial(re, refull *Regexp, text string) ([]int, string) {
//TODO 	re.longest = false
//TODO 	return re.FindStringSubmatchIndex(text), ""
//TODO }
//TODO
//TODO func runFullLongest(re, refull *Regexp, text string) ([]int, string) {
//TODO 	refull.longest = true
//TODO 	return refull.FindStringSubmatchIndex(text), "[full,longest]"
//TODO }
//TODO
//TODO func runPartialLongest(re, refull *Regexp, text string) ([]int, string) {
//TODO 	re.longest = true
//TODO 	return re.FindStringSubmatchIndex(text), "[longest]"
//TODO }
//TODO
//TODO var match = []func(*Regexp, *Regexp, string) (bool, string){
//TODO 	matchFull,
//TODO 	matchPartial,
//TODO 	matchFullLongest,
//TODO 	matchPartialLongest,
//TODO }
//TODO
//TODO func matchFull(re, refull *Regexp, text string) (bool, string) {
//TODO 	refull.longest = false
//TODO 	return refull.MatchString(text), "[full]"
//TODO }
//TODO
//TODO func matchPartial(re, refull *Regexp, text string) (bool, string) {
//TODO 	re.longest = false
//TODO 	return re.MatchString(text), ""
//TODO }
//TODO
//TODO func matchFullLongest(re, refull *Regexp, text string) (bool, string) {
//TODO 	refull.longest = true
//TODO 	return refull.MatchString(text), "[full,longest]"
//TODO }
//TODO
//TODO func matchPartialLongest(re, refull *Regexp, text string) (bool, string) {
//TODO 	re.longest = true
//TODO 	return re.MatchString(text), "[longest]"
//TODO }
//TODO
//TODO func isSingleBytes(s string) bool {
//TODO 	for _, c := range s {
//TODO 		if c >= utf8.RuneSelf {
//TODO 			return false
//TODO 		}
//TODO 	}
//TODO 	return true
//TODO }
//TODO
//TODO func tryCompile(s string) (re *Regexp, err error) {
//TODO 	// Protect against panic during Compile.
//TODO 	defer func() {
//TODO 		if r := recover(); r != nil {
//TODO 			err = fmt.Errorf("panic: %v", r)
//TODO 		}
//TODO 	}()
//TODO 	return Compile(s)
//TODO }
//TODO
//TODO func parseResult(t *testing.T, file string, lineno int, res string) []int {
//TODO 	// A single - indicates no match.
//TODO 	if res == "-" {
//TODO 		return nil
//TODO 	}
//TODO 	// Otherwise, a space-separated list of pairs.
//TODO 	n := 1
//TODO 	for j := 0; j < len(res); j++ {
//TODO 		if res[j] == ' ' {
//TODO 			n++
//TODO 		}
//TODO 	}
//TODO 	out := make([]int, 2*n)
//TODO 	i := 0
//TODO 	n = 0
//TODO 	for j := 0; j <= len(res); j++ {
//TODO 		if j == len(res) || res[j] == ' ' {
//TODO 			// Process a single pair.  - means no submatch.
//TODO 			pair := res[i:j]
//TODO 			if pair == "-" {
//TODO 				out[n] = -1
//TODO 				out[n+1] = -1
//TODO 			} else {
//TODO 				k := strings.Index(pair, "-")
//TODO 				if k < 0 {
//TODO 					t.Fatalf("%s:%d: invalid pair %s", file, lineno, pair)
//TODO 				}
//TODO 				lo, err1 := strconv.Atoi(pair[:k])
//TODO 				hi, err2 := strconv.Atoi(pair[k+1:])
//TODO 				if err1 != nil || err2 != nil || lo > hi {
//TODO 					t.Fatalf("%s:%d: invalid pair %s", file, lineno, pair)
//TODO 				}
//TODO 				out[n] = lo
//TODO 				out[n+1] = hi
//TODO 			}
//TODO 			n += 2
//TODO 			i = j + 1
//TODO 		}
//TODO 	}
//TODO 	return out
//TODO }
//TODO
//TODO func same(x, y []int) bool {
//TODO 	if len(x) != len(y) {
//TODO 		return false
//TODO 	}
//TODO 	for i, xi := range x {
//TODO 		if xi != y[i] {
//TODO 			return false
//TODO 		}
//TODO 	}
//TODO 	return true
//TODO }
//TODO
//TODO // TestFowler runs this package's regexp API against the
//TODO // POSIX regular expression tests collected by Glenn Fowler
//TODO // at http://www2.research.att.com/~astopen/testregex/testregex.html.
//TODO func TestFowler(t *testing.T) {
//TODO 	files, err := filepath.Glob("testdata/*.dat")
//TODO 	if err != nil {
//TODO 		t.Fatal(err)
//TODO 	}
//TODO 	for _, file := range files {
//TODO 		t.Log(file)
//TODO 		testFowler(t, file)
//TODO 	}
//TODO }
//TODO
//TODO var notab = MustCompilePOSIX(`[^\t]+`)
//TODO
//TODO func testFowler(t *testing.T, file string) {
//TODO 	f, err := os.Open(file)
//TODO 	if err != nil {
//TODO 		t.Error(err)
//TODO 		return
//TODO 	}
//TODO 	defer f.Close()
//TODO 	b := bufio.NewReader(f)
//TODO 	lineno := 0
//TODO 	lastRegexp := ""
//TODO Reading:
//TODO 	for {
//TODO 		lineno++
//TODO 		line, err := b.ReadString('\n')
//TODO 		if err != nil {
//TODO 			if err != io.EOF {
//TODO 				t.Errorf("%s:%d: %v", file, lineno, err)
//TODO 			}
//TODO 			break Reading
//TODO 		}
//TODO
//TODO 		// http://www2.research.att.com/~astopen/man/man1/testregex.html
//TODO 		//
//TODO 		// INPUT FORMAT
//TODO 		//   Input lines may be blank, a comment beginning with #, or a test
//TODO 		//   specification. A specification is five fields separated by one
//TODO 		//   or more tabs. NULL denotes the empty string and NIL denotes the
//TODO 		//   0 pointer.
//TODO 		if line[0] == '#' || line[0] == '\n' {
//TODO 			continue Reading
//TODO 		}
//TODO 		line = line[:len(line)-1]
//TODO 		field := notab.FindAllString(line, -1)
//TODO 		for i, f := range field {
//TODO 			if f == "NULL" {
//TODO 				field[i] = ""
//TODO 			}
//TODO 			if f == "NIL" {
//TODO 				t.Logf("%s:%d: skip: %s", file, lineno, line)
//TODO 				continue Reading
//TODO 			}
//TODO 		}
//TODO 		if len(field) == 0 {
//TODO 			continue Reading
//TODO 		}
//TODO
//TODO 		//   Field 1: the regex(3) flags to apply, one character per REG_feature
//TODO 		//   flag. The test is skipped if REG_feature is not supported by the
//TODO 		//   implementation. If the first character is not [BEASKLP] then the
//TODO 		//   specification is a global control line. One or more of [BEASKLP] may be
//TODO 		//   specified; the test will be repeated for each mode.
//TODO 		//
//TODO 		//     B 	basic			BRE	(grep, ed, sed)
//TODO 		//     E 	REG_EXTENDED		ERE	(egrep)
//TODO 		//     A	REG_AUGMENTED		ARE	(egrep with negation)
//TODO 		//     S	REG_SHELL		SRE	(sh glob)
//TODO 		//     K	REG_SHELL|REG_AUGMENTED	KRE	(ksh glob)
//TODO 		//     L	REG_LITERAL		LRE	(fgrep)
//TODO 		//
//TODO 		//     a	REG_LEFT|REG_RIGHT	implicit ^...$
//TODO 		//     b	REG_NOTBOL		lhs does not match ^
//TODO 		//     c	REG_COMMENT		ignore space and #...\n
//TODO 		//     d	REG_SHELL_DOT		explicit leading . match
//TODO 		//     e	REG_NOTEOL		rhs does not match $
//TODO 		//     f	REG_MULTIPLE		multiple \n separated patterns
//TODO 		//     g	FNM_LEADING_DIR		testfnmatch only -- match until /
//TODO 		//     h	REG_MULTIREF		multiple digit backref
//TODO 		//     i	REG_ICASE		ignore case
//TODO 		//     j	REG_SPAN		. matches \n
//TODO 		//     k	REG_ESCAPE		\ to escape [...] delimiter
//TODO 		//     l	REG_LEFT		implicit ^...
//TODO 		//     m	REG_MINIMAL		minimal match
//TODO 		//     n	REG_NEWLINE		explicit \n match
//TODO 		//     o	REG_ENCLOSED		(|&) magic inside [@|&](...)
//TODO 		//     p	REG_SHELL_PATH		explicit / match
//TODO 		//     q	REG_DELIMITED		delimited pattern
//TODO 		//     r	REG_RIGHT		implicit ...$
//TODO 		//     s	REG_SHELL_ESCAPED	\ not special
//TODO 		//     t	REG_MUSTDELIM		all delimiters must be specified
//TODO 		//     u	standard unspecified behavior -- errors not counted
//TODO 		//     v	REG_CLASS_ESCAPE	\ special inside [...]
//TODO 		//     w	REG_NOSUB		no subexpression match array
//TODO 		//     x	REG_LENIENT		let some errors slide
//TODO 		//     y	REG_LEFT		regexec() implicit ^...
//TODO 		//     z	REG_NULL		NULL subexpressions ok
//TODO 		//     $	                        expand C \c escapes in fields 2 and 3
//TODO 		//     /	                        field 2 is a regsubcomp() expression
//TODO 		//     =	                        field 3 is a regdecomp() expression
//TODO 		//
//TODO 		//   Field 1 control lines:
//TODO 		//
//TODO 		//     C		set LC_COLLATE and LC_CTYPE to locale in field 2
//TODO 		//
//TODO 		//     ?test ...	output field 5 if passed and != EXPECTED, silent otherwise
//TODO 		//     &test ...	output field 5 if current and previous passed
//TODO 		//     |test ...	output field 5 if current passed and previous failed
//TODO 		//     ; ...	output field 2 if previous failed
//TODO 		//     {test ...	skip if failed until }
//TODO 		//     }		end of skip
//TODO 		//
//TODO 		//     : comment		comment copied as output NOTE
//TODO 		//     :comment:test	:comment: ignored
//TODO 		//     N[OTE] comment	comment copied as output NOTE
//TODO 		//     T[EST] comment	comment
//TODO 		//
//TODO 		//     number		use number for nmatch (20 by default)
//TODO 		flag := field[0]
//TODO 		switch flag[0] {
//TODO 		case '?', '&', '|', ';', '{', '}':
//TODO 			// Ignore all the control operators.
//TODO 			// Just run everything.
//TODO 			flag = flag[1:]
//TODO 			if flag == "" {
//TODO 				continue Reading
//TODO 			}
//TODO 		case ':':
//TODO 			i := strings.Index(flag[1:], ":")
//TODO 			if i < 0 {
//TODO 				t.Logf("skip: %s", line)
//TODO 				continue Reading
//TODO 			}
//TODO 			flag = flag[1+i+1:]
//TODO 		case 'C', 'N', 'T', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
//TODO 			t.Logf("skip: %s", line)
//TODO 			continue Reading
//TODO 		}
//TODO
//TODO 		// Can check field count now that we've handled the myriad comment formats.
//TODO 		if len(field) < 4 {
//TODO 			t.Errorf("%s:%d: too few fields: %s", file, lineno, line)
//TODO 			continue Reading
//TODO 		}
//TODO
//TODO 		// Expand C escapes (a.k.a. Go escapes).
//TODO 		if strings.Contains(flag, "$") {
//TODO 			f := `"` + field[1] + `"`
//TODO 			if field[1], err = strconv.Unquote(f); err != nil {
//TODO 				t.Errorf("%s:%d: cannot unquote %s", file, lineno, f)
//TODO 			}
//TODO 			f = `"` + field[2] + `"`
//TODO 			if field[2], err = strconv.Unquote(f); err != nil {
//TODO 				t.Errorf("%s:%d: cannot unquote %s", file, lineno, f)
//TODO 			}
//TODO 		}
//TODO
//TODO 		//   Field 2: the regular expression pattern; SAME uses the pattern from
//TODO 		//     the previous specification.
//TODO 		//
//TODO 		if field[1] == "SAME" {
//TODO 			field[1] = lastRegexp
//TODO 		}
//TODO 		lastRegexp = field[1]
//TODO
//TODO 		//   Field 3: the string to match.
//TODO 		text := field[2]
//TODO
//TODO 		//   Field 4: the test outcome...
//TODO 		ok, shouldCompile, shouldMatch, pos := parseFowlerResult(field[3])
//TODO 		if !ok {
//TODO 			t.Errorf("%s:%d: cannot parse result %#q", file, lineno, field[3])
//TODO 			continue Reading
//TODO 		}
//TODO
//TODO 		//   Field 5: optional comment appended to the report.
//TODO
//TODO 	Testing:
//TODO 		// Run test once for each specified capital letter mode that we support.
//TODO 		for _, c := range flag {
//TODO 			pattern := field[1]
//TODO 			syn := syntax.POSIX | syntax.ClassNL
//TODO 			switch c {
//TODO 			default:
//TODO 				continue Testing
//TODO 			case 'E':
//TODO 				// extended regexp (what we support)
//TODO 			case 'L':
//TODO 				// literal
//TODO 				pattern = QuoteMeta(pattern)
//TODO 			}
//TODO
//TODO 			for _, c := range flag {
//TODO 				switch c {
//TODO 				case 'i':
//TODO 					syn |= syntax.FoldCase
//TODO 				}
//TODO 			}
//TODO
//TODO 			re, err := compile(pattern, syn, true)
//TODO 			if err != nil {
//TODO 				if shouldCompile {
//TODO 					t.Errorf("%s:%d: %#q did not compile", file, lineno, pattern)
//TODO 				}
//TODO 				continue Testing
//TODO 			}
//TODO 			if !shouldCompile {
//TODO 				t.Errorf("%s:%d: %#q should not compile", file, lineno, pattern)
//TODO 				continue Testing
//TODO 			}
//TODO 			match := re.MatchString(text)
//TODO 			if match != shouldMatch {
//TODO 				t.Errorf("%s:%d: %#q.Match(%#q) = %v, want %v", file, lineno, pattern, text, match, shouldMatch)
//TODO 				continue Testing
//TODO 			}
//TODO 			have := re.FindStringSubmatchIndex(text)
//TODO 			if (len(have) > 0) != match {
//TODO 				t.Errorf("%s:%d: %#q.Match(%#q) = %v, but %#q.FindSubmatchIndex(%#q) = %v", file, lineno, pattern, text, match, pattern, text, have)
//TODO 				continue Testing
//TODO 			}
//TODO 			if len(have) > len(pos) {
//TODO 				have = have[:len(pos)]
//TODO 			}
//TODO 			if !same(have, pos) {
//TODO 				t.Errorf("%s:%d: %#q.FindSubmatchIndex(%#q) = %v, want %v", file, lineno, pattern, text, have, pos)
//TODO 			}
//TODO 		}
//TODO 	}
//TODO }
//TODO
//TODO func parseFowlerResult(s string) (ok, compiled, matched bool, pos []int) {
//TODO 	//   Field 4: the test outcome. This is either one of the posix error
//TODO 	//     codes (with REG_ omitted) or the match array, a list of (m,n)
//TODO 	//     entries with m and n being first and last+1 positions in the
//TODO 	//     field 3 string, or NULL if REG_NOSUB is in effect and success
//TODO 	//     is expected. BADPAT is acceptable in place of any regcomp(3)
//TODO 	//     error code. The match[] array is initialized to (-2,-2) before
//TODO 	//     each test. All array elements from 0 to nmatch-1 must be specified
//TODO 	//     in the outcome. Unspecified endpoints (offset -1) are denoted by ?.
//TODO 	//     Unset endpoints (offset -2) are denoted by X. {x}(o:n) denotes a
//TODO 	//     matched (?{...}) expression, where x is the text enclosed by {...},
//TODO 	//     o is the expression ordinal counting from 1, and n is the length of
//TODO 	//     the unmatched portion of the subject string. If x starts with a
//TODO 	//     number then that is the return value of re_execf(), otherwise 0 is
//TODO 	//     returned.
//TODO 	switch {
//TODO 	case s == "":
//TODO 		// Match with no position information.
//TODO 		ok = true
//TODO 		compiled = true
//TODO 		matched = true
//TODO 		return
//TODO 	case s == "NOMATCH":
//TODO 		// Match failure.
//TODO 		ok = true
//TODO 		compiled = true
//TODO 		matched = false
//TODO 		return
//TODO 	case 'A' <= s[0] && s[0] <= 'Z':
//TODO 		// All the other error codes are compile errors.
//TODO 		ok = true
//TODO 		compiled = false
//TODO 		return
//TODO 	}
//TODO 	compiled = true
//TODO
//TODO 	var x []int
//TODO 	for s != "" {
//TODO 		var end byte = ')'
//TODO 		if len(x)%2 == 0 {
//TODO 			if s[0] != '(' {
//TODO 				ok = false
//TODO 				return
//TODO 			}
//TODO 			s = s[1:]
//TODO 			end = ','
//TODO 		}
//TODO 		i := 0
//TODO 		for i < len(s) && s[i] != end {
//TODO 			i++
//TODO 		}
//TODO 		if i == 0 || i == len(s) {
//TODO 			ok = false
//TODO 			return
//TODO 		}
//TODO 		var v = -1
//TODO 		var err error
//TODO 		if s[:i] != "?" {
//TODO 			v, err = strconv.Atoi(s[:i])
//TODO 			if err != nil {
//TODO 				ok = false
//TODO 				return
//TODO 			}
//TODO 		}
//TODO 		x = append(x, v)
//TODO 		s = s[i+1:]
//TODO 	}
//TODO 	if len(x)%2 != 0 {
//TODO 		ok = false
//TODO 		return
//TODO 	}
//TODO 	ok = true
//TODO 	matched = true
//TODO 	pos = x
//TODO 	return
//TODO }
//TODO
//TODO var text []byte
//TODO
//TODO func makeText(n int) []byte {
//TODO 	if len(text) >= n {
//TODO 		return text[:n]
//TODO 	}
//TODO 	text = make([]byte, n)
//TODO 	x := ^uint32(0)
//TODO 	for i := range text {
//TODO 		x += x
//TODO 		x ^= 1
//TODO 		if int32(x) < 0 {
//TODO 			x ^= 0x88888eef
//TODO 		}
//TODO 		if x%31 == 0 {
//TODO 			text[i] = '\n'
//TODO 		} else {
//TODO 			text[i] = byte(x%(0x7E+1-0x20) + 0x20)
//TODO 		}
//TODO 	}
//TODO 	return text
//TODO }
//TODO
//TODO func BenchmarkMatch(b *testing.B) {
//TODO 	isRaceBuilder := strings.HasSuffix(testenv.Builder(), "-race")
//TODO
//TODO 	for _, data := range benchData {
//TODO 		r := MustCompile(data.re)
//TODO 		for _, size := range benchSizes {
//TODO 			if isRaceBuilder && size.n > 1<<10 {
//TODO 				continue
//TODO 			}
//TODO 			t := makeText(size.n)
//TODO 			b.Run(data.name+"/"+size.name, func(b *testing.B) {
//TODO 				b.SetBytes(int64(size.n))
//TODO 				for i := 0; i < b.N; i++ {
//TODO 					if r.Match(t) {
//TODO 						b.Fatal("match!")
//TODO 					}
//TODO 				}
//TODO 			})
//TODO 		}
//TODO 	}
//TODO }
//TODO
//TODO var benchData = []struct{ name, re string }{
//TODO 	{"Easy0", "ABCDEFGHIJKLMNOPQRSTUVWXYZ$"},
//TODO 	{"Easy0i", "(?i)ABCDEFGHIJklmnopqrstuvwxyz$"},
//TODO 	{"Easy1", "A[AB]B[BC]C[CD]D[DE]E[EF]F[FG]G[GH]H[HI]I[IJ]J$"},
//TODO 	{"Medium", "[XYZ]ABCDEFGHIJKLMNOPQRSTUVWXYZ$"},
//TODO 	{"Hard", "[ -~]*ABCDEFGHIJKLMNOPQRSTUVWXYZ$"},
//TODO 	{"Hard1", "ABCD|CDEF|EFGH|GHIJ|IJKL|KLMN|MNOP|OPQR|QRST|STUV|UVWX|WXYZ"},
//TODO }
//TODO
//TODO var benchSizes = []struct {
//TODO 	name string
//TODO 	n    int
//TODO }{
//TODO 	{"32", 32},
//TODO 	{"1K", 1 << 10},
//TODO 	{"32K", 32 << 10},
//TODO 	{"1M", 1 << 20},
//TODO 	{"32M", 32 << 20},
//TODO }
//TODO
//TODO func TestLongest(t *testing.T) {
//TODO 	re, err := Compile(`a(|b)`)
//TODO 	if err != nil {
//TODO 		t.Fatal(err)
//TODO 	}
//TODO 	if g, w := re.FindString("ab"), "a"; g != w {
//TODO 		t.Errorf("first match was %q, want %q", g, w)
//TODO 	}
//TODO 	re.Longest()
//TODO 	if g, w := re.FindString("ab"), "ab"; g != w {
//TODO 		t.Errorf("longest match was %q, want %q", g, w)
//TODO 	}
//TODO }
//TODO
//TODO // TestProgramTooLongForBacktrack tests that a regex which is too long
//TODO // for the backtracker still executes properly.
//TODO func TestProgramTooLongForBacktrack(t *testing.T) {
//TODO 	longRegex := MustCompile(`(one|two|three|four|five|six|seven|eight|nine|ten|eleven|twelve|thirteen|fourteen|fifteen|sixteen|seventeen|eighteen|nineteen|twenty|twentyone|twentytwo|twentythree|twentyfour|twentyfive|twentysix|twentyseven|twentyeight|twentynine|thirty|thirtyone|thirtytwo|thirtythree|thirtyfour|thirtyfive|thirtysix|thirtyseven|thirtyeight|thirtynine|forty|fortyone|fortytwo|fortythree|fortyfour|fortyfive|fortysix|fortyseven|fortyeight|fortynine|fifty|fiftyone|fiftytwo|fiftythree|fiftyfour|fiftyfive|fiftysix|fiftyseven|fiftyeight|fiftynine|sixty|sixtyone|sixtytwo|sixtythree|sixtyfour|sixtyfive|sixtysix|sixtyseven|sixtyeight|sixtynine|seventy|seventyone|seventytwo|seventythree|seventyfour|seventyfive|seventysix|seventyseven|seventyeight|seventynine|eighty|eightyone|eightytwo|eightythree|eightyfour|eightyfive|eightysix|eightyseven|eightyeight|eightynine|ninety|ninetyone|ninetytwo|ninetythree|ninetyfour|ninetyfive|ninetysix|ninetyseven|ninetyeight|ninetynine|onehundred)`)
//TODO 	if !longRegex.MatchString("two") {
//TODO 		t.Errorf("longRegex.MatchString(\"two\") was false, want true")
//TODO 	}
//TODO 	if longRegex.MatchString("xxx") {
//TODO 		t.Errorf("longRegex.MatchString(\"xxx\") was true, want false")
//TODO 	}
//TODO }