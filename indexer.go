// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gofill

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sync"
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

var fset = token.NewFileSet()

type simpleIndexer struct {
	sync.Mutex
	m *Indexer
}

func SimpleIndexer() *Index {
	p := new(simpleIndexer)

	p.Lock()
	p.m = new(Indexer)
	p.Unlock()

	path := filepath.Join(build.Default.GOROOT, "src", "pkg")
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		return nil
	}
	children, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		return nil
	}

	var wg sync.WaitGroup
	for _, child := range children {
		if child.IsDir() {
			wg.Add(1)
			go func(path, name string) {
				defer wg.Done()
				p.loadPkg(&wg, path, name)
			}(path, child.Name())
		}
	}
	wg.Wait()

	p.Lock()
	defer p.Unlock()
	return p.m.Index()
}

func (p *simpleIndexer) loadPkg(wg *sync.WaitGroup, root, pkgrelpath string) {
	importPath := filepath.ToSlash(pkgrelpath)

	buildPkg, err := build.Import(importPath, "", 0)
	if err == nil {
		for _, fileName := range buildPkg.GoFiles {
			path := filepath.Join(root, importPath, fileName)
			f, err := parser.ParseFile(fset, path, nil, parser.ParseComments|parser.AllErrors)
			if err != nil {
				continue
			}
			p.Lock()
			p.m.AddFile(importPath, f)
			p.Unlock()
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
				p.loadPkg(wg, root, name)
			}(root, filepath.Join(importPath, name))
		}
	}
}
