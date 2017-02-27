// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the GO-LICENSE file.

// +build !race

package regexp

//TODO import (
//TODO 	"testing"
//TODO )
//TODO
//TODO // This test is excluded when running under the race detector because
//TODO // it is a very expensive test and takes too long.
//TODO func TestRE2Exhaustive(t *testing.T) {
//TODO 	if testing.Short() {
//TODO 		t.Skip("skipping TestRE2Exhaustive during short test")
//TODO 	}
//TODO 	testRE2(t, "testdata/re2-exhaustive.txt.bz2")
//TODO }
