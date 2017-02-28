// Copyright 2017 The Regexp Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Source code and documentation of this package contains copied and/or
// modified source code and documentation from the stdlib Go package:
//
// ============================================================================
// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the GO-LICENSE file.
// ============================================================================

// Package regexp implements regular expression search.
//
// The syntax of the regular expressions accepted is the same
// general syntax used by Perl, Python, and other languages.
// More precisely, it is the syntax accepted by RE2 and described at
// https://golang.org/s/re2syntax, except for \C.
// For an overview of the syntax, run
//   go doc regexp/syntax
//
// The regexp implementation provided by this package is
// guaranteed to run in time linear in the size of the input.
// (This is a property not guaranteed by most open source
// implementations of regular expressions.) For more information
// about this property, see
//	http://swtch.com/~rsc/regexp/regexp1.html
// or any book about automata theory.
//
// All characters are UTF-8-encoded code points.
//
// There are 16 methods of Regexp that match a regular expression and identify
// the matched text. Their names are matched by this regular expression:
//
//	Find(All)?(String)?(Submatch)?(Index)?
//
// If 'All' is present, the routine matches successive non-overlapping
// matches of the entire expression. Empty matches abutting a preceding
// match are ignored. The return value is a slice containing the successive
// return values of the corresponding non-'All' routine. These routines take
// an extra integer argument, n; if n >= 0, the function returns at most n
// matches/submatches.
//
// If 'String' is present, the argument is a string; otherwise it is a slice
// of bytes; return values are adjusted as appropriate.
//
// If 'Submatch' is present, the return value is a slice identifying the
// successive submatches of the expression. Submatches are matches of
// parenthesized subexpressions (also known as capturing groups) within the
// regular expression, numbered from left to right in order of opening
// parenthesis. Submatch 0 is the match of the entire expression, submatch 1
// the match of the first parenthesized subexpression, and so on.
//
// If 'Index' is present, matches and submatches are identified by byte index
// pairs within the input string: result[2*n:2*n+1] identifies the indexes of
// the nth submatch. The pair for n==0 identifies the match of the entire
// expression. If 'Index' is not present, the match is identified by the
// text of the match/submatch. If an index is negative, it means that
// subexpression did not match any string in the input.
//
// There is also a subset of the methods that can be applied to text read
// from a RuneReader:
//
//	MatchReader, FindReaderIndex, FindReaderSubmatchIndex
//
// This set may grow. Note that regular expression matches may need to
// examine text beyond the text returned by a match, so the methods that
// match text from a RuneReader may read arbitrarily far into the input
// before returning.
//
// (There are a few other methods that do not match this pattern.)
//
package regexp

import (
	"bytes"
	"io"
	"regexp/syntax"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/cznic/internal/buffer"
)

const (
	maxBacktrackVector = 256 * 1024
	maxProg            = 1e4  // Prevent x{1000}{1000}.
	maxRepCount        = 1000 // Prevent x{1001}.
)

func compile(expr string, mode syntax.Flags, longest bool) (*Regexp, error) {
	p := newParser(expr, newRegexp(expr))
	re, err := p.parse()
	if err != nil {
		return nil, err
	}

	re.longest = longest
	return re, nil
}

// Compile parses a regular expression and returns, if successful, a Regexp
// object that can be used to match against text.
//
// When matching against text, the regexp returns a match that begins as early
// as possible in the input (leftmost), and among those it chooses the one that
// a backtracking search would have found first. This so-called leftmost-first
// matching is the same semantics that Perl, Python, and other implementations
// use, although this package implements it without the expense of
// backtracking. For POSIX leftmost-longest matching, see CompilePOSIX.
func Compile(expr string) (*Regexp, error) { return compile(expr, syntax.Perl, false) }

// CompilePOSIX is like Compile but restricts the regular expression
// to POSIX ERE (egrep) syntax and changes the match semantics to
// leftmost-longest.
//
// That is, when matching against text, the regexp returns a match that
// begins as early as possible in the input (leftmost), and among those
// it chooses a match that is as long as possible.
// This so-called leftmost-longest matching is the same semantics
// that early regular expression implementations used and that POSIX
// specifies.
//
// However, there can be multiple leftmost-longest matches, with different
// submatch choices, and here this package diverges from POSIX.
// Among the possible leftmost-longest matches, this package chooses
// the one that a backtracking search would have found first, while POSIX
// specifies that the match be chosen to maximize the length of the first
// subexpression, then the second, and so on from left to right.
// The POSIX rule is computationally prohibitive and not even well-defined.
// See http://swtch.com/~rsc/regexp/regexp2.html#posix for details.
func CompilePOSIX(expr string) (*Regexp, error) { return compile(expr, syntax.POSIX, false) }

// MatchString checks whether a textual regular expression matches a string.
// More complicated queries need to use Compile and the full Regexp interface.
func MatchString(pattern string, s string) (matched bool, err error) {
	re, err := Compile(pattern)
	if err != nil {
		return false, err
	}

	return re.MatchString(s), nil
}

// Longest makes future searches prefer the leftmost-longest match.
// That is, when matching against text, the regexp returns a match that
// begins as early as possible in the input (leftmost), and among those
// it chooses a match that is as long as possible.
func (re *Regexp) Longest() {
	re.longestMu.Lock()
	re.longest = true
	re.longestMu.Unlock()
}

// Copy returns a new Regexp object copied from re.
//
// When using a Regexp in multiple goroutines, giving each goroutine
// its own copy helps to avoid lock contention.
func (re *Regexp) Copy() *Regexp {
	re.longestMu.Lock()
	x := *re
	re.longestMu.Unlock()
	x.longestMu = &sync.Mutex{}
	return &x
}

// Find returns a slice holding the text of the leftmost match in b of the
// regular expression. A return value of nil indicates no match.
func (re *Regexp) Find(b []byte) []byte {
	if a := re.FindIndex(b); a != nil {
		return b[a[0]:a[1]]
	}

	return nil
}

func (re *Regexp) findAllIndex(rd io.RuneReader, n int) [][]int {
	vm := newVM(re, rd)
	var r [][]int
	var prev []int
	for vm.c != pastEOF && len(r) != n {
		a := vm.find()
		if a == nil {
			return r
		}

		// If 'All' is present, the routine matches successive
		// non-overlapping matches of the entire expression.  Empty
		// matches abutting a preceding match are ignored.
		if prev == nil || a[0] != a[1] || prev[0] == prev[1] {
			r = append(r, a[:2])
		}
		prev = a
	}
	return r
}

// FindAll is the 'All' version of Find; it returns a slice of all successive
// matches of the expression, as defined by the 'All' description in the
// package comment. A return value of nil indicates no match.
func (re *Regexp) FindAll(b []byte, n int) [][]byte {
	var r [][]byte
	for _, a := range re.findAllIndex(bytes.NewReader(b), n) {
		r = append(r, b[a[0]:a[1]])
	}
	return r
}

// FindAllIndex is the 'All' version of FindIndex; it returns a slice of all
// successive matches of the expression, as defined by the 'All' description in
// the package comment. A return value of nil indicates no match.
func (re *Regexp) FindAllIndex(b []byte, n int) [][]int { return re.findAllIndex(bytes.NewReader(b), n) }

// FindAllString is the 'All' version of FindString; it returns a slice of all
// successive matches of the expression, as defined by the 'All' description in
// the package comment. A return value of nil indicates no match.
func (re *Regexp) FindAllString(s string, n int) []string {
	var r []string
	for _, a := range re.findAllIndex(strings.NewReader(s), n) {
		r = append(r, s[a[0]:a[1]])
	}
	return r
}

// FindAllStringIndex is the 'All' version of FindStringIndex; it returns a
// slice of all successive matches of the expression, as defined by the 'All'
// description in the package comment. A return value of nil indicates no
// match.
func (re *Regexp) FindAllStringIndex(s string, n int) [][]int {
	return re.findAllIndex(strings.NewReader(s), n)
}

// FindAllStringSubmatch is the 'All' version of FindStringSubmatch; it returns
// a slice of all successive matches of the expression, as defined by the 'All'
// description in the package comment. A return value of nil indicates no
// match.
func (re *Regexp) FindAllStringSubmatch(s string, n int) [][]string {
	var r [][]string
	for _, a := range re.FindAllStringSubmatchIndex(s, n) {
		var t []string
		for i := 0; i < len(a); i += 2 {
			switch lo := a[i]; {
			case lo < 0:
				t = append(t, "")
			default:
				t = append(t, s[a[i]:a[i+1]])
			}
		}
		r = append(r, t)
	}
	return r
}

// FindAllStringSubmatchIndex is the 'All' version of FindStringSubmatchIndex;
// it returns a slice of all successive matches of the expression, as defined
// by the 'All' description in the package comment. A return value of nil
// indicates no match.
func (re *Regexp) FindAllStringSubmatchIndex(s string, n int) [][]int {
	return re.findAllSubmatchIndex(strings.NewReader(s), n)
}

func (re *Regexp) findAllSubmatchIndex(rd io.RuneReader, n int) [][]int {
	vm := newVM(re, rd)
	var r [][]int
	var prev []int
	for vm.c != pastEOF && len(r) != n {
		a := vm.find()
		if a == nil {
			return r
		}

		// If 'All' is present, the routine matches successive
		// non-overlapping matches of the entire expression.  Empty
		// matches abutting a preceding match are ignored.
		if prev == nil || a[0] != a[1] || prev[0] == prev[1] {
			r = append(r, a)
		}
		prev = a
	}
	return r
}

// FindAllSubmatch is the 'All' version of FindSubmatch; it returns a slice of
// all successive matches of the expression, as defined by the 'All'
// description in the package comment. A return value of nil indicates no
// match.
func (re *Regexp) FindAllSubmatch(b []byte, n int) [][][]byte {
	var r [][][]byte
	for _, a := range re.FindAllSubmatchIndex(b, n) {
		var t [][]byte
		for i := 0; i < len(a); i += 2 {
			switch lo := a[i]; {
			case lo < 0:
				t = append(t, nil)
			default:
				t = append(t, b[a[i]:a[i+1]])
			}
		}
		r = append(r, t)
	}
	return r
}

// FindAllSubmatchIndex is the 'All' version of FindSubmatchIndex; it returns a
// slice of all successive matches of the expression, as defined by the 'All'
// description in the package comment. A return value of nil indicates no
// match.
func (re *Regexp) FindAllSubmatchIndex(b []byte, n int) [][]int {
	return re.findAllSubmatchIndex(bytes.NewReader(b), n)
}

// FindIndex returns a two-element slice of integers defining the location of
// the leftmost match in b of the regular expression. The match itself is at
// b[loc[0]:loc[1]]. A return value of nil indicates no match.
func (re *Regexp) FindIndex(b []byte) (loc []int) {
	if loc = re.FindSubmatchIndex(b); loc != nil {
		loc = loc[:2]
	}
	return loc
}

// FindReaderIndex returns a two-element slice of integers defining the
// location of the leftmost match of the regular expression in text read from
// the RuneReader. The match text was found in the input stream at byte offset
// loc[0] through loc[1]-1. A return value of nil indicates no match.
func (re *Regexp) FindReaderIndex(r io.RuneReader) (loc []int) {
	if loc = re.FindReaderSubmatchIndex(r); loc != nil {
		loc = loc[:2]
	}
	return loc
}

// FindReaderSubmatchIndex returns a slice holding the index pairs identifying
// the leftmost match of the regular expression of text read by the RuneReader,
// and the matches, if any, of its subexpressions, as defined by the 'Submatch'
// and 'Index' descriptions in the package comment. A return value of nil
// indicates no match.
func (re *Regexp) FindReaderSubmatchIndex(r io.RuneReader) []int { return newVM(re, r).find() }

// FindString returns a string holding the text of the leftmost match in s of
// the regular expression. If there is no match, the return value is an empty
// string, but it will also be empty if the regular expression successfully
// matches an empty string. Use FindStringIndex or FindStringSubmatch if it is
// necessary to distinguish these cases.
func (re *Regexp) FindString(s string) string {
	if a := re.FindStringIndex(s); a != nil {
		return s[a[0]:a[1]]
	}

	return ""
}

// FindStringIndex returns a two-element slice of integers defining the
// location of the leftmost match in s of the regular expression. The match
// itself is at s[loc[0]:loc[1]]. A return value of nil indicates no match.
func (re *Regexp) FindStringIndex(s string) (loc []int) {
	if loc = re.FindStringSubmatchIndex(s); loc != nil {
		loc = loc[:2]
	}
	return loc
}

// FindStringSubmatch returns a slice of strings holding the text of the
// leftmost match of the regular expression in s and the matches, if any, of
// its subexpressions, as defined by the 'Submatch' description in the package
// comment. A return value of nil indicates no match.
func (re *Regexp) FindStringSubmatch(s string) []string {
	if a := re.FindStringSubmatchIndex(s); a != nil {
		ret := make([]string, len(a)/2)
		for i := range ret {
			switch lo := a[2*i]; {
			case lo < 0:
				ret[i] = ""
			default:
				ret[i] = s[a[2*i]:a[2*i+1]]
			}
		}
		return ret
	}

	return nil
}

// FindStringSubmatchIndex returns a slice holding the index pairs identifying
// the leftmost match of the regular expression in s and the matches, if any,
// of its subexpressions, as defined by the 'Submatch' and 'Index' descriptions
// in the package comment. A return value of nil indicates no match.
func (re *Regexp) FindStringSubmatchIndex(s string) []int {
	return newVM(re, strings.NewReader(s)).find()
}

// FindSubmatch returns a slice of slices holding the text of the leftmost
// match of the regular expression in b and the matches, if any, of its
// subexpressions, as defined by the 'Submatch' descriptions in the package
// comment. A return value of nil indicates no match.
func (re *Regexp) FindSubmatch(b []byte) [][]byte {
	if a := re.FindSubmatchIndex(b); a != nil {
		ret := make([][]byte, len(a)/2)
		for i := range ret {
			switch lo := a[2*i]; {
			case lo < 0:
				ret[i] = nil
			default:
				ret[i] = b[a[2*i]:a[2*i+1]]
			}
		}
		return ret
	}

	return nil
}

// FindSubmatchIndex returns a slice holding the index pairs identifying the
// leftmost match of the regular expression in b and the matches, if any, of
// its subexpressions, as defined by the 'Submatch' and 'Index' descriptions in
// the package comment. A return value of nil indicates no match.
func (re *Regexp) FindSubmatchIndex(b []byte) []int { return newVM(re, bytes.NewReader(b)).find() }

// Match reports whether the Regexp matches the byte slice b.
func (re *Regexp) Match(b []byte) bool {
	return newVM(re, bytes.NewReader(b)).match()
}

// MatchString reports whether the Regexp matches the string s.
func (re *Regexp) MatchString(s string) bool {
	return newVM(re, strings.NewReader(s)).match()
}

// NumSubexp returns the number of parenthesized subexpressions in this Regexp.
func (re *Regexp) NumSubexp() int {
	return re.groups - 1
}

// SubexpNames returns the names of the parenthesized subexpressions
// in this Regexp. The name for the first sub-expression is names[1],
// so that if m is a match slice, the name for m[i] is SubexpNames()[i].
// Since the Regexp as a whole cannot be named, names[0] is always
// the empty string. The slice should not be modified.
func (re *Regexp) SubexpNames() []string {
	return re.groupNames
}

// Split slices s into substrings separated by the expression and returns a slice of
// the substrings between those expression matches.
//
// The slice returned by this method consists of all the substrings of s
// not contained in the slice returned by FindAllString. When called on an expression
// that contains no metacharacters, it is equivalent to strings.SplitN.
//
// Example:
//   s := regexp.MustCompile("a*").Split("abaabaccadaaae", 5)
//   // s: ["", "b", "b", "c", "cadaaae"]
//
// The count determines the number of substrings to return:
//   n > 0: at most n substrings; the last substring will be the unsplit remainder.
//   n == 0: the result is nil (zero substrings)
//   n < 0: all substrings
func (re *Regexp) Split(s string, n int) []string {

	if n == 0 {
		return nil
	}

	if len(re.src) > 0 && len(s) == 0 {
		return []string{""}
	}

	matches := re.FindAllStringIndex(s, n)
	strings := make([]string, 0, len(matches))

	beg := 0
	end := 0
	for _, match := range matches {
		if n > 0 && len(strings) >= n-1 {
			break
		}

		end = match[0]
		if match[1] != 0 {
			strings = append(strings, s[beg:end])
		}
		beg = match[1]
	}

	if end != len(s) {
		strings = append(strings, s[beg:])
	}

	return strings
}

// ReplaceAllLiteralString returns a copy of src, replacing matches of the Regexp
// with the replacement string repl. The replacement repl is substituted directly,
// without using Expand.
func (re *Regexp) ReplaceAllLiteralString(src, repl string) string {
	var out buffer.Bytes
	vm := newVM(re, strings.NewReader(src))
	pos := 0
	var prev []int
	for vm.c != pastEOF {
		a := vm.find()
		if a == nil {
			break
		}

		// If 'All' is present, the routine matches successive
		// non-overlapping matches of the entire expression.  Empty
		// matches abutting a preceding match are ignored.
		if prev == nil || a[0] != a[1] || prev[0] == prev[1] {
			first := a[0]
			if pos < first {
				out.WriteString(src[pos:first])
			}
			out.WriteString(repl)
			pos = a[1]
		}
		prev = a
	}
	if pos < len(src) {
		out.WriteString(src[pos:])
	}
	return string(out.Bytes())
}

// ReplaceAllLiteral returns a copy of src, replacing matches of the Regexp
// with the replacement bytes repl. The replacement repl is substituted directly,
// without using Expand.
func (re *Regexp) ReplaceAllLiteral(src, repl []byte) []byte {
	var out buffer.Bytes
	vm := newVM(re, bytes.NewReader(src))
	pos := 0
	var prev []int
	for vm.c != pastEOF {
		a := vm.find()
		if a == nil {
			break
		}

		// If 'All' is present, the routine matches successive
		// non-overlapping matches of the entire expression.  Empty
		// matches abutting a preceding match are ignored.
		if prev == nil || a[0] != a[1] || prev[0] == prev[1] {
			first := a[0]
			if pos < first {
				out.Write(src[pos:first])
			}
			out.Write(repl)
			pos = a[1]
		}
		prev = a
	}
	if pos < len(src) {
		out.Write(src[pos:])
	}
	return out.Bytes()
}

// ReplaceAllString returns a copy of src, replacing matches of the Regexp
// with the replacement string repl. Inside repl, $ signs are interpreted as
// in Expand, so for instance $1 represents the text of the first submatch.
func (re *Regexp) ReplaceAllString(src, repl string) string {
	var out buffer.Bytes
	vm := newVM(re, strings.NewReader(src))
	pos := 0
	var prev []int
	for vm.c != pastEOF {
		a := vm.find()
		if a == nil {
			break
		}

		// If 'All' is present, the routine matches successive
		// non-overlapping matches of the entire expression.  Empty
		// matches abutting a preceding match are ignored.
		if prev == nil || a[0] != a[1] || prev[0] == prev[1] {
			first := a[0]
			if pos < first {
				out.WriteString(src[pos:first])
			}
			out.Write(re.expand(nil, repl, nil, src, a))
			pos = a[1]
		}
		prev = a
	}
	if pos < len(src) {
		out.WriteString(src[pos:])
	}
	return string(out.Bytes())
}

// ReplaceAll returns a copy of src, replacing matches of the Regexp
// with the replacement text repl. Inside repl, $ signs are interpreted as
// in Expand, so for instance $1 represents the text of the first submatch.
func (re *Regexp) ReplaceAll(src, repl []byte) []byte {
	srepl := string(repl)
	var out buffer.Bytes
	vm := newVM(re, bytes.NewReader(src))
	pos := 0
	var prev []int
	for vm.c != pastEOF {
		a := vm.find()
		if a == nil {
			break
		}

		// If 'All' is present, the routine matches successive
		// non-overlapping matches of the entire expression.  Empty
		// matches abutting a preceding match are ignored.
		if prev == nil || a[0] != a[1] || prev[0] == prev[1] {
			first := a[0]
			if pos < first {
				out.Write(src[pos:first])
			}
			out.Write(re.expand(nil, srepl, src, "", a))
			pos = a[1]
		}
		prev = a
	}
	if pos < len(src) {
		out.Write(src[pos:])
	}
	return out.Bytes()
}

// ReplaceAllStringFunc returns a copy of src in which all matches of the
// Regexp have been replaced by the return value of function repl applied
// to the matched substring. The replacement returned by repl is substituted
// directly, without using Expand.
func (re *Regexp) ReplaceAllStringFunc(src string, repl func(string) string) string {
	var out buffer.Bytes
	vm := newVM(re, strings.NewReader(src))
	pos := 0
	var prev []int
	for vm.c != pastEOF {
		a := vm.find()
		if a == nil {
			break
		}

		// If 'All' is present, the routine matches successive
		// non-overlapping matches of the entire expression.  Empty
		// matches abutting a preceding match are ignored.
		if prev == nil || a[0] != a[1] || prev[0] == prev[1] {
			first := a[0]
			if pos < first {
				out.WriteString(src[pos:first])
			}
			pos = a[1]
			out.WriteString(repl(src[first:pos]))
		}
		prev = a
	}
	if pos < len(src) {
		out.WriteString(src[pos:])
	}
	return string(out.Bytes())
}

// ReplaceAllFunc returns a copy of src in which all matches of the
// Regexp have been replaced by the return value of function repl applied
// to the matched byte slice. The replacement returned by repl is substituted
// directly, without using Expand.
func (re *Regexp) ReplaceAllFunc(src []byte, repl func([]byte) []byte) []byte {
	var out buffer.Bytes
	vm := newVM(re, bytes.NewReader(src))
	pos := 0
	var prev []int
	for vm.c != pastEOF {
		a := vm.find()
		if a == nil {
			break
		}

		// If 'All' is present, the routine matches successive
		// non-overlapping matches of the entire expression.  Empty
		// matches abutting a preceding match are ignored.
		if prev == nil || a[0] != a[1] || prev[0] == prev[1] {
			first := a[0]
			if pos < first {
				out.Write(src[pos:first])
			}
			pos = a[1]
			out.Write(repl(src[first:pos]))
		}
		prev = a
	}
	if pos < len(src) {
		out.Write(src[pos:])
	}
	return out.Bytes()
}

// Expand appends template to dst and returns the result; during the
// append, Expand replaces variables in the template with corresponding
// matches drawn from src. The match slice should have been returned by
// FindSubmatchIndex.
//
// In the template, a variable is denoted by a substring of the form
// $name or ${name}, where name is a non-empty sequence of letters,
// digits, and underscores. A purely numeric name like $1 refers to
// the submatch with the corresponding index; other names refer to
// capturing parentheses named with the (?P<name>...) syntax. A
// reference to an out of range or unmatched index or a name that is not
// present in the regular expression is replaced with an empty slice.
//
// In the $name form, name is taken to be as long as possible: $1x is
// equivalent to ${1x}, not ${1}x, and, $10 is equivalent to ${10}, not ${1}0.
//
// To insert a literal $ in the output, use $$ in the template.
func (re *Regexp) Expand(dst []byte, template []byte, src []byte, match []int) []byte {
	return re.expand(dst, string(template), src, "", match)
}

// ExpandString is like Expand but the template and source are strings.
// It appends to and returns a byte slice in order to give the calling
// code control over allocation.
func (re *Regexp) ExpandString(dst []byte, template string, src string, match []int) []byte {
	return re.expand(dst, template, nil, src, match)
}

// extract returns the name from a leading "$name" or "${name}" in str.
// If it is a number, extract returns num set to that number; otherwise num = -1.
func extract(str string) (name string, num int, rest string, ok bool) {
	if len(str) < 2 || str[0] != '$' {
		return
	}
	brace := false
	if str[1] == '{' {
		brace = true
		str = str[2:]
	} else {
		str = str[1:]
	}
	i := 0
	for i < len(str) {
		rune, size := utf8.DecodeRuneInString(str[i:])
		if !unicode.IsLetter(rune) && !unicode.IsDigit(rune) && rune != '_' {
			break
		}
		i += size
	}
	if i == 0 {
		// empty name is not okay
		return
	}
	name = str[:i]
	if brace {
		if i >= len(str) || str[i] != '}' {
			// missing closing brace
			return
		}
		i++
	}

	// Parse number.
	num = 0
	for i := 0; i < len(name); i++ {
		if name[i] < '0' || '9' < name[i] || num >= 1e8 {
			num = -1
			break
		}
		num = num*10 + int(name[i]) - '0'
	}
	// Disallow leading zeros.
	if name[0] == '0' && len(name) > 1 {
		num = -1
	}

	rest = str[i:]
	ok = true
	return
}

func (re *Regexp) expand(dst []byte, template string, bsrc []byte, src string, match []int) []byte {
	for len(template) > 0 {
		i := strings.Index(template, "$")
		if i < 0 {
			break
		}
		dst = append(dst, template[:i]...)
		template = template[i:]
		if len(template) > 1 && template[1] == '$' {
			// Treat $$ as $.
			dst = append(dst, '$')
			template = template[2:]
			continue
		}
		name, num, rest, ok := extract(template)
		if !ok {
			// Malformed; treat $ as raw text.
			dst = append(dst, '$')
			template = template[1:]
			continue
		}
		template = rest
		if num >= 0 {
			if 2*num+1 < len(match) && match[2*num] >= 0 {
				if bsrc != nil {
					dst = append(dst, bsrc[match[2*num]:match[2*num+1]]...)
				} else {
					dst = append(dst, src[match[2*num]:match[2*num+1]]...)
				}
			}
		} else {
			for i, namei := range re.groupNames {
				if name == namei && 2*i+1 < len(match) && match[2*i] >= 0 {
					if bsrc != nil {
						dst = append(dst, bsrc[match[2*i]:match[2*i+1]]...)
					} else {
						dst = append(dst, src[match[2*i]:match[2*i+1]]...)
					}
					break
				}
			}
		}
	}
	dst = append(dst, template...)
	return dst
}

// LiteralPrefix returns a literal string that must begin any match
// of the regular expression re. It returns the boolean true if the
// literal string comprises the entire regular expression.
func (re *Regexp) LiteralPrefix() (prefix string, complete bool) {
	return re.prefix, re.complete
}

func quote(s string) string {
	if strconv.CanBackquote(s) {
		return "`" + s + "`"
	}
	return strconv.Quote(s)
}

// MustCompile is like Compile but panics if the expression cannot be parsed.
// It simplifies safe initialization of global variables holding compiled regular
// expressions.
func MustCompile(str string) *Regexp {
	regexp, error := Compile(str)
	if error != nil {
		panic(`regexp: Compile(` + quote(str) + `): ` + error.Error())
	}
	return regexp
}

// MustCompilePOSIX is like CompilePOSIX but panics if the expression cannot be parsed.
// It simplifies safe initialization of global variables holding compiled regular
// expressions.
func MustCompilePOSIX(str string) *Regexp {
	regexp, error := CompilePOSIX(str)
	if error != nil {
		panic(`regexp: CompilePOSIX(` + quote(str) + `): ` + error.Error())
	}
	return regexp
}

var specialBytes = []byte(`\.+*?()|[]{}^$`)

func special(b byte) bool {
	return bytes.IndexByte(specialBytes, b) >= 0
}

// QuoteMeta returns a string that quotes all regular expression metacharacters
// inside the argument text; the returned string is a regular expression matching
// the literal text. For example, QuoteMeta(`[foo]`) returns `\[foo\]`.
func QuoteMeta(s string) string {
	// A byte loop is correct because all metacharacters are ASCII.
	var i int
	for i = 0; i < len(s); i++ {
		if special(s[i]) {
			break
		}
	}
	// No meta characters found, so return original string.
	if i >= len(s) {
		return s
	}

	b := make([]byte, 2*len(s)-i)
	copy(b, s[:i])
	j := i
	for ; i < len(s); i++ {
		if special(s[i]) {
			b[j] = '\\'
			j++
		}
		b[j] = s[i]
		j++
	}
	return string(b[:j])
}
