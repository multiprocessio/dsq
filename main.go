package main

import (
	"bufio"
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
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/chzyer/readline"
	"github.com/multiprocessio/datastation/runner"

	"github.com/google/uuid"
	"github.com/olekukonko/tablewriter"
)

func openTruncate(out string) (*os.File, error) {
	base := filepath.Dir(out)
	_ = os.MkdirAll(base, os.ModePerm)
	return os.OpenFile(out, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, os.ModePerm)
}

func resolveContentType(fileExtensionOrContentType string) runner.MimeType {
	if strings.Contains(fileExtensionOrContentType, string(filepath.Separator)) {
		return runner.MimeType(fileExtensionOrContentType)
	}

	return runner.GetMimeType("x."+fileExtensionOrContentType, runner.ContentTypeInfo{})
}

func evalFileInto(file, mimetype string, convertNumbers bool, out *os.File) error {
	if mimetype == "" {
		mimetype = string(runner.GetMimeType(file, runner.ContentTypeInfo{}))
	} else {
		mimetype = string(resolveContentType(mimetype))
	}

	if mimetype == "" {
		return fmt.Errorf("Unknown mimetype for file: %s.\n", file)
	}

	w := bufio.NewWriterSize(out, 1e6)
	defer w.Flush()

	return runner.TransformFile(file, runner.ContentTypeInfo{
		Type: mimetype,
	}, convertNumbers, w)
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

	size := int64(-1)
	fi, err := fd.Stat()
	if err == nil {
		size = fi.Size()
	}

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
		// This is intentional to add a newline
		fmt.Println()
		return nil
	}

	var rows []map[string]interface{}
	if size != 0 {
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
	}

	if len(rows) == 1 {
		fmt.Println("(1 row)")
	} else {
		fmt.Printf("(%d rows)\n", len(rows))
	}

	return nil
}

func getFileContentHash(sha1 hash.Hash, fileName string) error {
	file, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(sha1, file)
	return err
}

func getFilesContentHash(files []string) (string, error) {
	sha1 := sha1.New()

	for _, file := range files {
		err := getFileContentHash(sha1, file)
		if err != nil {
			return "", err
		}
	}

	return hex.EncodeToString(sha1.Sum(nil)), nil
}

func importFile(projectId string, file, mimetype string, convertNumbers bool, ec runner.EvalContext) (*runner.PanelInfo, error) {
	panelId := uuid.New().String()
	resultFile := ec.GetPanelResultsFile(projectId, panelId)
	out, err := openTruncate(resultFile)
	if err != nil {
		return nil, err
	}
	defer out.Close()

	if err := evalFileInto(file, mimetype, convertNumbers, out); err != nil {
		return nil, err
	}

	s, err := getShape(resultFile, panelId)
	if err != nil {
		return nil, err
	}

	return &runner.PanelInfo{
		ResultMeta: runner.PanelResult{
			Shape: *s,
		},
		Id:   panelId,
		Name: uuid.New().String(),
	}, nil
}

func runQuery(queryRaw string, project *runner.ProjectState, ec *runner.EvalContext, args *args, files []string) error {
	query := rewriteQuery(queryRaw)
	panel := &runner.PanelInfo{
		Type:    runner.DatabasePanel,
		Content: query,
		Id:      uuid.New().String(),
		Name:    uuid.New().String(),
		DatabasePanelInfo: &runner.DatabasePanelInfo{
			Database: runner.DatabasePanelInfoDatabase{
				ConnectorId: project.Connectors[0].Id,
			},
		},
	}

	err := ec.EvalDatabasePanel(project, 0, panel, nil, args.cacheSettings)
	if err != nil {
		if e, ok := err.(*runner.DSError); ok && e.Name == "NotAnArrayOfObjectsError" {
			rest := "."
			nth, err := strconv.Atoi(e.TargetPanelId)
			if err == nil {
				rest = ": " + files[nth] + "."
			}
			return fmt.Errorf("Input is not an array of objects%s\n", rest)
		}

		return err
	}

	resultFile := ec.GetPanelResultsFile(project.Id, panel.Id)
	return dumpJSONFile(resultFile, args.pretty, args.schema)
}

func repl(project *runner.ProjectState, ec *runner.EvalContext, args *args, files []string) error {
	completer := readline.NewPrefixCompleter(
		readline.PcItem("SELECT"),
		readline.PcItem("FROM"),
		readline.PcItem("WHERE"),
		readline.PcItem("AND"),
		readline.PcItem("OR"),
		readline.PcItem("IN"),
		readline.PcItem("JOIN"),
	)

	filterInput := func(r rune) (rune, bool) {
		switch r {
		// block CtrlZ feature
		case readline.CharCtrlZ:
			return r, false
		}
		return r, true
	}

	historyFile := path.Join(runner.HOME, "dsq_history")
	l, err := readline.NewEx(&readline.Config{
		Prompt:              "dsq> ",
		HistoryFile:         historyFile,
		InterruptPrompt:     "^D",
		EOFPrompt:           "exit",
		HistorySearchFold:   true,
		FuncFilterInputRune: filterInput,
		AutoComplete:        completer,
	})
	if err != nil {
		return err
	}

	defer l.Close()

	for {
		queryRaw, err := l.Readline()
		if err != nil {
			return err
		}

		queryRaw = strings.TrimSpace(queryRaw)
		if queryRaw == "" {
			continue
		}

		if queryRaw == "exit" {
			// prints bye like mysql
			fmt.Println("bye")
			return nil
		}

		err = runQuery(queryRaw, project, ec, args, files)
		if err != nil {
			return err
		}
	}
}

type args struct {
	pipedMimetype  string
	pretty         bool
	schema         bool
	sqlFile        string
	cacheSettings  runner.CacheSettings
	nonFlagArgs    []string
	dumpCacheFile  bool
	isInteractive  bool
	convertNumbers bool
}

func getArgs() (*args, error) {
	args := &args{}
	osArgs := os.Args[1:]
	for i := 0; i < len(osArgs); i++ {
		arg := osArgs[i]
		isLast := i == len(osArgs)-1

		if arg == "--verbose" {
			runner.Verbose = true
			continue
		}

		if arg == "-s" || arg == "--stdin" {
			if isLast {
				return nil, errors.New("Must specify stdin mimetype.")
			}

			args.pipedMimetype = osArgs[i+1]
			i++

			continue
		}

		if arg == "-h" || arg == "--help" {
			log.Println(HELP)
			return nil, nil
		}

		if arg == "-p" || arg == "--pretty" {
			args.pretty = true
			continue
		}

		if arg == "-v" || arg == "--version" {
			log.Println("dsq " + Version)
			return nil, nil
		}

		if arg == "-c" || arg == "--schema" {
			args.schema = true
			continue
		}

		if arg == "-f" || arg == "--file" {
			if isLast {
				return nil, errors.New("Must specify a SQL file.")
			}

			args.sqlFile = osArgs[i+1]
			i++

			continue
		}

		if arg == "--cache" || arg == "-C" {
			args.cacheSettings.Enabled = true
			continue
		}

		if arg == "--cache-file" || arg == "-D" {
			args.dumpCacheFile = true
			args.cacheSettings.Enabled = true
			continue
		}

		if arg == "--interactive" || arg == "-i" {
			args.isInteractive = true
			args.pretty = true
			args.cacheSettings.Enabled = true
			continue
		}

		if arg == "-n" || arg == "--convert-numbers" {
			args.convertNumbers = true
			continue
		}

		args.nonFlagArgs = append(args.nonFlagArgs, arg)
	}

	return args, nil
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

	args, err := getArgs()
	if err != nil {
		return err
	}

	// Some commands exit naturally here. Like -h and -v
	if args == nil {
		return nil
	}

	lastNonFlagArg := ""
	files := args.nonFlagArgs

	// Grab from stdin into local file
	mimetypeOverride := map[string]string{}
	if args.pipedMimetype != "" {
		pipedTmp, err := ioutil.TempFile("", "dsq-stdin")
		if err != nil {
			return err
		}
		defer os.Remove(pipedTmp.Name())

		mimetypeOverride[pipedTmp.Name()] = args.pipedMimetype

		_, err = io.Copy(pipedTmp, os.Stdin)
		if err != nil {
			return err
		}
		pipedTmp.Close()
		files = append([]string{pipedTmp.Name()}, files...)
	}

	// If -f|--file not present, query is the last argument
	if args.sqlFile == "" {
		if len(files) > 1 {
			lastNonFlagArg = files[len(files)-1]
			if strings.Contains(lastNonFlagArg, " ") {
				files = files[:len(files)-1]
			}
		}
	} else {
		// Otherwise read -f|--file as query
		content, err := os.ReadFile(args.sqlFile)
		if err != nil {
			return errors.New("Error opening sql file: " + err.Error())
		}

		lastNonFlagArg = string(content)
		if lastNonFlagArg == "" {
			return errors.New("SQL file is empty.")
		}
	}

	if len(files) == 0 {
		return errors.New("No input files.")
	}

	var projectIdHashOrTmp string
	if args.cacheSettings.Enabled {
		var err error
		projectIdHashOrTmp, err = getFilesContentHash(files)
		if err != nil {
			log.Printf("Error creating hash for cache mode: %v, defaulting to normal mode", err)
			args.cacheSettings.Enabled = false
		}
	} else {
		projectTmp, err := ioutil.TempFile("", "dsq-project")
		if err != nil {
			return err
		}
		defer os.Remove(projectTmp.Name())

		projectIdHashOrTmp = projectTmp.Name()
	}

	project := &runner.ProjectState{
		Id: projectIdHashOrTmp,
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

	// Does no harm in calculating this even if caching is not on. A few places use this path.
	cachedPath := filepath.Join(os.TempDir(), "dsq-cache-"+projectIdHashOrTmp+".db")
	if args.cacheSettings.Enabled {
		info, err := os.Stat(cachedPath)
		args.cacheSettings.CachePresent = err == nil && info.Size() != 0
		if !args.cacheSettings.CachePresent {
			log.Println("Cache invalid, re-import required.")
			os.Remove(cachedPath)
		}
	}

	ec := runner.NewEvalContext(*runner.DefaultSettings, tmpDir)
	// When dumping schema, need to injest even if cache is on.
	if !args.cacheSettings.CachePresent || !args.cacheSettings.Enabled || lastNonFlagArg == "" {
		for _, file := range files {
			panel, err := importFile(project.Id, file, mimetypeOverride[file], args.convertNumbers, ec)
			if err != nil {
				return err
			}
			project.Pages[0].Panels = append(project.Pages[0].Panels, *panel)
		}
	}

	if args.dumpCacheFile {
		fmt.Println(cachedPath)
		return nil
	}

	// No query, just dump transformed file directly out
	if lastNonFlagArg == "" && !args.isInteractive {
		resultFile := ec.GetPanelResultsFile(project.Id, project.Pages[0].Panels[0].Id)
		return dumpJSONFile(resultFile, args.pretty, args.schema)
	}

	connector, err := runner.MakeTmpSQLiteConnector()
	if err != nil {
		return err
	}

	if args.cacheSettings.Enabled {
		connector.DatabaseConnectorInfo.Database.Database = cachedPath
	}
	project.Connectors = append(project.Connectors, *connector)

	if args.isInteractive {
		return repl(project, &ec, args, files)
	}

	return runQuery(lastNonFlagArg, project, &ec, args, files)
}

func main() {
	err := _main()
	if err != nil {
		log.Fatal(err)
	}
}
