package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/ssa"
)

var (
	gopath    string
	goroot    string
	pkgRegexp *regexp.Regexp
)

func init() {
	gopath = os.Getenv("GOPATH")
	if gopath == "" {
		fmt.Fprintf(os.Stderr, "ERROR: GOPATH environment not found\n")
		os.Exit(2)
	}
	goroot = os.Getenv("GOROOT")
	if goroot == "" {
		fmt.Fprintf(os.Stderr, "ERROR: GOROOT environment not found\n")
		os.Exit(2)
	}
	var re string
	flag.StringVar(&re, "p", `\.`, "package name regexp to include in analysis")
	flag.Parse()
	pkgRegexp = regexp.MustCompile(re)
}

func main() {
	if err := doMain(); err != nil {
		fmt.Fprintf(os.Stderr, "gofunk: %s\n", err)
		os.Exit(1)
	}
}

type Visitor interface {
	VisitInstr(ssa.Instruction) Visitor
	VisitValue(ssa.Value) Visitor
}

type CallCounter struct {
	calls map[string]int
}

func (c *CallCounter) VisitInstr(ins ssa.Instruction) Visitor {
	return c
}

func (c *CallCounter) VisitValue(val ssa.Value) Visitor {
	switch x := val.(type) {
	case *ssa.Function:
		rel := x.RelString(nil)
		if strings.Contains(rel, "$") {
			return c
		} else {
			c.calls[rel]++
			return nil
		}
	}
	return c

}

func doMain() error {
	config := &loader.Config{
		SourceImports: true,
	}
	if _, err := config.FromArgs(flag.Args(), false); err != nil {
		return fmt.Errorf("cloud not get package from args: %s", flag.Args())
	}
	program, err := config.Load()
	if err != nil {
		return fmt.Errorf("error loading: %v", err)
	}
	prog := ssa.Create(program, 0)
	prog.BuildAll()
	pkgs := prog.AllPackages()
	calls := map[string]int{}
	v := &CallCounter{calls}
	for _, pkg := range pkgs {
		pkgPath := pkg.Object.Path()
		if !pkgRegexp.MatchString(pkgPath) {
			continue
		}
		fmt.Printf("ANALYZING %s\n", pkgPath)
		for _, member := range pkg.Members {
			m, ok := member.(*ssa.Function)
			if !ok {
				continue
			}
			rel := m.RelString(nil)
			if _, ok := calls[rel]; !ok {
				calls[rel] = 0
			}
			for _, block := range m.Blocks {
				for i := range block.Instrs {
					WalkInstr(v, block.Instrs[i])
				}
			}
			if m.Recover != nil {
				for i := range m.Recover.Instrs {
					WalkInstr(v, m.Recover.Instrs[i])
				}
			}
			for i := range m.AnonFuncs {
				WalkValue(v, m.AnonFuncs[i])
			}
		}
	}
	fmt.Println("USAGE:")
	max := 0
	for _, n := range calls {
		if n > max {
			max = n
		}
	}
	max = int(math.Floor(math.Log10(float64(max)))) + 1
	formatter := fmt.Sprintf("%%0%dd %%s", max)
	s := []string{}
	for name, n := range calls {
		s = append(s, fmt.Sprintf(formatter, n, name))
	}
	sort.Strings(s)
	for i := range s {
		fmt.Println(s[i])
	}
	return nil
}

var phiVisited = []*ssa.Phi{}

func WalkInstr(v Visitor, ins ssa.Instruction) {
	if ins == nil || v == nil {
		return
	}
	v = v.VisitInstr(ins)
	if v == nil {
		return
	}
	switch x := ins.(type) {
	case *ssa.BinOp:
		WalkValue(v, x.X)
		WalkValue(v, x.Y)
	case *ssa.Call:
		common := x.Common()
		if common == nil {
			return
		}
		WalkValue(v, common.Value)
		for i := range common.Args {
			WalkValue(v, common.Args[i])
		}
	case *ssa.ChangeInterface:
		WalkValue(v, x.X)
	case *ssa.ChangeType:
		WalkValue(v, x.X)
	case *ssa.Convert:
		WalkValue(v, x.X)
	case *ssa.DebugRef:
		WalkValue(v, x.X)
	case *ssa.Extract:
		WalkValue(v, x.Tuple)
	case *ssa.Field:
		WalkValue(v, x.X)
	case *ssa.FieldAddr:
		WalkValue(v, x.X)
	case *ssa.If:
		WalkValue(v, x.Cond)
	case *ssa.Index:
		WalkValue(v, x.X)
		WalkValue(v, x.Index)
	case *ssa.IndexAddr:
		WalkValue(v, x.X)
		WalkValue(v, x.Index)
	case *ssa.Lookup:
		WalkValue(v, x.X)
		WalkValue(v, x.Index)
	case *ssa.MakeChan:
		WalkValue(v, x.Size)
	case *ssa.MakeClosure:
		WalkValue(v, x.Fn)
		for i := range x.Bindings {
			WalkValue(v, x.Bindings[i])
		}
	case *ssa.MakeInterface:
		WalkValue(v, x.X)
	case *ssa.MakeMap:
		WalkValue(v, x.Reserve)
	case *ssa.MakeSlice:
		WalkValue(v, x.Len)
		WalkValue(v, x.Cap)
	case *ssa.MapUpdate:
		WalkValue(v, x.Map)
		WalkValue(v, x.Key)
		WalkValue(v, x.Value)
	case *ssa.Next:
		WalkValue(v, x.Iter)
	case *ssa.Panic:
		WalkValue(v, x.X)
	case *ssa.Phi:
		for _, addr := range phiVisited {
			if addr == x {
				return
			}
		}
		phiVisited = append(phiVisited, x)
		for _, edge := range x.Edges {
			WalkValue(v, edge)
		}
	case *ssa.Range:
		WalkValue(v, x.X)
	case *ssa.Return:
		for i := range x.Results {
			WalkValue(v, x.Results[i])
		}
	case *ssa.Select:
		for _, state := range x.States {
			WalkValue(v, state.Chan)
			WalkValue(v, state.Send)
		}
	case *ssa.Send:
		WalkValue(v, x.Chan)
		WalkValue(v, x.X)
	case *ssa.Slice:
		for _, val := range []ssa.Value{x.X, x.Low, x.High, x.Max} {
			WalkValue(v, val)
		}
	case *ssa.Store:
		WalkValue(v, x.Addr)
		WalkValue(v, x.Val)
	case *ssa.TypeAssert:
		WalkValue(v, x.X)
	case *ssa.UnOp:
		WalkValue(v, x.X)
	}
}

func WalkValue(v Visitor, val ssa.Value) {
	if val == nil || v == nil {
		return
	}
	v = v.VisitValue(val)
	if v == nil {
		return
	}
	switch x := val.(type) {
	case *ssa.BinOp:
		WalkValue(v, x.X)
		WalkValue(v, x.Y)
	case *ssa.Call:
		common := x.Common()
		if common == nil {
			return
		}
		WalkValue(v, common.Value)
		for i := range common.Args {
			WalkValue(v, common.Args[i])
		}
	case *ssa.ChangeInterface:
		WalkValue(v, x.X)
	case *ssa.ChangeType:
		WalkValue(v, x.X)
	case *ssa.Convert:
		WalkValue(v, x.X)
	case *ssa.Extract:
		WalkValue(v, x.Tuple)
	case *ssa.Field:
		WalkValue(v, x.X)
	case *ssa.FieldAddr:
		WalkValue(v, x.X)
	case *ssa.Function:
		for i := range x.Params {
			WalkValue(v, x.Params[i])
		}
		for i := range x.FreeVars {
			WalkValue(v, x.FreeVars[i])
		}
		for i := range x.Locals {
			WalkValue(v, x.Locals[i])
		}
		for _, block := range x.Blocks {
			for i := range block.Instrs {
				WalkInstr(v, block.Instrs[i])
			}
		}
		if x.Recover != nil {
			for i := range x.Recover.Instrs {
				WalkInstr(v, x.Recover.Instrs[i])
			}
		}
		for i := range x.AnonFuncs {
			WalkValue(v, x.AnonFuncs[i])
		}
	case *ssa.Index:
		WalkValue(v, x.X)
		WalkValue(v, x.Index)
	case *ssa.IndexAddr:
		WalkValue(v, x.X)
		WalkValue(v, x.Index)
	case *ssa.Lookup:
		WalkValue(v, x.X)
		WalkValue(v, x.Index)
	case *ssa.MakeChan:
		WalkValue(v, x.Size)
	case *ssa.MakeClosure:
		WalkValue(v, x.Fn)
		for i := range x.Bindings {
			WalkValue(v, x.Bindings[i])
		}
	case *ssa.MakeInterface:
		WalkValue(v, x.X)
	case *ssa.MakeMap:
		WalkValue(v, x.Reserve)
	case *ssa.MakeSlice:
		WalkValue(v, x.Len)
		WalkValue(v, x.Cap)
	case *ssa.Next:
		WalkValue(v, x.Iter)
	case *ssa.Phi:
		for _, addr := range phiVisited {
			if addr == x {
				return
			}
		}
		phiVisited = append(phiVisited, x)
		for _, edge := range x.Edges {
			WalkValue(v, edge)
		}
	case *ssa.Range:
		WalkValue(v, x.X)
	case *ssa.Select:
		for _, state := range x.States {
			WalkValue(v, state.Chan)
			WalkValue(v, state.Send)
		}
	case *ssa.Slice:
		for _, val := range []ssa.Value{x.X, x.Low, x.High, x.Max} {
			WalkValue(v, val)
		}
	case *ssa.TypeAssert:
		WalkValue(v, x.X)
	case *ssa.UnOp:
		WalkValue(v, x.X)
	}
}
