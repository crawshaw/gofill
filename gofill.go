// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package gofill completes incomplete Go.
package gofill

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"code.google.com/p/go.tools/astutil"
)

var Default = &Index{}

type Index struct {
	pkgNames map[string]map[string]bool // "template" -> {"html/template", "text/template"}
	pkgs     map[string]*pkgDecl        // "text/template" -> ...
}

func (x *Index) scopeSearch(query *queryState, n *ast.Ident) {
	// Start by searching the scope.
	for name := range query.scope {
		if strings.HasPrefix(name, n.Name) {
			if len(name) == len(n.Name) {
				// Do not make suggestions if they have something complete.
				query.res.Suggest = nil
				return
			}
			query.res.Suggest = append(query.res.Suggest, Suggestion{
				Range: query.pos,
				Name:  name,
			})
		}
	}

	if len(query.res.Suggest) > 0 {
		return
	}

	// If nothing in the scope matches, speculate
	// about potential packages.
	// TODO(crawshaw): suffixarray
	for name := range x.pkgNames {
		if strings.HasPrefix(name, n.Name) {
			if len(name) == len(n.Name) {
				query.res.Suggest = nil
				break
			}
			query.res.Suggest = append(query.res.Suggest, Suggestion{
				Range: query.pos,
				Name:  name,
			})

		}
	}
}

func (x *Index) selectorSearch(query *queryState, primary, secondary string) {
	// Qualified identifier (package name primary, idenitifier secondary).
	obj, ok := query.scope[primary]
	if !ok {
		// TODO
		pkgs := x.pkgNames[primary]
		if len(pkgs) == 1 {
			// For now we do not offer any suggestions if
			// we are unsure what the package is.
			//
			// TODO(crawshaw): apply the goimports logic to
			// the file before doing this, as a prior template.HTML
			// is enough information to safely guess which template
			// package you want.
			for name := range pkgs {
				x.pkgSearch(query, x.pkgs[name], secondary)
			}
		}
		// Attempt speculative package match.
		// x.pkgSearch(query, pkg, secondary)
		return
	}
	switch obj.kind {
	case ast.Pkg:
		x.pkgSearch(query, obj.pkg, secondary)
	case ast.Var:
		// TODO: Method expression

		// TODO: Field selector
	default:
		fmt.Printf("%q is unexpected: %s\n", primary, query.scope[primary])
	}
}

func (x *Index) pkgSearch(query *queryState, pkg *pkgDecl, name string) {
	for _, decl := range pkg.decls {
		if !ast.IsExported(decl.name) {
			continue
		}
		if strings.HasPrefix(decl.name, name) {
			if len(name) == len(decl.name) {
				query.res.Suggest = nil
				return
			}
			query.res.Suggest = append(query.res.Suggest, Suggestion{
				Range: query.pos,
				Name:  decl.name,
				Doc:   decl.doc,
			})
		}
	}
}

type queryState struct {
	f     *ast.File
	path  []ast.Node
	scope map[string]scopeObj
	pos   Range
	res   Result
}

func (x *Index) Query(src string, offset int) Result {
	// We begin with a deeply offensive hack.
	// When faced with a syntactically correct selector,
	// e.g. fmt.P, the parser generates:
	//
	// 	*ast.ExprStmt {
	//	.  X: *ast.SelectorExpr {
	//	.  .  X: *ast.Ident {
	// 	.  .  .  NamePos: file.go:4:1
	// 	.  .  .  Name: "fmt"
	// 	.  .  }
	// 	.  .  Sel: *ast.Ident {
	// 	.  .  .  NamePos: file.go:4:5
	// 	.  .  .  Name: "P"
	// 	.  .  }
	// 	.  }
	// 	}
	//
	// If however, the selector identifier is missing,
	// we get a very BadExpr:
	//
	//	*ast.ExprStmt {
	//	.  X: *ast.BadExpr {
	//	.  .  From: file.go:5:1
	// 	.  .  To: file.go:5:2
	//	.  }
	//	}
	//
	// This is hard to deal with in the AST, particularly
	// as standard tools like ast.Inspect and ast.Walk
	// don't play well with BadExpr.
	//
	// So we cheat. If the caret offset is placed directly
	// after a selector, and the following rune is not a
	// valid initial identifier rune, we insert one.
	var fakeIdentifier bool
	if offset > 0 {
		sel, _ := utf8.DecodeLastRuneInString(src[:offset])
		identStart, _ := utf8.DecodeRuneInString(src[offset:])
		fmt.Printf("sel=%c, identStart=%c\n", sel, identStart)
		if sel == '.' && !unicode.IsLetter(identStart) {
			src = src[:offset] + "X" + src[offset:]
			fakeIdentifier = true
		}
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "file.go", src, parser.ParseComments|parser.AllErrors)
	ast.Print(fset, f)

	pos := f.Package + token.Pos(offset)
	end := pos
	// Step back one when searching for the enclosing expression as
	// we typically complete partially typed words.
	if offset > 0 {
		pos--
	}
	if offset < len(src)-1 {
		end++
	}
	path, _ := astutil.PathEnclosingInterval(f, pos, end)
	fmt.Printf("PathEnclosingInterval(%d, %d): %#+v\n", pos, end, path)

	query := &queryState{
		f:     f,
		path:  path,
		scope: scope(x.pkgs, path),
		pos:   Range{}, // TODO path[0] Pos
		res:   Result{},
	}

	if err != nil {
		query.res.Error = append(query.res.Error, Error{
			Range: Range{-1, -1}, // TODO
			Error: err.Error(),   // TODO split out all the errors
		})
	}

	if len(path) <= 2 {
		// We do nothing useful at the top level yet (maybe never).
		//
		// In particular, ignore top-level Idents. In the top-level,
		// suggesting from the scope doesn't make sense.
		return query.res
	}

	if n, ok := path[1].(*ast.SelectorExpr); ok {
		// TODO(crawshaw): do something sensible with `x.y.‸`
		var primary, secondary string
		if n, ok := n.X.(*ast.Ident); ok {
			primary = n.Name
		}
		if !fakeIdentifier {
			secondary = n.Sel.Name
		}
		x.selectorSearch(query, primary, secondary)
	} else if n, ok := path[0].(*ast.Ident); ok {
		x.scopeSearch(query, n)
	}

	sort.Sort(byName(query.res.Suggest))
	return query.res
}

type byName []Suggestion

func (s byName) Len() int           { return len(s) }
func (s byName) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s byName) Less(i, j int) bool { return s[i].Name < s[j].Name }

type Range struct {
	Pos int // relative to offset
	End int // relative to offset
}

type Error struct {
	Range Range
	Error string
}

type Suggestion struct {
	Range Range
	Name  string
	// TODO Kind  string
	Doc string `json:",omitempty"`
}

type Result struct {
	Suggest []Suggestion `json:",omitempty"`
	Related []Range      `json:",omitempty"`
	Error   []Error      `json:",omitempty"`
}

type pkgDecl struct {
	shortName string
	decls     []*decl // TODO(crawshaw): suffixarray?
}

type decl struct {
	name string
	typ  string // TODO
	doc  string
}
