package main

import (
	"strings"

	"golang.org/x/tools/go/ssa"
)

// a visitor walks the ssa tree looking for function usages.
type visitor struct {
	calls map[string]int

	// a map of ssa.Instructions and ssa.Values which have
	// already been visited
	visited map[interface{}]bool
}

func (v *visitor) VisitInstr(ins ssa.Instruction) *visitor {
	return v
}

func (v *visitor) VisitValue(val ssa.Value) *visitor {
	if fn, ok := val.(*ssa.Function); ok {

		if !includeStdPkgs && inStandardPackages(fn) {
			return nil
		}
		rel := funcName(fn)

		// '$' indicates special functions such as 'main()'
		if strings.Contains(rel, "$") {
			return v

		}
		// ssa.allFunctions should list all possible functions within the
		// scope of these programs. If we see a function it didn't list,
		// there's a problem.
		if _, ok := v.calls[rel]; !ok {
			panic("unexpected function visited " + rel)
		}
		v.calls[rel]++
		return nil
	}
	return v
}

// exhaustive walk function.
// TODO: automatically generate these
func (v *visitor) walkInstr(ins ssa.Instruction) {
	if ins == nil || v == nil || v.visited[ins] {
		return
	}
	v = v.VisitInstr(ins)
	v.visited[ins] = true
	if v == nil {
		return
	}
	switch x := ins.(type) {
	case *ssa.BinOp:
		v.walkValue(x.X)
		v.walkValue(x.Y)
	case *ssa.Call:
		common := x.Common()
		if common == nil {
			return
		}
		v.walkValue(common.Value)
		for i := range common.Args {
			v.walkValue(common.Args[i])
		}
	case *ssa.ChangeInterface:
		v.walkValue(x.X)
	case *ssa.ChangeType:
		v.walkValue(x.X)
	case *ssa.Convert:
		v.walkValue(x.X)
	case *ssa.DebugRef:
		v.walkValue(x.X)
	case *ssa.Defer:
		v.walkValue(x.Call.Value)
	case *ssa.Extract:
		v.walkValue(x.Tuple)
	case *ssa.Field:
		v.walkValue(x.X)
	case *ssa.FieldAddr:
		v.walkValue(x.X)
	case *ssa.Go:
		v.walkValue(x.Call.Value)
	case *ssa.If:
		v.walkValue(x.Cond)
	case *ssa.Index:
		v.walkValue(x.X)
		v.walkValue(x.Index)
	case *ssa.IndexAddr:
		v.walkValue(x.X)
		v.walkValue(x.Index)
	case *ssa.Lookup:
		v.walkValue(x.X)
		v.walkValue(x.Index)
	case *ssa.MakeChan:
		v.walkValue(x.Size)
	case *ssa.MakeClosure:
		v.walkValue(x.Fn)
		for i := range x.Bindings {
			v.walkValue(x.Bindings[i])
		}
	case *ssa.MakeInterface:
		v.walkValue(x.X)
	case *ssa.MakeMap:
		v.walkValue(x.Reserve)
	case *ssa.MakeSlice:
		v.walkValue(x.Len)
		v.walkValue(x.Cap)
	case *ssa.MapUpdate:
		v.walkValue(x.Map)
		v.walkValue(x.Key)
		v.walkValue(x.Value)
	case *ssa.Next:
		v.walkValue(x.Iter)
	case *ssa.Panic:
		v.walkValue(x.X)
	case *ssa.Phi:
		for _, edge := range x.Edges {
			v.walkValue(edge)
		}
	case *ssa.Range:
		v.walkValue(x.X)
	case *ssa.Return:
		for i := range x.Results {
			v.walkValue(x.Results[i])
		}
	case *ssa.Select:
		for _, state := range x.States {
			v.walkValue(state.Chan)
			v.walkValue(state.Send)
		}
	case *ssa.Send:
		v.walkValue(x.Chan)
		v.walkValue(x.X)
	case *ssa.Slice:
		for _, val := range []ssa.Value{x.X, x.Low, x.High, x.Max} {
			v.walkValue(val)
		}
	case *ssa.Store:
		v.walkValue(x.Addr)
		v.walkValue(x.Val)
	case *ssa.TypeAssert:
		v.walkValue(x.X)
	case *ssa.UnOp:
		v.walkValue(x.X)
	}
}

func (v *visitor) walkValue(val ssa.Value) {
	if val == nil || v == nil || v.visited[val] {
		return
	}
	v = v.VisitValue(val)
	if v == nil {
		return
	}
	v.visited[val] = true
	switch x := val.(type) {
	case *ssa.BinOp:
		v.walkValue(x.X)
		v.walkValue(x.Y)
	case *ssa.Call:
		common := x.Common()
		if common == nil {
			return
		}
		v.walkValue(common.Value)
		for i := range common.Args {
			v.walkValue(common.Args[i])
		}
	case *ssa.ChangeInterface:
		v.walkValue(x.X)
	case *ssa.ChangeType:
		v.walkValue(x.X)
	case *ssa.Convert:
		v.walkValue(x.X)
	case *ssa.Extract:
		v.walkValue(x.Tuple)
	case *ssa.Field:
		v.walkValue(x.X)
	case *ssa.FieldAddr:
		v.walkValue(x.X)
	case *ssa.Function:
		for i := range x.Params {
			v.walkValue(x.Params[i])
		}
		for i := range x.FreeVars {
			v.walkValue(x.FreeVars[i])
		}
		for i := range x.Locals {
			v.walkValue(x.Locals[i])
		}
		for _, block := range x.Blocks {
			for i := range block.Instrs {
				v.walkInstr(block.Instrs[i])
			}
		}
		if x.Recover != nil {
			for i := range x.Recover.Instrs {
				v.walkInstr(x.Recover.Instrs[i])
			}
		}
		for i := range x.AnonFuncs {
			v.walkValue(x.AnonFuncs[i])
		}
	case *ssa.Index:
		v.walkValue(x.X)
		v.walkValue(x.Index)
	case *ssa.IndexAddr:
		v.walkValue(x.X)
		v.walkValue(x.Index)
	case *ssa.Lookup:
		v.walkValue(x.X)
		v.walkValue(x.Index)
	case *ssa.MakeChan:
		v.walkValue(x.Size)
	case *ssa.MakeClosure:
		v.walkValue(x.Fn)
		for i := range x.Bindings {
			v.walkValue(x.Bindings[i])
		}
	case *ssa.MakeInterface:
		v.walkValue(x.X)
	case *ssa.MakeMap:
		v.walkValue(x.Reserve)
	case *ssa.MakeSlice:
		v.walkValue(x.Len)
		v.walkValue(x.Cap)
	case *ssa.Next:
		v.walkValue(x.Iter)
	case *ssa.Phi:
		for _, edge := range x.Edges {
			v.walkValue(edge)
		}
	case *ssa.Range:
		v.walkValue(x.X)
	case *ssa.Select:
		for _, state := range x.States {
			v.walkValue(state.Chan)
			v.walkValue(state.Send)
		}
	case *ssa.Slice:
		for _, val := range []ssa.Value{x.X, x.Low, x.High, x.Max} {
			v.walkValue(val)
		}
	case *ssa.TypeAssert:
		v.walkValue(x.X)
	case *ssa.UnOp:
		v.walkValue(x.X)
	}
}
