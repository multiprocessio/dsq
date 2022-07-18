//go:build !windows

package main

import (
	"fmt"
	"strings"

	"github.com/multiprocessio/datastation/runner"

	q "github.com/pganalyze/pg_query_go/v2"
)

func parse(query string) (*q.SelectStmt, bool) {
	ast, err := q.Parse(query)
	if err != nil {
		return nil, false
	}

	return ast.Stmts[0].Stmt.GetSelectStmt(), true
}

func getValidIdentifier(n *q.Node) ([]string, bool) {
	if n == nil {
		return nil, false
	}

	// Constants are fine
	if n.GetAConst() != nil {
		return nil, true
	}

	if fc := n.GetFuncCall(); fc != nil {
		var fields []string
		for _, arg := range fc.Args {
			_fields, ok := getValidIdentifier(arg)
			if !ok {
				return nil, false
			}

			fields = append(fields, _fields...)
		}

		return fields, true
	}

	if e := n.GetAExpr(); e != nil {
		l, ok := getValidIdentifier(e.Lexpr)
		if !ok {
			return nil, false
		}

		r, ok := getValidIdentifier(e.Rexpr)
		if !ok {
			return nil, false
		}

		return append(l, r...), true
	}

	// Otherwise must be an identifier
	cr := n.GetColumnRef()
	if cr == nil {
		return nil, false
	}

	var parts []string
	for _, field := range cr.Fields {
		s := field.GetString_()
		if s == nil {
			return nil, false
		}

		parts = append(parts, s.Str)
	}

	s := strings.Join(parts, ".")
	return []string{s}, true
}

func identifiers(slct *q.SelectStmt) ([]string, bool) {
	var fields []string
	for _, t := range slct.TargetList {
		v := t.GetResTarget().GetVal()

		_fields, ok := getValidIdentifier(v)
		if !ok {
			return nil, false
		}

		fields = append(fields, _fields...)
	}

	if len(slct.FromClause) != 1 {
		return nil, false
	}
	rv := slct.FromClause[0].GetRangeVar()
	if rv == nil {
		return nil, false
	}
	if rv.GetRelname() == "" {
		return nil, false
	}

	if slct.WhereClause != nil {
		where, ok := getValidIdentifier(slct.WhereClause)
		if !ok {
			return nil, false
		}
		fields = append(fields, where...)
	}

	return fields, true
}

func evalNode(n *q.Node, row map[string]any) any {
	if n == nil {
		return nil
	}

	if c := n.GetAConst(); c != nil {
		if i := c.GetVal().GetInteger(); i != nil {
			return fmt.Sprintf("%d", i.Ival)
		}

		if s := c.GetVal().GetString_(); s != nil {
			return s.Str
		}

		// Unsupported const type
		return nil
	}

	// Filtering on function calls unsupported
	if fc := n.GetFuncCall(); fc != nil {
		return nil
	}

	if e := n.GetAExpr(); e != nil {
		if len(e.Name) != 1 {
			return nil
		}

		_l := evalNode(e.Lexpr, row)
		if _l == nil {
			return nil
		}

		l, ok := _l.(string)
		if !ok {
			return nil
		}

		_r := evalNode(e.Rexpr, row)
		if _r == nil {
			return nil
		}

		r, ok := _r.(string)
		if !ok {
			return nil
		}

		switch e.Name[0].GetString_().Str {
		case ">":
			return l > r
		case "<":
			return l < r
		case ">=":
			return l >= r
		case "<=":
			return l <= r
		case "=":
			return l == r
		}
	}

	// Otherwise must be an identifier
	cr := n.GetColumnRef()
	if cr == nil {
		return nil
	}

	if len(cr.Fields) != 1 {
		return nil
	}

	s := cr.Fields[0].GetString_()
	if s == nil {
		return nil
	}

	v := runner.GetObjectAtPath(row, s.Str)
	switch v.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf("%#v", v)
	}
}

func filter(slct *q.SelectStmt) func(m map[string]any) bool {
	return func(row map[string]any) bool {
		if slct.WhereClause == nil {
			return false
		}

		x := evalNode(slct.WhereClause, row)
		return x != true && x != nil
	}
}
