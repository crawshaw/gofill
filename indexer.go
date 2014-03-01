// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gofill

import (
	"go/ast"
)

type Indexer struct {
	pkgNames map[string]map[string]bool // "template" -> {"html/template", "text/template"}
	pkgs     map[string]*pkgDecl        // "text/template" -> ...
}

func (p *Indexer) Index() *Index {
	return &Index{
		pkgNames: p.pkgNames,
		pkgs:     p.pkgs,
	}
}

func (p *Indexer) AddFile(dirname string, file *ast.File) {
	if p.pkgs == nil {
		p.pkgs = make(map[string]*pkgDecl)
	}
	if p.pkgNames == nil {
		p.pkgNames = make(map[string]map[string]bool)
	}

	pkgName := file.Name.Name
	pkgMap := p.pkgNames[pkgName]
	if pkgMap == nil {
		pkgMap = make(map[string]bool)
		p.pkgNames[pkgName] = pkgMap
	}
	pkgMap[dirname] = true
	pkg := p.pkgs[dirname]
	if pkg == nil {
		pkg = &pkgDecl{shortName: pkgName}
	}
	p.pkgs[dirname] = pkg

	for _, d := range file.Decls {
		switch d := d.(type) {
		case *ast.GenDecl:
			if len(d.Specs) == 0 {
				continue
			}
			switch s := d.Specs[0].(type) {
			case *ast.ValueSpec:
				for _, n := range s.Names {
					pkg.decls = append(pkg.decls, &decl{
						name: n.Name,
						doc:  s.Comment.Text(),
					})
				}
			case *ast.TypeSpec:
				pkg.decls = append(pkg.decls, &decl{
					name: s.Name.Name,
					doc:  s.Comment.Text(),
				})
			}
		case *ast.FuncDecl:
			if d.Recv != nil {
				continue
			}
			pkg.decls = append(pkg.decls, &decl{
				name: d.Name.Name,
				doc:  d.Doc.Text(),
			})
		}
	}
}
