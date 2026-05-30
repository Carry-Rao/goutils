package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	fset := token.NewFileSet()
	err := filepath.Walk(".", func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.Contains(p, "vendor") || strings.Contains(p, ".git") || strings.Contains(p, ".github") {
			return nil
		}

		f, err := parser.ParseFile(fset, p, nil, 0)
		if err != nil {
			return fmt.Errorf("%s parse error", p)
		}

		ast.Inspect(f, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok {
				return true
			}
			l := fset.Position(fn.End()).Line - fset.Position(fn.Pos()).Line + 1
			pos := fset.Position(fn.Pos())
			if l <= 3 {
				fmt.Fprintf(os.Stderr, "WARNING: %s:%d function %s too short (%d lines)\n", pos.Filename, pos.Line, fn.Name.Name, l)
			}
			if l > 50 {
				fmt.Fprintf(os.Stderr, "WARNING: %s:%d function %s too long (%d lines)\n", pos.Filename, pos.Line, fn.Name.Name, l)
			}
			return true
		})

		ast.Inspect(f, func(n ast.Node) bool {
			a, ok := n.(*ast.AssignStmt)
			if !ok {
				return true
			}
			for i, lh := range a.Lhs {
				id, ok := lh.(*ast.Ident)
				if !ok || id.Name != "_" || i >= len(a.Rhs) {
					continue
				}
				c, ok := a.Rhs[i].(*ast.CallExpr)
				if !ok {
					continue
				}
				s, ok := c.Fun.(*ast.SelectorExpr)
				if !ok {
					continue
				}
				pi, ok := s.X.(*ast.Ident)
				if !ok || pi.Name != "fmt" {
					pos := fset.Position(n.Pos())
					fmt.Fprintf(os.Stderr, "WARNING: %s:%d ignored non-fmt error\n", pos.Filename, pos.Line)
				}
			}
			return true
		})
		return nil
	})

	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
}
