// Package apicheck lists the exported API of a package as sorted text lines,
// so that a facade package can be compared with the API it promised to keep.
//
// The listing names every exported constant (with its value), variable,
// function and type, the exported fields of structs, the methods of
// interfaces and the method set of every named type. Parameter names are
// left out: they are not part of the API. A type defined inside this module
// is printed under the name a facade package gives it, so an alias such as
// "type App = orpheus.App" in pkg/orpheus prints exactly like the type it
// replaced.
//
// It is used only by tests.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0
package apicheck

import (
	"bufio"
	"bytes"
	"fmt"
	"go/importer"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
)

// Module is the import path of this module; types under it are printed
// relative to it.
const Module = "github.com/agilira/orpheus"

// Facades are the packages that keep the v1.11.0 import paths.
var Facades = []string{
	Module + "/pkg/orpheus",
	Module + "/pkg/metrics",
	Module + "/pkg/term",
}

// Universe holds type-checked packages that share one set of type objects,
// so that an alias and its target are the same *types.TypeName.
type Universe struct {
	imp   types.Importer
	names map[*types.TypeName]string
}

// Load imports the facades, and any extra packages, from compiler export data.
func Load(extra ...string) (*Universe, error) {
	paths := append(append([]string{}, Facades...), extra...)
	args := append([]string{"list", "-export", "-deps", "-f", "{{.ImportPath}}\t{{.Export}}"}, paths...)
	// #nosec G204 -- the command is the go tool; its arguments are fixed
	// flags and the import paths the facade tests pass in, never user input.
	cmd := exec.Command("go", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list: %v: %s", err, stderr.String())
	}
	exports := make(map[string]string)
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		path, file, ok := strings.Cut(sc.Text(), "\t")
		if ok && file != "" {
			exports[path] = file
		}
	}
	lookup := func(path string) (io.ReadCloser, error) {
		file, ok := exports[path]
		if !ok {
			return nil, fmt.Errorf("no export data for %s", path)
		}
		return os.Open(file) // #nosec G304 -- path comes from go list
	}
	u := &Universe{
		imp:   importer.ForCompiler(token.NewFileSet(), "gc", lookup),
		names: make(map[*types.TypeName]string),
	}
	for _, path := range Facades {
		pkg, err := u.Package(path)
		if err != nil {
			return nil, err
		}
		u.addNames(pkg)
	}
	return u, nil
}

// Package returns the type-checked package at path.
func (u *Universe) Package(path string) (*types.Package, error) {
	return u.imp.Import(path)
}

// addNames records the facade name of every named type the package exports.
// When two names point at one type (Error and OrpheusError), the name equal
// to the type's own name wins.
func (u *Universe) addNames(pkg *types.Package) {
	rel := strings.TrimPrefix(pkg.Path(), Module+"/")
	scope := pkg.Scope()
	for _, name := range scope.Names() {
		tn, ok := scope.Lookup(name).(*types.TypeName)
		if !ok || !tn.Exported() {
			continue
		}
		named, ok := types.Unalias(tn.Type()).(*types.Named)
		if !ok {
			continue
		}
		target := named.Obj()
		if prev, seen := u.names[target]; seen && (strings.HasSuffix(prev, "."+target.Name()) || name != target.Name()) {
			continue
		}
		u.names[target] = rel + "." + name
	}
}

// API returns the sorted API listing of the package at path.
func (u *Universe) API(path string) ([]string, error) {
	pkg, err := u.Package(path)
	if err != nil {
		return nil, err
	}
	var lines []string
	scope := pkg.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if !obj.Exported() {
			continue
		}
		lines = append(lines, u.describe(obj)...)
	}
	sort.Strings(lines)
	return lines, nil
}

func (u *Universe) describe(obj types.Object) []string {
	name := obj.Name()
	switch o := obj.(type) {
	case *types.Const:
		return []string{fmt.Sprintf("const %s %s = %s", name, u.typ(o.Type()), o.Val().ExactString())}
	case *types.Var:
		return []string{fmt.Sprintf("var %s %s", name, u.typ(o.Type()))}
	case *types.Func:
		return []string{fmt.Sprintf("func %s%s", name, u.sig(o.Type().(*types.Signature)))}
	case *types.TypeName:
		return u.describeType(name, o)
	}
	return []string{fmt.Sprintf("? %s", name)}
}

func (u *Universe) describeType(name string, tn *types.TypeName) []string {
	t := types.Unalias(tn.Type())
	named, ok := t.(*types.Named)
	if !ok {
		return []string{fmt.Sprintf("type %s = %s", name, u.typ(t))}
	}
	canonical := u.names[named.Obj()]
	if canonical != "" && !strings.HasSuffix(canonical, "."+name) {
		return []string{fmt.Sprintf("type %s = %s", name, canonical)}
	}
	lines := []string{fmt.Sprintf("type %s %s", name, u.underlying(named.Underlying()))}
	ms := types.NewMethodSet(types.NewPointer(named))
	values := types.NewMethodSet(named)
	for i := 0; i < ms.Len(); i++ {
		m := ms.At(i).Obj()
		if !m.Exported() {
			continue
		}
		recv := "*" + name
		if values.Lookup(m.Pkg(), m.Name()) != nil {
			recv = name
		}
		lines = append(lines, fmt.Sprintf("method (%s) %s%s", recv, m.Name(), u.sig(m.Type().(*types.Signature))))
	}
	return lines
}

func (u *Universe) underlying(t types.Type) string {
	switch t := t.(type) {
	case *types.Struct:
		var fields []string
		for i := 0; i < t.NumFields(); i++ {
			f := t.Field(i)
			if !f.Exported() {
				continue
			}
			s := f.Name() + " " + u.typ(f.Type())
			if f.Embedded() {
				s = "embedded " + u.typ(f.Type())
			}
			if tag := t.Tag(i); tag != "" {
				s += " `" + tag + "`"
			}
			fields = append(fields, s)
		}
		return "struct{" + strings.Join(fields, "; ") + "}"
	case *types.Interface:
		return u.iface(t)
	}
	return u.typ(t)
}

func (u *Universe) iface(t *types.Interface) string {
	var methods []string
	hidden := false
	for i := 0; i < t.NumMethods(); i++ {
		m := t.Method(i)
		if !m.Exported() {
			hidden = true
			continue
		}
		methods = append(methods, m.Name()+u.sig(m.Type().(*types.Signature)))
	}
	sort.Strings(methods)
	if hidden {
		methods = append(methods, "<unexported methods>")
	}
	return "interface{" + strings.Join(methods, "; ") + "}"
}

func (u *Universe) sig(s *types.Signature) string {
	params := u.tuple(s.Params(), s.Variadic())
	results := u.tuple(s.Results(), false)
	out := "(" + params + ")"
	switch s.Results().Len() {
	case 0:
	case 1:
		out += " " + results
	default:
		out += " (" + results + ")"
	}
	return out
}

func (u *Universe) tuple(t *types.Tuple, variadic bool) string {
	parts := make([]string, t.Len())
	for i := 0; i < t.Len(); i++ {
		typ := t.At(i).Type()
		if variadic && i == t.Len()-1 {
			parts[i] = "..." + u.typ(typ.(*types.Slice).Elem())
			continue
		}
		parts[i] = u.typ(typ)
	}
	return strings.Join(parts, ", ")
}

func (u *Universe) typ(t types.Type) string {
	switch t := t.(type) {
	case *types.Alias:
		return u.typ(types.Unalias(t))
	case *types.Named:
		obj := t.Obj()
		if name, ok := u.names[obj]; ok {
			return name
		}
		if obj.Pkg() == nil {
			return obj.Name()
		}
		path := obj.Pkg().Path()
		if path == Module || strings.HasPrefix(path, Module+"/") {
			path = "UNMAPPED:" + path
		}
		return path + "." + obj.Name()
	case *types.Basic:
		return t.Name()
	case *types.Pointer:
		return "*" + u.typ(t.Elem())
	case *types.Slice:
		return "[]" + u.typ(t.Elem())
	case *types.Array:
		return fmt.Sprintf("[%d]%s", t.Len(), u.typ(t.Elem()))
	case *types.Map:
		return "map[" + u.typ(t.Key()) + "]" + u.typ(t.Elem())
	case *types.Chan:
		switch t.Dir() {
		case types.SendOnly:
			return "chan<- " + u.typ(t.Elem())
		case types.RecvOnly:
			return "<-chan " + u.typ(t.Elem())
		}
		return "chan " + u.typ(t.Elem())
	case *types.Signature:
		return "func" + u.sig(t)
	case *types.Interface:
		return u.iface(t)
	case *types.Struct:
		return u.underlying(t)
	}
	return t.String()
}

// Diff compares the API of the package at path with the golden file and
// returns a readable difference, or "" when they match. With update set it
// rewrites the golden file instead.
func (u *Universe) Diff(path, golden string, update bool) (string, error) {
	lines, err := u.API(path)
	if err != nil {
		return "", err
	}
	got := strings.Join(lines, "\n") + "\n"
	if update {
		return "", os.WriteFile(golden, []byte(got), 0o600)
	}
	want, err := os.ReadFile(golden) // #nosec G304 -- test fixture path
	if err != nil {
		return "", err
	}
	return lineDiff(strings.Split(strings.TrimSuffix(string(want), "\n"), "\n"), lines), nil
}

func lineDiff(want, got []string) string {
	inWant := make(map[string]bool, len(want))
	for _, l := range want {
		inWant[l] = true
	}
	inGot := make(map[string]bool, len(got))
	for _, l := range got {
		inGot[l] = true
	}
	var sb strings.Builder
	for _, l := range want {
		if !inGot[l] {
			sb.WriteString("- " + l + "\n")
		}
	}
	for _, l := range got {
		if !inWant[l] {
			sb.WriteString("+ " + l + "\n")
		}
	}
	return sb.String()
}

// Mirror checks that the facade package exposes every exported identifier
// of each source package: a type through an alias of the very same type, a
// constant with the same type and value, a variable or function with an
// identical type. renames maps "pkgname.Name" to the facade name when it
// differs; skip lists "pkgname.Name" identifiers the facade deliberately
// leaves out. It returns one line per problem.
func (u *Universe) Mirror(facade string, sources []string, renames map[string]string, skip map[string]bool) ([]string, error) {
	fpkg, err := u.Package(facade)
	if err != nil {
		return nil, err
	}
	var problems []string
	for _, src := range sources {
		spkg, err := u.Package(src)
		if err != nil {
			return nil, err
		}
		scope := spkg.Scope()
		for _, name := range scope.Names() {
			obj := scope.Lookup(name)
			key := spkg.Name() + "." + name
			if !obj.Exported() || skip[key] {
				continue
			}
			fname := name
			if r, ok := renames[key]; ok {
				fname = r
			}
			fobj := fpkg.Scope().Lookup(fname)
			if fobj == nil {
				problems = append(problems, fmt.Sprintf("%s: no %s in %s", key, fname, facade))
				continue
			}
			if msg := sameAs(obj, fobj); msg != "" {
				problems = append(problems, fmt.Sprintf("%s: %s %s", key, fname, msg))
			}
		}
	}
	return problems, nil
}

func sameAs(obj, fobj types.Object) string {
	switch o := obj.(type) {
	case *types.TypeName:
		if _, ok := fobj.(*types.TypeName); !ok || !types.Identical(types.Unalias(o.Type()), types.Unalias(fobj.Type())) {
			return "is not an alias of the same type"
		}
	case *types.Const:
		fc, ok := fobj.(*types.Const)
		if !ok || !types.Identical(o.Type(), fc.Type()) || o.Val().ExactString() != fc.Val().ExactString() {
			return "is not the same constant"
		}
	case *types.Var:
		if _, ok := fobj.(*types.Var); !ok || !types.Identical(o.Type(), fobj.Type()) {
			return "is not a variable of the same type"
		}
	case *types.Func:
		if _, ok := fobj.(*types.Func); !ok || !types.Identical(o.Type(), fobj.Type()) {
			return "is not a function with the same signature"
		}
	}
	return ""
}
