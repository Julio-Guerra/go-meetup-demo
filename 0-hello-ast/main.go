package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"os"
)

func main() {
	printGo := flag.Bool("go", false, "print back to Go")
	flag.Parse()
	file := flag.Arg(0)

	fset := token.NewFileSet()
	root, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
	if err != nil {
		log.Fatalf("%s: %v", file, err)
	}

	if *printGo {
		printer.Fprint(os.Stdout, fset, root)
	} else {
		ast.Print(fset, root)
	}

	ast.Inspect(root, func (n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			fmt.Println(funcDecl.Name.Name, "is a function")
		}
		return true
	})
}


