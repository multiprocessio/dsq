package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/multiprocessio/datastation/runner"

	"github.com/google/uuid"
	"github.com/olekukonko/tablewriter"
)

func resolveContentType(fileExtensionOrContentType string) runner.MimeType {
	if strings.Contains(fileExtensionOrContentType, string(filepath.Separator)) {
		return runner.MimeType(fileExtensionOrContentType)
	}

	return runner.GetMimeType("x."+fileExtensionOrContentType, runner.ContentTypeInfo{})
}

func openTruncate(out string) (*os.File, error) {
	base := filepath.Dir(out)
	_ = os.MkdirAll(base, os.ModePerm)
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
	return runner.ShapeFromFile(resultFile, panelId, runner.DefaultShapeMaxBytesToRead, 100)
}

var tableFileRe = regexp.MustCompile(`({(?P<number>[0-9]+)(((,\s*(?P<numbersinglepath>"(?:[^"\\]|\\.)*\"))?)|(,\s*(?P<numberdoublepath>'(?:[^'\\]|\\.)*\'))?)})|({((((?P<singlepath>"(?:[^"\\]|\\.)*\"))?)|((?P<doublepath>'(?:[^'\\]|\\.)*\'))?)})`)

func rewriteQuery(query string) string {
	query = strings.ReplaceAll(query, "{}", "DM_getPanel(0)")

	query = tableFileRe.ReplaceAllStringFunc(query, func(m string) string {
		matchForSubexps := tableFileRe.FindStringSubmatch(m)
		index := "0"
		path := ""
		for i, name := range tableFileRe.SubexpNames() {
			if matchForSubexps[i] == "" {
				continue
			}

			switch name {
			case "number":
				index = matchForSubexps[i]
			case "numberdoublepath", "numbersinglepath", "doublepath", "singlepath":
				path = matchForSubexps[i]
			}
		}

		if path != "" {
			return fmt.Sprintf("DM_getPanel(%s, %s)", index, path)
		}

		return fmt.Sprintf("DM_getPanel(%s)", index)
	})

	return query
}

func dumpJSONFile(file string, pretty bool, schema bool) error {
	fd, err := os.Open(file)
	if err != nil {
		return err
	}
	defer fd.Close()

	if schema {
		s, err := runner.ShapeFromFile(file, "doesn't-matter", runner.DefaultShapeMaxBytesToRead, 100)
		if err != nil {
			return err
		}

		if pretty {
			_, err = fmt.Fprintf(os.Stdout, "%s\n", s.Pretty(""))
			return err
		}

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(s)
	}

	if !pretty {
		// Dump the result to stdout
		_, err := io.Copy(os.Stdout, fd)
		if err != nil {
			return err
		}
		fmt.Println()
		return nil
	}

	s, err := runner.ShapeFromFile(file, "doesn't-matter", runner.DefaultShapeMaxBytesToRead, 100)
	if err != nil {
		return err
	}
	var columns []string
	for name := range s.ArrayShape.Children.ObjectShape.Children {
		columns = append(columns, name)
	}
	sort.Strings(columns)

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

func getFilesContentIteration(stdinTmpCopy *bool, sha1 hash.Hash, fileName, tmp string, i int) (int, error) {
	if fileName == "" {
		i += 1
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return i, err
		}

		if _, err := sha1.Write(b); err != nil {
			return i, err
		}

		file, err := os.OpenFile(tmp, os.O_RDWR, os.ModePerm)
		if err != nil {
			return i, err
		}
		defer file.Close()

		if _, err := file.Write(b); err != nil {
			return i, err
		}
		*stdinTmpCopy = true

		return i, nil
	} else {
		file, err := os.Open(fileName)
		if err != nil {
			return i, err
		}
		defer file.Close()

		_, err = io.Copy(sha1, file)
		if err != nil {
			return i, err
		}

		return i, nil
	}
}

func getFilesContentHash(files []string, tmp string) (string, bool, error) {
	sha1 := sha1.New()
	stdinTmpCopy := false
	var err error

	for i := 0; i < len(files); i++ {
		i, err = getFilesContentIteration(&stdinTmpCopy, sha1, files[i], tmp, i)
		if err != nil {
			return "", false, err
		}
	}

	return hex.EncodeToString(sha1.Sum(nil)), stdinTmpCopy, nil
}

func getCachedDBPath(projectID string) string {
	return filepath.Join(os.TempDir(), projectID+".db")
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

func importFilesIteration(cacheMode bool, filesPtr *[]string, ec *runner.EvalContext, project *runner.ProjectState, i int) (int, error) {
	files := *filesPtr
	file := files[i]
	panelId := uuid.New().String()
	resultFile := ec.GetPanelResultsFile(project.Id, panelId)
	out, err := openTruncate(resultFile)
	if err != nil {
		return i, err
	}
	defer out.Close()

	if file == "" {
		b, err := ioutil.ReadAll(os.Stdin)
		if err == nil {
			if i == len(files)-1 {
				return i, errors.New("Expected file extension or mimetype: e.g. cat x.csv | dsq -s csv, or cat x.csv | dsq -s text/csv")
			}
			mimetype := files[i+1]
			if !strings.Contains(mimetype, string(filepath.Separator)) {
				mimetype = string(runner.GetMimeType("x."+mimetype, runner.ContentTypeInfo{}))
			}

			if mimetype == "" {
				return i, fmt.Errorf("Unknown mimetype or file extension: %s.", mimetype)
			}
			i += 1

			var r io.Reader = bytes.NewReader(b)
			if cacheMode {
				r, err = os.Open(files[len(files)-1])
				if err != nil {
					return i, fmt.Errorf("Error opening copied stdin file: %v", err) // returns as it's not possible to go back to default.
				} else {
					*filesPtr = files[:len(files)-1]
				}
			}

			cti := runner.ContentTypeInfo{Type: string(mimetype)}
			err := runner.TransformReader(r, "", cti, out)
			if err != nil {
				return i, err
			}

			s, err := getShape(resultFile, "")
			if err != nil {
				return i, err
			}

			project.Pages[0].Panels = append(project.Pages[0].Panels, runner.PanelInfo{
				ResultMeta: runner.PanelResult{
					Shape: *s,
				},
				Id:   panelId,
				Name: file,
			})

		}

		return i, nil
	}

	if err := evalFileInto(file, out); err != nil {
		return i, err
	}

	s, err := getShape(resultFile, file)
	if err != nil {
		return i, err
	}

	project.Pages[0].Panels = append(project.Pages[0].Panels, runner.PanelInfo{
		ResultMeta: runner.PanelResult{
			Shape: *s,
		},
		Id:   panelId,
		Name: uuid.New().String(),
	})

	return i, nil
}

func importFiles(cacheMode bool, files []string, ec *runner.EvalContext, project *runner.ProjectState) error {
	var err error
	for i := 0; i < len(files); i++ {
		if i, err = importFilesIteration(cacheMode, &files, ec, project, i); err != nil {
			return err
		}
	}
	return nil
}

func _main() error {
	log.SetFlags(0)
	runner.Verbose = false
	var nonFlagArgs []string
	stdin := false
	pretty := false
	schema := false
	cacheSettings := runner.DefaultCacheSettings
	for _, arg := range os.Args[1:] {
		if arg == "--verbose" {
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

		if arg == "-v" || arg == "--version" {
			log.Println("dsq " + Version)
			return nil
		}

		if arg == "-c" || arg == "--schema" {
			schema = true
			continue
		}

		if arg == "--cache" {
			cacheSettings.Enabled = true
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
	projectID := projectTmp.Name()

	if cacheSettings.Enabled {
		stdinTmpCopy := false
		projectID, stdinTmpCopy, err = getFilesContentHash(files, projectTmp.Name())
		if err != nil {
			log.Printf("Error creating hash for cache mode: %v, defaulting to normal mode", err)
			cacheSettings.Enabled = false
		}
		if stdinTmpCopy {
			files = append(files, projectTmp.Name())
		}
	}
	defer os.Remove(projectTmp.Name())

	project := &runner.ProjectState{
		Id: projectID,
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
	if cacheSettings.Enabled {
		if info, err := os.Stat(getCachedDBPath(projectID)); err == nil && info.Size() != 0 {
			cacheSettings.CachePresent = true
		}
	}

	ec := runner.NewEvalContext(*runner.DefaultSettings, tmpDir)
	if !cacheSettings.CachePresent || !cacheSettings.Enabled {
		err := importFiles(cacheSettings.Enabled, files, &ec, project)
		if err != nil {
			return err
		}
	}

	// No query, just dump transformed file directly out
	if lastNonFlagArg == "" {
		resultFile := ec.GetPanelResultsFile(project.Id, project.Pages[0].Panels[0].Id)
		return dumpJSONFile(resultFile, pretty, schema)
	}

	connector, err := runner.MakeTmpSQLiteConnector()
	if err != nil {
		return err
	}

	if cacheSettings.Enabled {
		path, err := filepath.Abs(getCachedDBPath(projectID))
		if err != nil {
			return err
		}
		connector.DatabaseConnectorInfo.Database.Database = path
	}
	project.Connectors = append(project.Connectors, *connector)

	query := rewriteQuery(lastNonFlagArg)
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

	err = ec.EvalDatabasePanel(project, 0, panel, nil, *cacheSettings)
	if err != nil {
		if e, ok := err.(*runner.DSError); ok && e.Name == "NotAnArrayOfObjectsError" {
			rest := "."
			nth, err := strconv.Atoi(e.TargetPanelId)
			if err == nil {
				rest = ": " + files[nth] + "."
			}
			return fmt.Errorf("Input is not an array of objects%s", rest)
		}

		if e, ok := err.(*runner.DSError); ok && e.Name == "UserError" {
			if e.Message[len(e.Message)-1] != '.' {
				e.Message += "."
			}
			return errors.New(e.Message)
		}
		return err
	}

	resultFile := ec.GetPanelResultsFile(project.Id, panel.Id)
	return dumpJSONFile(resultFile, pretty, schema)
}

func main() {
	err := _main()
	if err != nil {
		log.Fatal(err)
	}
}
