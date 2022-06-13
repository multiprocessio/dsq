package main

import (
	"strings"
	"errors"
	"database/sql"

	"github.com/multiprocessio/datastation/runner"
)

type SQLiteResultItemWriter struct {
	db *sql.DB
	fields []string
	panelId string
	rowBuffer []map[string]any
}

func openSQLiteResultItemWriter(f string, panelId string) (runner.ResultItemWriter, error) {
	var sw SQLiteResultItemWriter
	sw.panelId = panelId

	sw.rowBuffer = make([]map[string]any, 100)

	var err error
	sw.db, err = sql.Open("sqlite3_extended", f)
	if err != nil {
		return nil, err
	}

	return &sw, nil
}

func (sw *SQLiteResultItemWriter) createTable() error {
	var columns []string
	for _, field := range sw.fields {
		columns = append(columns, "TEXT " + field)
	}
	_, err := sw.db.Exec("CREATE TABLE \"" + sw.panelId +"\"("+ strings.Join(columns, ", ") +");")
	return err
}

func (sw *SQLiteResultItemWriter) WriteRow(r any, written int) error {
	m, ok := r.(map[string]any)
	if !ok {
		return errors.New("Row must be a map")
	}

	if len(sw.fields) == 0 {
		for key := range m {
			sw.fields = append(sw.fields, key)
		}

		err := sw.createTable()
		if err != nil {
			return err
		}
	}

	var args []any
	var params []string
	for _, field := range sw.fields {
		args = append(args, m[field])
		params = append(params, "?")
	}

	_, err := sw.db.Exec("INSERT INTO \""+sw.panelId+"\" VALUES ("+ strings.Join(params, ", ") +")")
	return err
}

func (sw *SQLiteResultItemWriter) SetNamespace(key string) error {
	return errors.New("SetNamespace unimplemented")
}

func (sw *SQLiteResultItemWriter) Shape(id string, maxBytesToRead, sampleSize int) (*runner.Shape, error) {
	return nil, errors.New("Shape unimplemented")
}

func (sw *SQLiteResultItemWriter) Close() error {
	return sw.db.Close()
}
