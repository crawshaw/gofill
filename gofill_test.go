// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gofill

import (
	"reflect"
	"strings"
	"testing"
)

const fmtDoc = "Package fmt implements formatted I/O with functions analogous to C's printf and scanf. The format 'verbs' are derived from C's but are simpler."

const flagDoc = "Package flag implements command-line flag parsing."

type suggestTest struct {
	name string
	src  string // '‸' is removed and used as cursor position
	want []string
}

var suggestTests = []suggestTest{
	// Suggestions based on local scope.
	{
		"when valid names are in scope, don't suggest potential packages",
		`package main

		type fritters int

		func main() {
			if friedEggs, err := fn(); ok {
				var frenchToast float32
				lunch, friedBread := 5, 6
				f‸
			}
		}
		`,
		// no fmt nor flag
		[]string{"frenchToast", "friedBread", "friedEggs", "fritters"},
	},
	{
		"already imported package, don't suggest other packages",
		`package main

		import (
			"fake/go-fakepkg" // actually named fakepkg
			f2 "fake/go-fakepkg"
		)

		func main() {
			f‸
		}
		`,
		[]string{"f2", "fakepkg"}, // no fmt, flag
	},
	{
		"favor scope variable names",
		`package main

		import "fmt"

		func main() {
			if fn, err := fn(); ok {
				f‸
			}
		}
		`,
		[]string{"fmt", "fn"},
	},
	{
		"do not suggest name if already complete",
		`package main

		var f int
		func main() { f‸ }
		`,
		nil, // nothing
	},
	{
		"don't suggest package declarations",
		`package n‸

		var name string
		`,
		nil,
	},
	{
		"don't suggest at the top-level",
		`package main

		var ty string

		t‸ AType struct{}
		`,
		nil,
	},
	{
		"suggest types",
		`package main

		type AnInt int
		type AStruct struct {
			X AnI‸
		}
		`,
		[]string{"AnInt"},
	},
	{
		"switch scope",
		`package main

		func main() {
			switch value := fn().(type) {
			case 4:
				val‸
			}
		}
		`,
		[]string{"value"},
	},
	{
		"expr switch",
		`package main

		func main() {
			switch value := f(); {
			default:
				val‸
			}
		}
		`,
		[]string{"value"},
	},

	// Resilience to badly formed code
	{
		"bad package def",
		`package func main() {
			‸
		}`,
		nil,
	},

	// TODO(crawshaw): suggest types, restrict to types.

	// Suggestions based on inferring local package names.
	{
		"empty scope, few matching packages",
		`package main

		func main() { fi‸ }
		`,
		[]string{"file", "filepath"},
	},
	{
		"package in expr",
		`package main

		func main() { _, err := fi‸ }
		`,
		[]string{"file", "filepath"},
	},
	{
		"one matching inferred package",
		`package main

		func main() { fm‸ }
		`,
		[]string{"fmt"},
	},
	{
		"don't suggest package names from other package names",
		`package fm‸
		`,
		nil,
	},

	// Qualified package identifiers
	{
		"fmt package contains functions",
		`package main

		import "fmt"

		func main() { fmt.Pri‸ }
		`,
		[]string{"Print", "Printf", "Println"},
	},
	{
		"BadExpr",
		`package main

		func main() { fmt.‸ }
		`,
		[]string{"Errorf", "Formatter", "Fprint", "Fprintf", "Fprintln", "Fscan", "Fscanf", "Fscanln", "GoStringer", "Print", "Printf", "Println", "Scan", "ScanState", "Scanf", "Scanln", "Scanner", "Sprint", "Sprintf", "Sprintln", "Sscan", "Sscanf", "Sscanln", "State", "Stringer"},
	},
	{
		"toplevel keywords do not complete as packages",
		`package main

		con‸ X = ""
		`,
		nil, // not "container"
	},
	{
		"resilience in the face of unknown packages",
		`package main

		import "rubbish"

		func main() { rubbish.‸ }
		`,
		nil,
	},

	/*
		{
			"goimports inference of ambiguous package",
			`package main

			func main() {
				var x template.HTML // narrows down which template package
				template.C‸
			}`,
			[]string{"html/template"},
		},
	*/
}

func TestSuggest(t *testing.T) {
	for _, test := range suggestTests {
		offset := strings.IndexRune(test.src, '‸')
		if offset < 0 {
			t.Fatalf("%q: no '‸'", test.name)
		}
		src := test.src[:offset] + test.src[offset+len("‸"):]
		res := index.Query(src, offset)
		var got []string
		for _, s := range res.Suggest {
			got = append(got, s.Name)
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("%q:\ngot  %v\nwant %v", test.name, got, test.want)
		}
	}
}

var index *Index

func init() {
	index = SimpleIndexer()
	index.pkgs["fake/go-fakepkg"] = &pkgDecl{
		shortName: "fakepkg",
	}
}
