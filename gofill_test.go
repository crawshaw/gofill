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

	// want
	passive []string
	active  []string // passive + these
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
		nil,
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
		nil,
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
		nil,
	},
	{
		"do not suggest name if already complete",
		`package main

		var f int
		func main() { f‸ }
		`,
		nil,
		nil,
	},
	{
		"don't suggest package declarations",
		`package n‸

		var name string
		`,
		nil,
		nil,
	},
	{
		"don't suggest at the top-level",
		`package main

		var ty string

		t‸ AType struct{}
		`,
		nil,
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
		nil,
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
		nil,
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
		nil,
	},

	// Resilience to badly formed code
	{
		"bad package def",
		`package func main() {
			‸
		}`,
		nil,
		nil,
	},

	// TODO(crawshaw): suggest types, restrict to types.

	// Suggestions based on inferring local package names.
	{
		"empty scope, multiple matching packages, avoid large repo overload",
		`package main

		func main() { fi‸ }
		`,
		nil,
		[]string{"file", "filepath"},
	},
	{
		"package in expr",
		`package main

		func main() { _, err := fi‸ }
		`,
		nil,
		[]string{"file", "filepath"},
	},
	{
		"one matching inferred package, suggest",
		`package main

		func main() { fm‸ }
		`,
		nil,
		[]string{"fmt"},
	},
	{
		"don't suggest package names from other package names",
		`package fm‸
		`,
		nil,
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
		nil,
	},
	{
		"BadExpr",
		`package main

		func main() { fmt.‸ }
		`,
		[]string{"Errorf", "Formatter", "Fprint", "Fprintf", "Fprintln", "Fscan", "Fscanf", "Fscanln", "GoStringer", "Print", "Printf", "Println", "Scan", "ScanState", "Scanf", "Scanln", "Scanner", "Sprint", "Sprintf", "Sprintln", "Sscan", "Sscanf", "Sscanln", "State", "Stringer"},
		nil,
	},
	{
		"toplevel keywords do not complete as packages",
		`package main

		import "container"

		con‸ X = ""
		`,
		nil, // not "container"
		nil,
	},
	{
		"resilience in the face of unknown packages",
		`package main

		import "rubbish"

		func main() { rubbish.‸ }
		`,
		nil,
		nil,
	},

	/*
		{
			"goimports inference of ambiguous package",
			`package main

			func main() {
				var x template.HTML // narrows down to "html/template"
				template.C‸
			}`,
			[]string{"CSS"},
			nil,
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
		if !reflect.DeepEqual(got, test.passive) {
			t.Errorf("%q passive:\ngot  %v\nwant %v", test.name, got, test.passive)
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
