// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"unicode/utf8"
)

func main() {
	if err := bake("static/*"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func bake(glob string) error {
	files, err := filepath.Glob(glob)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(os.Stdout)
	fmt.Fprintf(w, "%v\n\npackage gofill\n\n", warning)
	fmt.Fprintf(w, "var StaticFiles = map[string]string{\n")
	for _, fn := range files {
		b, err := ioutil.ReadFile(fn)
		if err != nil {
			return err
		}
		if !utf8.Valid(b) {
			return fmt.Errorf("file %s is not valid UTF-8", fn)
		}
		fmt.Fprintf(w, "\t%q: `%s`,\n", filepath.Base(fn), sanitize(b))
	}
	fmt.Fprintln(w, "}")
	return w.Flush()
}

// sanitize prepares a string as a raw string constant.
func sanitize(b []byte) []byte {
	// Replace ` with `+"`"+`
	b = bytes.Replace(b, []byte("`"), []byte("`+\"`\"+`"), -1)

	// Replace BOM with `+"\xEF\xBB\xBF"+`
	// (A BOM is valid UTF-8 but not permitted in Go source files.
	// I wouldn't bother handling this, but for some insane reason
	// jquery.js has a BOM somewhere in the middle.)
	return bytes.Replace(b, []byte("\xEF\xBB\xBF"), []byte("`+\"\\xEF\\xBB\\xBF\"+`"), -1)
}

const warning = "// ** This file was generated with the bake tool ** Edits will be ignored //"
