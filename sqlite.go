package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/multiprocessio/datastation/runner"
)

type SQLiteResultItemWriterOptions struct {
	convertNumbers bool
	prefilter      func(map[string]any) bool
	fieldsOverride []string
}

type SQLiteResultItemWriter struct {
	tableCreated bool
	db           *sql.DB
	fields       []string
	panelId      string
	rowBuffer    runner.Vector[any]

	SQLiteResultItemWriterOptions
}

func openSQLiteResultItemWriter(f string, panelId string, opts SQLiteResultItemWriterOptions) (runner.ResultItemWriter, error) {
	var sw SQLiteResultItemWriter
	sw.panelId = panelId
	sw.SQLiteResultItemWriterOptions = opts

	sw.fields = opts.fieldsOverride

	sw.rowBuffer = runner.Vector[any]{}

	var err error
	sw.db, err = sql.Open("sqlite3_extended", f)
	if err != nil {
		return nil, err
	}

	return &sw, nil
}

func (sw *SQLiteResultItemWriter) createTable() error {
	fieldType := "TEXT"
	if sw.convertNumbers {
		fieldType = "NUMERIC"
	}

	var columns []string
	for _, field := range sw.fields {
		columns = append(columns, `"`+field+`" `+fieldType)
	}
	create := "CREATE TABLE \"" + sw.panelId + "\"(" + strings.Join(columns, ", ") + ");"
	_, err := sw.db.Exec(create)
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
	rowsInBatch := 100
	query := sw.makeQuery(rowsInBatch)

	stmt, err := sw.db.Prepare(query)
	if err != nil {
		return err
	}

	rows := sw.rowBuffer.Index() / len(sw.fields)
	args := sw.rowBuffer.List()
	var leftover []any
	batchArgs := make([]any, rowsInBatch*len(sw.fields))
	for i := 0; i < rows; i += rowsInBatch {
		if i+rowsInBatch > rows {
			leftover = make([]any, (rows-i)*len(sw.fields))
			copy(leftover, args[i*len(sw.fields):])
			break
		}

		copy(batchArgs, args[i*len(sw.fields):(i+rowsInBatch)*len(sw.fields)])
		_, err = stmt.Exec(batchArgs...)
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
		return fmt.Errorf("Row must be a map, got: %#v", r)
	}

	if sw.prefilter != nil {
		canSkip := sw.prefilter(m)
		if canSkip {
			return nil
		}
	}

	if !sw.tableCreated {
		if len(sw.fields) == 0 {
			for key := range m {
				sw.fields = append(sw.fields, key)
			}
		}

		err := sw.createTable()
		if err != nil {
			return err
		}

		sw.tableCreated = true
	}

	for _, field := range sw.fields {
		v := m[field]
		switch t := v.(type) {
		case []any:
			bs, err := json.Marshal(t)
			if err != nil {
				return err
			}
			v = string(bs)
			// TODO: don't keep this
		case map[string]any:
			v = nil
		}
		sw.rowBuffer.Append(v)
	}

	// Flush data
	if written > 0 && written%10000 == 0 {
		return sw.flush()
	}

	return nil
}

func (sw *SQLiteResultItemWriter) SetNamespace(key string) error {
	return fmt.Errorf("SetNamespace unimplemented")
}

func (sw *SQLiteResultItemWriter) Shape(id string, maxBytesToRead, sampleSize int) (*runner.Shape, error) {
	return nil, fmt.Errorf("Shape unimplemented")
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
