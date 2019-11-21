package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"log"
)

func main() {
	flag.Parse()
	file := flag.Arg(0)

	// Parse
	fileNode, fset, err := parseFile(file)
	if err != nil {
		log.Fatal(err)
	}

	// Get type information
	cfg := types.Config{Importer: importer.Default()}
	typeinfo := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Uses: make(map[*ast.Ident]types.Object),
		Defs: make(map[*ast.Ident]types.Object),
	}
	pkg, err := cfg.Check(".", fset, []*ast.File{fileNode}, typeinfo)
	if err != nil {
		log.Fatal(err)
	}

	// Lookup for net/http.HandlerFunc type
	httpHandlerFuncType := lookupPkgObj(pkg,"net/http", "HandlerFunc")
	if httpHandlerFuncType == nil {
		log.Fatal("this go file doesn't use net/http")
	}

	// List objects assignable to net/http.HandlerFunc
	for id, obj := range typeinfo.Defs {
		if obj == nil {
			continue
		}
		// Check if obj's type is a function signature
		fnSignature, ok := obj.Type().(*types.Signature)
		if !ok {
			continue
		}

		// Check if it is assignable to a net/http.HandlerFunc value type
		if types.AssignableTo(httpHandlerFuncType, obj.Type()) {
			fmt.Printf("%s: `%s` defines `%v`\n", fset.Position(id.Pos()), obj.Name(), fnSignature)
		}
	}
}

func lookupPkgObj(pkg *types.Package, pkgPath string, ident string) types.Type {
	for _, importPkg := range pkg.Imports() {
		if importPkg.Path() == pkgPath {
			if obj := importPkg.Scope().Lookup(ident); obj != nil {
				return obj.Type()
			}
		}
	}
	return nil
}

func parseFile(file string) (root *ast.File,	fset *token.FileSet, err error) {
	fset = token.NewFileSet()
	root, err = parser.ParseFile(fset, file, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}
	return root, fset, nil
}
