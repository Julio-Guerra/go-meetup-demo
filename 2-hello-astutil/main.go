package main

import (
	"flag"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"log"
	"os"
	"strconv"

	"golang.org/x/tools/go/ast/astutil"
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
		Uses: make(map[*ast.Ident]types.Object),
		Defs: make(map[*ast.Ident]types.Object),
	}
	pkg, err := cfg.Check(".", fset, []*ast.File{fileNode}, typeinfo)
	if err != nil {
		log.Fatal(err)
	}

	// Lookup for net/http.HandlerFunc type
	httpHandlerFuncType := lookupPkgObj(pkg, "net/http", "HandlerFunc")
	if httpHandlerFuncType == nil {
		log.Fatal("this go file doesn't use net/http")
	}

	// Add opentracing spans to http handlers
	var doImportOpenTracing bool // true when the import is required
	pre := func(cursor *astutil.Cursor) bool {
		// Check if it is a function declaration
		funcDecl, ok := cursor.Node().(*ast.FuncDecl)
		if !ok {
			return true
		}

		obj := typeinfo.ObjectOf(funcDecl.Name)
		t := obj.Type()
		if !types.AssignableTo(httpHandlerFuncType, t) {
			return true
		}

		// Prepend two new statements to the http handler:
		//   sp := opentracing.StartSpan(id)
		//   defer sp.Finish()
		funcDecl.Body.List = append([]ast.Stmt{
			// sp := opentracing.StartSpan(id)
			&ast.AssignStmt{
				Lhs: []ast.Expr{ast.NewIdent("sp")},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{
					&ast.CallExpr{
						Fun:  &ast.SelectorExpr{X: ast.NewIdent("opentracing"), Sel: ast.NewIdent("StartSpan")},
						Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(obj.Name())}},
					},
				},
			},
			// defer sp.Finish()
			&ast.DeferStmt{Call: &ast.CallExpr{
				Fun: &ast.SelectorExpr{X: ast.NewIdent("sp"), Sel: ast.NewIdent("Finish")},
			}},
		}, funcDecl.Body.List...)

		// Add the opentracing import
		doImportOpenTracing = true
		return true
	}
	astutil.Apply(fileNode, pre, nil)

	// Add the import if spans were added
	if doImportOpenTracing {
		astutil.AddImport(fset, fileNode, "github.com/opentracing/opentracing-go")
	}

	// Print back the file to stdout
	printer.Fprint(os.Stdout, fset, fileNode)
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

func parseFile(file string) (root *ast.File, fset *token.FileSet, err error) {
	fset = token.NewFileSet()
	root, err = parser.ParseFile(fset, file, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}
	return root, fset, nil
}
