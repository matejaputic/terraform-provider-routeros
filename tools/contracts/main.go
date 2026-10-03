// Static reference inspection only. No SDK imports, callbacks or schema constructors run.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/constant"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
)

type Input struct {
	Files map[string]string `json:"files"`
}
type Inspector struct {
	fs        *token.FileSet
	files     map[string]*ast.File
	globals   map[string]ast.Expr
	functions map[string]*ast.FuncDecl
}

func (i *Inspector) text(e ast.Node) string {
	var b bytes.Buffer
	_ = format.Node(&b, i.fs, e)
	return b.String()
}
func (i *Inspector) location(n ast.Node) map[string]any {
	p := i.fs.Position(n.Pos())
	return map[string]any{"file": p.Filename, "line": p.Line}
}
func (i *Inspector) scalar(e ast.Expr, seen map[string]bool) (any, bool) {
	switch x := e.(type) {
	case *ast.BasicLit:
		v := constant.MakeFromLiteral(x.Value, x.Kind, 0)
		switch v.Kind() {
		case constant.String:
			return constant.StringVal(v), true
		case constant.Int:
			n, ok := constant.Int64Val(v)
			return n, ok
		case constant.Float:
			n, ok := constant.Float64Val(v)
			return n, ok
		}
	case *ast.Ident:
		if x.Name == "true" {
			return true, true
		}
		if x.Name == "false" {
			return false, true
		}
		if seen[x.Name] {
			return nil, false
		}
		v, ok := i.globals[x.Name]
		if ok {
			next := map[string]bool{}
			for k, v := range seen {
				next[k] = v
			}
			next[x.Name] = true
			return i.scalar(v, next)
		}
	case *ast.CompositeLit:
		if _, ok := x.Type.(*ast.ArrayType); ok {
			values := []any{}
			for _, item := range x.Elts {
				value, ok := i.scalar(item, seen)
				if !ok {
					return nil, false
				}
				values = append(values, value)
			}
			return values, true
		}
	case *ast.BinaryExpr:
		if x.Op == token.ADD {
			a, ok := i.scalar(x.X, seen)
			b, bok := i.scalar(x.Y, seen)
			as, aok := a.(string)
			bs, sok := b.(string)
			if ok && bok && aok && sok {
				return as + bs, true
			}
		}
	}
	return nil, false
}
func (i *Inspector) literal(e ast.Expr, seen map[string]bool) (*ast.CompositeLit, bool) {
	switch x := e.(type) {
	case *ast.CompositeLit:
		return x, true
	case *ast.UnaryExpr:
		if x.Op == token.AND {
			return i.literal(x.X, seen)
		}
	case *ast.Ident:
		if !seen[x.Name] {
			v, ok := i.globals[x.Name]
			if ok {
				next := map[string]bool{}
				for k, v := range seen {
					next[k] = v
				}
				next[x.Name] = true
				return i.literal(v, next)
			}
		}
	}
	return nil, false
}
func (i *Inspector) property(e ast.Expr) map[string]any {
	p := map[string]any{"expression": i.text(e), "source": i.location(e), "status": "needs-review"}
	if v, ok := i.scalar(e, map[string]bool{}); ok {
		p["value"] = v
		p["status"] = "static"
	}
	if call, ok := e.(*ast.CallExpr); ok {
		args := []any{}
		for _, arg := range call.Args {
			args = append(args, i.property(arg))
		}
		p["call"] = map[string]any{"callee": i.text(call.Fun), "arguments": args, "executed": false}
	}
	return p
}
func (i *Inspector) field(e ast.Expr) map[string]any {
	r := map[string]any{"expression": i.text(e), "source": i.location(e), "status": "needs-review"}
	if lit, ok := i.literal(e, map[string]bool{}); ok {
		props := map[string]any{}
		for _, el := range lit.Elts {
			if kv, ok := el.(*ast.KeyValueExpr); ok {
				if key, ok := kv.Key.(*ast.Ident); ok {
					props[key.Name] = i.property(kv.Value)
				}
			}
		}
		r["properties"] = props
		r["definition_source"] = i.location(lit)
		r["status"] = "static-declaration"
	}
	return r
}
func schemaMap(lit *ast.CompositeLit) bool {
	t, ok := lit.Type.(*ast.MapType)
	if !ok {
		return false
	}
	v, ok := t.Value.(*ast.StarExpr)
	if !ok {
		return false
	}
	s, ok := v.X.(*ast.SelectorExpr)
	return ok && s.Sel.Name == "Schema"
}
func (i *Inspector) contract(fn *ast.FuncDecl) map[string]any {
	fields := map[string]any{}
	metadata := map[string]any{}
	maps := []any{}
	mutations := []any{}
	resource := map[string]any{}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if lit, ok := n.(*ast.CompositeLit); ok {
			if schemaMap(lit) {
				maps = append(maps, i.location(lit))
				for _, el := range lit.Elts {
					kv, ok := el.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					key, ok := i.scalar(kv.Key, map[string]bool{})
					name, sok := key.(string)
					if !ok || !sok {
						mutations = append(mutations, map[string]any{"kind": "unresolved-field-key", "expression": i.text(kv.Key), "source": i.location(kv.Key)})
						continue
					}
					target := fields
					if strings.HasPrefix(name, "___") {
						target = metadata
					}
					if _, exists := target[name]; exists {
						mutations = append(mutations, map[string]any{"kind": "multiple-declarations", "field": name, "source": i.location(kv.Key)})
					}
					target[name] = i.field(kv.Value)
				}
				return false
			}
			if s, ok := lit.Type.(*ast.SelectorExpr); ok && s.Sel.Name == "Resource" {
				for _, el := range lit.Elts {
					if kv, ok := el.(*ast.KeyValueExpr); ok {
						if key, ok := kv.Key.(*ast.Ident); ok {
							resource[key.Name] = i.property(kv.Value)
						}
					}
				}
			}
		}
		switch x := n.(type) {
		case *ast.AssignStmt:
			for _, lhs := range x.Lhs {
				if _, ok := lhs.(*ast.IndexExpr); ok {
					mutations = append(mutations, map[string]any{"kind": "indexed-assignment", "expression": i.text(x), "source": i.location(x)})
				}
			}
		case *ast.CallExpr:
			if id, ok := x.Fun.(*ast.Ident); ok && (id.Name == "delete" || id.Name == "append") {
				mutations = append(mutations, map[string]any{"kind": "possible-schema-mutation", "expression": i.text(x), "source": i.location(x)})
			}
		}
		return true
	})
	status := "needs-review"
	if len(maps) == 1 && len(mutations) == 0 {
		status = "static-declarations-only"
	}
	return map[string]any{"source": i.location(fn), "status": status, "fields": fields, "metadata": metadata, "schema_maps": maps, "resource_properties": resource, "review_findings": mutations}
}
func inspect(input Input) (result map[string]any, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			result = nil
			err = fmt.Errorf("unsupported static reference shape: %v", recovered)
		}
	}()
	i := &Inspector{token.NewFileSet(), map[string]*ast.File{}, map[string]ast.Expr{}, map[string]*ast.FuncDecl{}}
	paths := []string{}
	for p := range input.Files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	tests := []any{}
	for _, p := range paths {
		f, err := parser.ParseFile(i.fs, p, input.Files[p], parser.ParseComments)
		if err != nil {
			return nil, err
		}
		i.files[p] = f
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if strings.HasSuffix(p, "_test.go") {
					if strings.HasPrefix(d.Name.Name, "Test") {
						tests = append(tests, map[string]any{"name": d.Name.Name, "source": i.location(d)})
					}
					continue
				}
				if d.Recv == nil {
					if _, ok := i.functions[d.Name.Name]; ok {
						return nil, fmt.Errorf("duplicate function %s", d.Name.Name)
					}
					i.functions[d.Name.Name] = d
				}
			case *ast.GenDecl:
				if strings.HasSuffix(p, "_test.go") {
					continue
				}
				for _, s := range d.Specs {
					if v, ok := s.(*ast.ValueSpec); ok && len(v.Names) == len(v.Values) {
						for n, id := range v.Names {
							i.globals[id.Name] = v.Values[n]
						}
					}
				}
			}
		}
	}
	registrations := map[string]map[string]string{"resources": {}, "data_sources": {}}
	for _, p := range paths {
		if strings.HasSuffix(p, "_test.go") {
			continue
		}
		ast.Inspect(i.files[p], func(n ast.Node) bool {
			kv, ok := n.(*ast.KeyValueExpr)
			if !ok {
				return true
			}
			id, ok := kv.Key.(*ast.Ident)
			if !ok {
				return true
			}
			category := ""
			if id.Name == "ResourcesMap" {
				category = "resources"
			}
			if id.Name == "DataSourcesMap" {
				category = "data_sources"
			}
			if category == "" {
				return true
			}
			lit, ok := kv.Value.(*ast.CompositeLit)
			if !ok {
				panic("registration map is not literal")
			}
			for _, el := range lit.Elts {
				entry, ok := el.(*ast.KeyValueExpr)
				if !ok {
					panic("non-keyed registration")
				}
				key, ok := i.scalar(entry.Key, map[string]bool{})
				name, sok := key.(string)
				if !ok || !sok {
					panic("unresolved registration name")
				}
				call, ok := entry.Value.(*ast.CallExpr)
				if !ok {
					panic("registration is not constructor call")
				}
				ctor, ok := call.Fun.(*ast.Ident)
				if !ok || len(call.Args) != 0 {
					panic("unsupported registration constructor")
				}
				if _, ok := registrations[category][name]; ok {
					panic("duplicate registration")
				}
				registrations[category][name] = ctor.Name
			}
			return false
		})
	}
	constructors := map[string]any{}
	groups := map[string][]string{}
	for category, entries := range registrations {
		for name, ctor := range entries {
			group := category + ":" + ctor
			groups[group] = append(groups[group], name)
			if _, ok := constructors[ctor]; !ok {
				fn, ok := i.functions[ctor]
				if !ok {
					return nil, fmt.Errorf("missing constructor %s", ctor)
				}
				constructors[ctor] = i.contract(fn)
			}
		}
	}
	for key, names := range groups {
		sort.Strings(names)
		groups[key] = names
	}
	helpers := map[string]any{}
	for name, e := range i.globals {
		if strings.HasPrefix(name, "Prop") || strings.HasPrefix(name, "Meta") || strings.HasPrefix(name, "Key") {
			helpers[name] = i.field(e)
		}
	}
	functions := map[string]any{}
	for name, fn := range i.functions {
		functions[name] = map[string]any{"source": i.location(fn)}
	}
	return map[string]any{"registrations": registrations, "alias_groups": groups, "constructors": constructors, "helpers": helpers, "functions": functions, "tests": tests}, nil
}
func main() {
	var input Input
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	result, err := inspect(input)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(result); err != nil {
		panic(err)
	}
}
