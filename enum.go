// Copyright 2017 The Regexp Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package regexp

type opcode int

const (
	opAccept opcode = iota
	opAssert
	opChar
	opCharClass
	opDot
	opDotNL
	opNotCharClass
	opNop
	opSave
	opSplit
)
