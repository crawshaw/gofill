// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gofill

import (
	"go/ast"
	"go/token"
	"strconv"
)

type scopeObj struct {
	name string
	kind ast.ObjKind
	pkg *pkgDecl
	// TODO declPos int
}

// scope builds a map of in-scope names at the end of the given path.
// E.g. var Name int will add the key "Name" to the returned map.
func scope(pkgs map[string]*pkgDecl, path []ast.Node) map[string]scopeObj {
	result := make(map[string]scopeObj)

	add := func(obj scopeObj) {
		if _, ok := result[obj.name]; !ok {
			result[obj.name] = obj
		}
	}

	addAssign := func(s *ast.AssignStmt) {
		if s.Tok != token.DEFINE {
			return
		}
		for _, lhs := range s.Lhs {
			if ident, ok := lhs.(*ast.Ident); ok {
				add(scopeObj{ident.Name, ast.Var, nil})
			}
		}
	}

	addGenDecl := func(decl *ast.GenDecl) {
		for _, spec := range decl.Specs {
			switch spec := spec.(type) {
			case *ast.ImportSpec:
				path, err := strconv.Unquote(spec.Path.Value)
				if err != nil {
					continue
				}
				pkg := pkgs[path]
				if pkg == nil {
					continue
				}
				name := pkg.shortName
				if spec.Name != nil {
					name = spec.Name.Name
				}
				add(scopeObj{name, ast.Pkg, pkg})
			case *ast.ValueSpec:
				for _, ident := range spec.Names {
					add(scopeObj{ident.Name, ast.Var, nil})
				}
			case *ast.TypeSpec:
				add(scopeObj{spec.Name.Name, ast.Typ, nil})
			}
		}
	}

	// Walk up, building the scope inside-out.
	for i, n := range path[1:] {
		switch n := n.(type) {
		case *ast.BlockStmt:
			for j := len(n.List) - 1; j >= 0; j-- {
				if n.List[j] == path[i-1] {
					// stop when we reach the next step in the path
					break
				}
				switch s := n.List[j].(type) {
				case *ast.DeclStmt:
					addGenDecl(s.Decl.(*ast.GenDecl))

				case *ast.AssignStmt:
					addAssign(s)
				}
			}
		case *ast.IfStmt:
			if s, ok := n.Init.(*ast.AssignStmt); ok {
				addAssign(s)
			}
		case *ast.TypeSwitchStmt:
			if s, ok := n.Assign.(*ast.AssignStmt); ok {
				addAssign(s)
			}
		case *ast.SwitchStmt:
			if s, ok := n.Init.(*ast.AssignStmt); ok {
				addAssign(s)
			}
			/* TODO
			case *ast.CaseClause:
			case *ast.Stmt:
				// for statement
				// switch statement
			case *ast.CommClause:
				// ?
			*/
		case *ast.File:
			for _, d := range n.Decls {
				if genDecl, ok := d.(*ast.GenDecl); ok {
					addGenDecl(genDecl)
				}
			}
		default:
			// shrug
		}
	}

	return result
}
