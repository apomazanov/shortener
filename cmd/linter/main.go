package main

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"
)

var panicDetector = &analysis.Analyzer{
	Name: "panicdetect",
	Doc:  "Detecting manual 'panic', 'log.Fatal' and 'os.Exit' calls",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {

			if decl, ok := n.(*ast.FuncDecl); ok {

				notMain := !(decl.Name.Name == "main" && pass.Pkg.Name() == "main")

				ast.Inspect(n, func(innerN ast.Node) bool {

					if call, ok := innerN.(*ast.CallExpr); ok {

						// detecting panic everywhere
						if fun, ok := call.Fun.(*ast.Ident); ok {
							if obj := pass.TypesInfo.Uses[fun]; obj != nil && obj.Pkg() == nil && obj.Name() == "panic" {
								pass.Reportf(call.Pos(), "pure panic call")
							}
						}

						// detecting log.Fatal and os.Exit only outside main
						if fun, ok := call.Fun.(*ast.SelectorExpr); ok {

							obj := pass.TypesInfo.Uses[fun.Sel]
							if obj != nil && obj.Pkg() != nil {
								fullName := obj.Pkg().Path() + "." + obj.Name()

								if notMain && (fullName == "log.Fatal" || fullName == "os.Exit") {
									pass.Reportf(call.Pos(), "%s is forbidden outside main", fullName)
								}
							}
						}
					}

					return true
				})

				return false // preventing double-check
			}

			return true
		})
	}
	return nil, nil
}

func main() {
	singlechecker.Main(panicDetector)
}
