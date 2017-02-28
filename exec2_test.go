// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the GO-LICENSE file.

// +build !race

package regexp

import (
	"path/filepath"
	"runtime"
	"testing"
)

// This test is excluded when running under the race detector because
// it is a very expensive test and takes too long.
func TestRE2Exhaustive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestRE2Exhaustive during short test")
	}
	testRE2(t, filepath.Join(runtime.GOROOT(), filepath.FromSlash("src/regexp/testdata/re2-exhaustive.txt.bz2")))
}
