// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gofill

import (
	"fmt"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
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

var pkgIndexer struct {
	sync.Mutex
	m *Indexer
}

func init() {
	pkgIndexer.Lock()
	pkgIndexer.m = new(Indexer)
	pkgIndexer.Unlock()

	path := filepath.Join(build.Default.GOROOT, "src", "pkg")
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		return
	}
	children, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		return
	}

	var wg sync.WaitGroup
	for _, child := range children {
		if child.IsDir() {
			wg.Add(1)
			go func(path, name string) {
				defer wg.Done()
				loadPkg(&wg, path, name)
			}(path, child.Name())
		}
	}
	wg.Wait()

	pkgIndexer.Lock()
	index = pkgIndexer.m.Index()
	pkgIndexer.Unlock()

	index.pkgs["fake/go-fakepkg"] = &pkgDecl{
		shortName: "fakepkg",
	}
}

var fset = token.NewFileSet()

func loadPkg(wg *sync.WaitGroup, root, pkgrelpath string) {
	importPath := filepath.ToSlash(pkgrelpath)

	buildPkg, err := build.Import(importPath, "", 0)
	if err == nil {
		for _, fileName := range buildPkg.GoFiles {
			path := filepath.Join(root, importPath, fileName)
			f, err := parser.ParseFile(fset, path, nil, parser.ParseComments|parser.AllErrors)
			if err != nil {
				continue
			}
			pkgIndexer.Lock()
			pkgIndexer.m.AddFile(importPath, f)
			pkgIndexer.Unlock()
		}
	}

	dir := filepath.Join(root, importPath)
	pkgDir, err := os.Open(dir)
	if err != nil {
		return
	}
	children, err := pkgDir.Readdir(-1)
	pkgDir.Close()
	if err != nil {
		return
	}
	for _, child := range children {
		name := child.Name()
		if name == "" {
			continue
		}
		if c := name[0]; c == '.' || ('0' <= c && c <= '9') {
			continue
		}
		if child.IsDir() {
			wg.Add(1)
			go func(root, name string) {
				defer wg.Done()
				loadPkg(wg, root, name)
			}(root, filepath.Join(importPath, name))
		}
	}
}
