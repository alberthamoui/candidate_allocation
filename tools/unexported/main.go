package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/types"
	"os"
	"strings"

	"golang.org/x/tools/go/packages"
)

var (
	failOnError = flag.Bool("error", false, "fail with non-zero exit code if unexported functions are found")
	allowlist   = flag.String("allowlist", "", "comma separated list of Package.Func names to ignore")
)

func main() {
	flag.Parse()

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedTypes | packages.NeedTypesInfo |
			packages.NeedSyntax,
		Tests: false,
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		fmt.Fprintf(os.Stderr, "load: %v\n", err)
		os.Exit(1)
	}
	if packages.PrintErrors(pkgs) > 0 {
		os.Exit(1)
	}

	if len(pkgs) == 0 {
		return
	}

	allowedFuncs := make(map[string]bool)
	for _, a := range strings.Split(*allowlist, ",") {
		if a != "" {
			allowedFuncs[a] = true
		}
	}

	exportedFuncs := make(map[*types.Func]bool)
	usedOutside := make(map[*types.Func]bool)
	fset := pkgs[0].Fset

	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			for _, decl := range file.Decls {
				if fd, ok := decl.(*ast.FuncDecl); ok {
					if fd.Name.IsExported() {
						obj := pkg.TypesInfo.Defs[fd.Name]
						if fn, ok := obj.(*types.Func); ok {
							exportedFuncs[fn] = true
						}
					}
				}
			}
		}
	}

	for _, pkg := range pkgs {
		for _, obj := range pkg.TypesInfo.Uses {
			if fn, ok := obj.(*types.Func); ok {
				if exportedFuncs[fn] && fn.Pkg() != pkg.Types {
					usedOutside[fn] = true
				}
			}
		}
		for _, sel := range pkg.TypesInfo.Selections {
			if fn, ok := sel.Obj().(*types.Func); ok {
				if exportedFuncs[fn] && fn.Pkg() != pkg.Types {
					usedOutside[fn] = true
				}
			}
		}
	}

	foundIssue := false
	for fn := range exportedFuncs {
		if !usedOutside[fn] {
			pkgPath := fn.Pkg().Path()
			if strings.HasSuffix(pkgPath, "/main") || pkgPath == "main" {
				continue
			}

			name := fn.Name()
			if sig, ok := fn.Type().(*types.Signature); ok && sig.Recv() != nil {
				recvStr := sig.Recv().Type().String()
				parts := strings.Split(recvStr, ".")
				if len(parts) > 0 {
					name = parts[len(parts)-1] + "." + name
				}
			}
			fullPath := pkgPath + "." + name

			if strings.HasPrefix(name, "Test") {
				continue
			}

			if allowedFuncs[fullPath] || allowedFuncs[pkgPath] || allowedFuncs[name] {
				continue
			}

			if strings.HasPrefix(name, "App.") || name == "NewApp" || name == "ProcessMapping" || name == "WailsInit" {
				continue
			}

			pos := fset.Position(fn.Pos())
			fmt.Printf("%s: exported function %s is never used outside its package, consider making it private\n", pos, name)
			foundIssue = true
		}
	}

	if foundIssue && *failOnError {
		os.Exit(1)
	}
}
