package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_identifiers(t *testing.T) {
	tests := []struct {
		in     string
		idents []string
		ok     bool
	}{
		{
			"SELECT a FROM x",
			[]string{"a"},
			true,
		},
		{
			"SELECT a, '1' FROM x",
			[]string{"a"},
			true,
		},
		{
			"SELECT a, b FROM x",
			[]string{"a", "b"},
			true,
		},
		{
			"SELECT avg(b) FROM x",
			[]string{"b"},
			true,
		},
		{
			"SELECT avg(b+1) FROM x",
			[]string{"b"},
			true,
		},
		{
			"SELECT b+1 FROM x",
			[]string{"b"},
			true,
		},
		{
			"SELECT a FROM x WHERE b > 1",
			[]string{"a", "b"},
			true,
		},
		{
			"SELECT * FROM x",
			nil,
			false,
		},
		{
			"SELECT a FROM x, y",
			nil,
			false,
		},
	}

	for _, test := range tests {
		t.Logf("Testing: %s", test.in)
		s, ok := parse(test.in)
		assert.True(t, ok)

		idents, ok := identifiers(s)
		assert.Equal(t, test.idents, idents)
		assert.Equal(t, test.ok, ok)
	}
}

func Test_filter(t *testing.T) {
	tests := []struct {
		query   string
		inRows  []map[string]any
		outRows []map[string]any
	}{
		{
			"SELECT a FROM x",
			[]map[string]any{
				{"a": 1},
				{"a": 2},
			},
			[]map[string]any{
				{"a": 1},
				{"a": 2},
			},
		},
		{
			"SELECT a FROM x WHERE avg(a) > 12",
			[]map[string]any{
				{"a": 1},
				{"a": 2},
			},
			[]map[string]any{
				{"a": 1},
				{"a": 2},
			},
		},
		{
			"SELECT a FROM x WHERE a = 12",
			[]map[string]any{
				{"a": 1},
				{"a": 12},
			},
			[]map[string]any{
				{"a": 12},
			},
		},
	}

	for _, test := range tests {
		t.Logf("Testing: %s", test.query)
		s, ok := parse(test.query)
		assert.True(t, ok)

		f := filter(s)
		var end []map[string]any
		for _, r := range test.inRows {
			canFilter := f(r)
			if !canFilter {
				end = append(end, r)
			}
		}

		assert.Equal(t, test.outRows, end)
	}
}
