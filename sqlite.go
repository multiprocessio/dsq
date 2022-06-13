package main

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/multiprocessio/datastation/runner"
)

type SQLiteResultItemWriter struct {
	db        *sql.DB
	fields    []string
	panelId   string
	rowBuffer runner.Vector[any]
}

func openSQLiteResultItemWriter(f string, panelId string) (runner.ResultItemWriter, error) {
	var sw SQLiteResultItemWriter
	sw.panelId = panelId

	sw.rowBuffer = runner.Vector[any]{}

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
		columns = append(columns, field+" TEXT")
	}
	_, err := sw.db.Exec("CREATE TABLE \"" + sw.panelId + "\"(" + strings.Join(columns, ", ") + ");")
	return err
}

func (sw *SQLiteResultItemWriter) flush() error {
	var query strings.Builder
	query.WriteString("INSERT INTO \"" + sw.panelId + "\" VALUES ")
	for i := 0; i < sw.rowBuffer.Index(); i++ {
		if i > 0 {
			query.WriteString(", ")
		}

		query.WriteByte('(')
		for i := range sw.fields {
			if i > 0 {
				query.WriteString(", ")
			}
			query.WriteByte('?')
		}
		query.WriteByte(')')
	}
	_, err := sw.db.Exec(query.String(), sw.rowBuffer.List()...)
	if err != nil {
		return err
	}

	sw.rowBuffer.Reset()
	return nil
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

	for _, field := range sw.fields {
		sw.rowBuffer.Append(m[field])
	}

	// Flush data
	if written > 0 && written%100 == 0 {
		return sw.flush()
	}

	return nil
}

func (sw *SQLiteResultItemWriter) SetNamespace(key string) error {
	return errors.New("SetNamespace unimplemented")
}

func (sw *SQLiteResultItemWriter) Shape(id string, maxBytesToRead, sampleSize int) (*runner.Shape, error) {
	return nil, errors.New("Shape unimplemented")
}

func (sw *SQLiteResultItemWriter) Close() error {
	if sw.rowBuffer.Index() > 0 {
		err := sw.flush()
		if err != nil {
			return err
		}
	}
	return sw.db.Close()
}
