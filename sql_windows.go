package main

func parse(query string) (*q.SelectStmt, bool) {
	return nil, false
}

func identifiers(slct *q.SelectStmt) ([]string, bool) {
	return nil, false
}

func filter(slct *q.SelectStmt) func(m map[string]any) bool {
	return func(_ map[string]any) bool {
		return false
	}
}
