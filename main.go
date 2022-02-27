package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/multiprocessio/datastation/runner"

	"github.com/google/uuid"
	"github.com/olekukonko/tablewriter"
)

func resolveContentType(fileExtensionOrContentType string) runner.MimeType {
	if strings.Contains(fileExtensionOrContentType, os.Separator) {
		return runner.MimeType(fileExtensionOrContentType)
	}

	return runner.GetMimeType("x."+fileExtensionOrContentType, runner.ContentTypeInfo{})
}

func openTruncate(out string) (*os.File, error) {
	base := filepath.Dir(out)
	_ = os.Mkdir(base, os.ModePerm)
	return os.OpenFile(out, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, os.ModePerm)
}

func evalFileInto(file string, out *os.File) error {
	if file == "" {
		return fmt.Errorf(`First argument when not used in a pipe should be a file. e.g. 'dsq test.csv "SELECT COUNT(1) FROM {}"'`)
	}

	mimetype := runner.GetMimeType(file, runner.ContentTypeInfo{})
	if mimetype == "" {
		return fmt.Errorf("Unknown mimetype for file: %s.", file)
	}

	return runner.TransformFile(file, runner.ContentTypeInfo{}, out)
}

func getShape(resultFile, panelId string) (*runner.Shape, error) {
	s, err := runner.ShapeFromFile(resultFile, panelId, 10_000, 100)
	if err != nil {
		return nil, err
	}

	if !runner.ShapeIsObjectArray(*s) {
		rest := "."
		if panelId != "" {
			rest = ": " + panelId + "."
		}
		return nil, fmt.Errorf("Input is not an array of objects%s", rest)
	}

	return s, nil
}

var Version = "latest"

var HELP = `dsq (Version ` + Version + `) - commandline SQL engine for data files

Usage:  dsq [file...] $query
        dsq $file [query]
        cat $file | dsq -s $filetype [query]

dsq is a tool for running SQL on one or more data files. It uses
SQLite's SQL dialect. Files as tables are accessible via "{N}" where N
is the 0-based index of the file in the commandline.

The shorthand "{}" is replaced with "{0}".

Examples:

    # This simply dumps the CSV as JSON
    $ dsq test.csv

    # This dumps the first 10 rows of the parquet file as JSON.
    $ dsq data.parquet "SELECT * FROM {} LIMIT 10"

    # This joins two datasets of differing origin types (CSV and JSON).
    $ dsq testdata/join/users.csv testdata/join/ages.json \
          "select {0}.name, {1}.age from {0} join {1} on {0}.id = {1}.id"

See the repo for more details: https://github.com/multiprocessio/dsq.`

func _main() error {
	log.SetFlags(0)
	runner.Verbose = false
	var nonFlagArgs []string
	stdin := false
	pretty := false
	for _, arg := range os.Args[1:] {
		if arg == "-v" || arg == "--verbose" {
			runner.Verbose = true
			continue
		}

		if arg == "-s" || arg == "--stdin" {
			stdin = true
			continue
		}

		if arg == "-h" || arg == "--help" {
			log.Println(HELP)
			return nil
		}

		if arg == "-p" || arg == "--pretty" {
			pretty = true
			continue
		}

		nonFlagArgs = append(nonFlagArgs, arg)
	}

	lastNonFlagArg := ""
	files := nonFlagArgs

	// Empty marker meaning to process from stdin
	if stdin {
		files = append([]string{""}, files...)
	}

	if len(nonFlagArgs) > 1 {
		lastNonFlagArg = nonFlagArgs[len(nonFlagArgs)-1]
		if strings.Contains(lastNonFlagArg, " ") {
			files = files[:len(files)-1]
		}
	}

	if len(files) == 0 {
		return errors.New("No input files.")
	}

	projectTmp, err := ioutil.TempFile("", "dsq-project")
	if err != nil {
		return err
	}
	defer os.Remove(projectTmp.Name())
	project := &runner.ProjectState{
		Id: projectTmp.Name(),
		Pages: []runner.ProjectPage{
			{
				Panels: nil,
			},
		},
	}

	tmpDir, err := os.MkdirTemp("", "dsq")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	ec := runner.NewEvalContext(*runner.DefaultSettings, tmpDir)
	for i := 0; i < len(files); i++ {
		file := files[i]
		panelId := uuid.New().String()
		resultFile := ec.GetPanelResultsFile(project.Id, panelId)
		out, err := openTruncate(resultFile)
		if err != nil {
			return err
		}

		readFromStdin := false
		if file == "" {
			b, err := ioutil.ReadAll(os.Stdin)
			if err == nil {
				if i == len(files)-1 {
					return errors.New("Expected file extension or mimetype: e.g. cat x.csv | dsq -s csv, or cat x.csv | dsq -s text/csv")
				}
				mimetype := files[i+1]
				if !strings.Contains(mimetype, os.Separator) {
					mimetype = string(runner.GetMimeType("x."+mimetype, runner.ContentTypeInfo{}))
				}

				if mimetype == "" {
					return fmt.Errorf("Unknown mimetype or file extension: %s.", mimetype)
				}
				i += 1

				cti := runner.ContentTypeInfo{Type: string(mimetype)}
				err := runner.TransformReader(bytes.NewReader(b), "", cti, out)
				if err != nil {
					return err
				}

				s, err := getShape(resultFile, "")
				if err != nil {
					return err
				}

				project.Pages[0].Panels = append(project.Pages[0].Panels, runner.PanelInfo{
					ResultMeta: runner.PanelResult{
						Shape: *s,
					},
					Id:   panelId,
					Name: uuid.New().String(),
				})

				readFromStdin = true
			}

			continue
		}

		if !readFromStdin {
			err := evalFileInto(file, out)
			if err != nil {
				return err
			}
		}

		s, err := getShape(resultFile, file)
		if err != nil {
			return err
		}

		project.Pages[0].Panels = append(project.Pages[0].Panels, runner.PanelInfo{
			ResultMeta: runner.PanelResult{
				Shape: *s,
			},
			Id:   panelId,
			Name: uuid.New().String(),
		})

		out.Close()
	}

	// No query, just dump transformed file directly out
	if lastNonFlagArg == "" {
		resultFile := ec.GetPanelResultsFile(project.Id, project.Pages[0].Panels[0].Id)
		fd, err := os.Open(resultFile)
		if err != nil {
			return err
		}
		defer fd.Close()

		_, err = io.Copy(os.Stdout, fd)
		if err != nil {
			return err
		}

		return nil
	}

	connector, err := runner.MakeTmpSQLiteConnector()
	if err != nil {
		return err
	}
	project.Connectors = append(project.Connectors, *connector)

	query := lastNonFlagArg
	query = strings.ReplaceAll(query, "{}", "DM_getPanel(0)")
	for i := 0; i < len(project.Pages[0].Panels); i++ {
		query = strings.ReplaceAll(query, fmt.Sprintf("{%d}", i), fmt.Sprintf("DM_getPanel(%d)", i))
	}
	panel := &runner.PanelInfo{
		Type:    runner.DatabasePanel,
		Content: query,
		Id:      uuid.New().String(),
		Name:    uuid.New().String(),
		DatabasePanelInfo: &runner.DatabasePanelInfo{
			Database: runner.DatabasePanelInfoDatabase{
				ConnectorId: connector.Id,
			},
		},
	}

	err = ec.EvalDatabasePanel(project, 0, panel, nil)
	if err != nil {
		return err
	}

	resultFile := ec.GetPanelResultsFile(project.Id, panel.Id)
	fd, err := os.Open(resultFile)
	if err != nil {
		return fmt.Errorf("Could not open results file: %s", err)
	}

	if !pretty {
		// Dump the result to stdout
		_, err = io.Copy(os.Stdout, fd)
		if err != nil {
			return err
		}
		fmt.Println()
		return nil
	}

	s, err := runner.ShapeFromFile(resultFile, "results", 10_000, 100)
	if err != nil {
		return err
	}

	var columns []string
	for name := range s.ArrayShape.Children.ObjectShape.Children {
		columns = append(columns, name)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(columns)
	table.SetAutoFormatHeaders(false)

	dec := json.NewDecoder(fd)
	var rows []map[string]interface{}
	err = dec.Decode(&rows)
	if err != nil {
		return err
	}

	for _, objRow := range rows {
		var row []string
		for _, column := range columns {
			var cell string
			switch t := objRow[column].(type) {
			case bool, byte, complex64, complex128, error, float32, float64,
				int, int8, int16, int32, int64,
				uint, uint16, uint32, uint64, uintptr:
				cell = fmt.Sprintf("%#v", t)
			case string:
				cell = t
			default:
				cellBytes, _ := json.Marshal(t)
				cell = string(cellBytes)
			}
			row = append(row, cell)
		}
		table.Append(row)
	}

	table.Render()
	return nil
}

func main() {
	err := _main()
	if err != nil {
		log.Fatal(err)
	}
}
