package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/multiprocessio/datastation/runner"
)

type SQLiteResultItemWriter struct {
	db        *sql.DB
	fields    []string
	tableName   string
	rowBuffer runner.Vector[any]
}

func openSQLiteResultItemWriter(f string, tableName string) (runner.ResultItemWriter, error) {
	var sw SQLiteResultItemWriter
	sw.tableName = tableName

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
	_, err := sw.db.Exec("CREATE TABLE \"" + sw.tableName + "\"(" + strings.Join(columns, ", ") + ");")
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
	query.WriteString("INSERT INTO \"" + sw.tableName + "\" VALUES ")
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

func (sw *SQLiteResultItemWriter) flush(newFields []string) error {
	if len(newFields) > 0 {
		for _, f := range newFields {
			_, err := sw.db.Exec("ALTER TABLE \""+sw.tableName+"\" ADD \""+ f +"\" TEXT")
			if err != nil {
				return err
			}
		}
	}

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

func (sw *SQLiteResultItemWriter) getMapFields(m map[string]any) []string {
	var newFields []string

	for k, v := range m {
		switch t := v.(type) {
		case map[string]any:
			newFields = append(newFields, sw.getMapFields(t)...)
		default:
			newFields = append(newFields, k)
		}
	}

	return newFields
}

func (sw *SQLiteResultItemWriter) WriteRow(r any, written int) error {
	m, ok := r.(map[string]any)
	if !ok {
		return fmt.Errorf("Row must be a map, got: %#v", r)
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

	var newFields []string

	for _, field := range sw.fields {
		v := runner.GetObjectAtPath(m, field)
		switch t := v.(type) {
			// TODO: it's possible this doesn't support cases like if there's a map[int8]string due to Parquet or something.
		case map[string]any:
			for _, f := range sw.getMapFields(t) {
				// Only add new fields
				newField := true
				for _, nf := range newFields {
					if nf == f {
						newField = false
						break
					}
				}

				if newField {
					newFields = append(newFields, f)
					sw.fields = append(sw.fields, f)
				}
			}
		case []any:
			bs, err := json.Marshal(t)
			if err != nil {
				return err
			}
			v = string(bs)
		}
		sw.rowBuffer.Append(v)
	}

	// Flush data
	if written > 0 && written%10000 == 0 {
		return sw.flush(newFields)
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
		// TODO: this is a bug. new fields should still be added here too but they are silently dropped.
		err := sw.flush(nil)
		if err != nil {
			return err
		}
	}
	return sw.db.Close()
}
