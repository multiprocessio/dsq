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

func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func (sw *SQLiteResultItemWriter) makeQuery(rows int) string {
	var query strings.Builder
	query.WriteString("INSERT INTO \"" + sw.panelId + "\" VALUES ")
	for i := 0; i < rows; i++ {
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

	return query.String()
}

func (sw *SQLiteResultItemWriter) flush() error {
	rowsInBatch := 10
	query := sw.makeQuery(rowsInBatch)

	stmt, err := sw.db.Prepare(query)
	if err != nil {
		return err
	}

	rows := sw.rowBuffer.Index() / len(sw.fields)
	args := sw.rowBuffer.List()
	var leftover []any
	for i := 0; i < rows; i += rowsInBatch {
		if (i+rowsInBatch)*len(sw.fields) > rows {
			leftover = args[i*len(sw.fields):]
			break
		}

		_, err = stmt.Exec(args[i*len(sw.fields) : (i+rowsInBatch)*len(sw.fields)]...)
		if err != nil {
			return err
		}
	}

	err = stmt.Close()
	if err != nil {
		return err
	}

	if len(leftover) > 0 {
		remainingRows := len(leftover) / len(sw.fields)
		_, err := sw.db.Exec(sw.makeQuery(remainingRows), leftover...)
		if err != nil {
			return err
		}
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
	if written > 0 && written%10000 == 0 {
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
