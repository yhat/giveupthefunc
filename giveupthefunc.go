package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
	"golang.org/x/tools/go/types"
)

var usage = `giveupthefunc [flags] [list of packages]`

func main() {
	err := doMain()
	if err != nil {
		fmt.Fprintf(os.Stderr, "giveupthefunc: %s\n", err)
		os.Exit(1)
	}
}

// Generate the list of standard packages
//go:generate /bin/bash -c ./genpkgs.sh

// List the package paths of a given type. Because types can sometimes be
// struct compositions of several types, it's possible to return more than
// one package path.
func typePackages(t types.Type) []string {
	switch t := t.(type) {
	case *types.Named:
		return []string{t.Obj().Pkg().Path()}
	case *types.Pointer:
		return typePackages(t.Elem())
	case *types.Struct:
		pkgs := []string{}
		for i := 0; i < t.NumFields(); i++ {
			pkgs = append(pkgs, typePackages(t.Field(i).Type())...)
		}
		return pkgs
	default:
		msg := fmt.Sprintf("unexpected type %s", reflect.TypeOf(t))
		panic(msg)
	}
}

// List the package paths of the type acting on a function.
func funcPackages(fn *ssa.Function) []string {
	if pkg := fn.Package(); pkg != nil {
		return []string{pkg.Object.Path()}
	}

	recv := fn.Signature.Recv()
	if recv == nil {
		if len(fn.FreeVars) == 1 {
			// it's a '$bound'
			fv := fn.FreeVars[0].Type()
			return typePackages(fv)
		} else if len(fn.Params) > 0 {
			// it's a '$thunk'
			recv := fn.Params[0].Type()
			return typePackages(recv)
		}
		panic("type had no free variables or params")
	}

	return typePackages(recv.Type())
}

// Function names sometimes show up twice. For instance
//    (*github.com/yhat/giveupthefunc/test.Foo).UsedInAnon
//    (github.com/yhat/giveupthefunc/test.Foo).UsedInAnon
// To standardize these, always remove the star.
func funcName(fn *ssa.Function) string {
	return strings.Replace(fn.RelString(nil), "*", "", 1)
}

// is the function in the standard library?
func inStandardPackages(fn *ssa.Function) bool {
	pkgs := funcPackages(fn)
	for _, pkg := range pkgs {
		_, ok := stdPkgs[pkg]
		if !ok {
			return false
		}
	}
	return true
}

var (
	includeStdPkgs bool
	usagesMatcher  *regexp.Regexp
	scopeMatcher   *regexp.Regexp
)

func doMain() error {

	var analysisScope string
	var usages string

	flag.StringVar(&usages, "usages", ".*", "a regexp to match packages to count function usages in")
	flag.StringVar(&analysisScope, "scope", ".*", "a regexp to match packages who's functions should be displayed")
	flag.BoolVar(&includeStdPkgs, "std", false, "if functions from standard packages should be included in analysis")

	flag.Parse()
	usagesMatcher = regexp.MustCompile(usages)
	scopeMatcher = regexp.MustCompile(analysisScope)

	config := &loader.Config{}

	for _, importpath := range flag.Args() {
		config.Import(importpath)
	}

	program, err := config.Load()
	if err != nil {
		return fmt.Errorf("error loading: %v", err)
	}

	prog := ssa.Create(program, 0)
	prog.BuildAll()

	calls := map[string]int{}
	// create a map of names to function values to use later
	fnNames := map[string]*ssa.Function{}

	// AllFunctions list all functions reachable by this set of programs.
	funcs := ssautil.AllFunctions(prog)
	pkgs := prog.AllPackages()
	if len(pkgs) == 0 {
		return fmt.Errorf("no packages specified")
	}

	for fn := range funcs {

		name := funcName(fn)
		fnNames[name] = fn
		if includeStdPkgs || !inStandardPackages(fn) {
			calls[name] = 0
		}
	}

	// the visitor will track function usages as it walks the ssa tree
	v := &visitor{calls, make(map[interface{}]bool)}

	for _, pkg := range pkgs {
		pkgPath := pkg.Object.Path()
		if !usagesMatcher.MatchString(pkgPath) {
			continue
		}

		// given a top level function, walk it looking for function usages
		walkFunc := func(fn *ssa.Function) {
			if fn.Pkg.Object.Path() != pkgPath {
				return
			}
			for _, block := range fn.Blocks {
				for i := range block.Instrs {
					v.walkInstr(block.Instrs[i])
				}
			}
			if fn.Recover != nil {
				for i := range fn.Recover.Instrs {
					v.walkInstr(fn.Recover.Instrs[i])
				}
			}
			for i := range fn.AnonFuncs {
				v.walkValue(fn.AnonFuncs[i])
			}
			return
		}

		for _, mem := range pkg.Members {
			switch mem := mem.(type) {
			case *ssa.Function:
				walkFunc(mem)
			case *ssa.Type:
				// if the member is a *ssa.Type walk all methods on that type
				namedType, ok := mem.Type().(*types.Named)
				if !ok {
					panic("global type is not a named type!")
				}
				for i := 0; i < namedType.NumMethods(); i++ {
					fn := prog.FuncValue(namedType.Method(i))
					walkFunc(fn)
				}
			case *ssa.Global:
			}
		}
	}
	max := 0
	for _, n := range calls {
		if n > max {
			max = n
		}
	}

	shouldPrint := func(fn *ssa.Function) bool {
		for _, pkg := range funcPackages(fn) {
			if scopeMatcher.MatchString(pkg) {
				return true
			}
		}
		return false
	}

	max = int(math.Floor(math.Log10(float64(max)))) + 1
	formatter := fmt.Sprintf("%%0%dd %%s", max)
	s := []string{}
	for name, n := range calls {
		if strings.Contains(name, "$") {
			continue
		}
		fn := fnNames[name]
		if shouldPrint(fn) {
			s = append(s, fmt.Sprintf(formatter, n, name))
		}
	}
	sort.Strings(s)
	for i := range s {
		fmt.Println(s[i])
	}
	return nil
}
