package main

type SelectStmt struct{}

func parse(query string) (*SelectStmt, bool) {
	return nil, false
}

func identifiers(slct *SelectStmt) ([]string, bool) {
	return nil, false
}

func filter(slct *SelectStmt) func(m map[string]any) bool {
	return func(_ map[string]any) bool {
		return false
	}
}
