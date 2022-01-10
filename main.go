package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/multiprocessio/datastation/runner"

	"github.com/google/uuid"
)

func isinpipe() bool {
	fi, _ := os.Stdin.Stat()
	if fi == nil {
		return false
	}

	// This comes back incorrect in automated environments like Github Actions.
	return !(fi.Mode()&os.ModeNamedPipe == 0)
}

func resolveContentType(fileExtensionOrContentType string) string {
	if strings.Contains(fileExtensionOrContentType, "/") {
		return fileExtensionOrContentType
	}

	return runner.GetMimeType("x."+fileExtensionOrContentType, runner.ContentTypeInfo{})
}

var firstNonFlagArg = ""

func getResult(res interface{}) error {
	out := bytes.NewBuffer(nil)
	arg := firstNonFlagArg

	mimetype := resolveContentType(arg)

	cti := runner.ContentTypeInfo{Type: mimetype}

	// isinpipe() is sometimes incorrect. If the first arg
	// is a file, fall back to acting like this isn't in a
	// pipe.
	runAsFile := !isinpipe()
	if !runAsFile && cti.Type == arg {
		if _, err := os.Stat(arg); err == nil {
			runAsFile = true
		}
	}

	if !runAsFile {
		if mimetype == "" {
			return fmt.Errorf(`First argument when used in a pipe should be file extension or content type. e.g. 'cat test.csv | dsq csv "SELECT * FROM {}"'`)
		}

		err := runner.TransformReader(os.Stdin, "", cti, out)
		if err != nil {
			return err
		}
	} else {
		if arg == "" {
			return fmt.Errorf(`First argument when not used in a pipe should be a file. e.g. 'dsq test.csv "SELECT COUNT(1) FROM {}"'`)
		}

		err := runner.TransformFile(arg, runner.ContentTypeInfo{}, out)
		if err != nil {
			return err
		}
	}

	decoder := json.NewDecoder(out)
	return decoder.Decode(res)
}

func main() {
	log.SetFlags(0)
	runner.Verbose = false
	inputTable := "{}"
	var nonFlagArgs []string
	for i, arg := range os.Args[1:] {
		if arg == "-i" || arg == "--input-table-alias" {
			if i > len(os.Args)-2 {
				log.Fatal(`Expected input table alias after flag. e.g. 'dsq -i XX names.csv "SELECT * FROM XX"'`)
			}

			inputTable = os.Args[i+1]
			continue
		}

		if arg == "-v" || arg == "--verbose" {
			runner.Verbose = true
			continue
		}

		if arg == "-h" || arg == "--help" {
			log.Println("See the README on Github for details.\n\nhttps://github.com/multiprocessio/datastation/blob/main/runner/cmd/dsq/README.md")
			return
		}

		nonFlagArgs = append(nonFlagArgs, arg)
	}

	if len(nonFlagArgs) > 0 {
		firstNonFlagArg = nonFlagArgs[0]
	}

	lastNonFlagArg := ""
	if len(nonFlagArgs) > 1 {
		lastNonFlagArg = nonFlagArgs[len(nonFlagArgs)-1]
	}

	var res []map[string]interface{}
	err := getResult(&res)
	if err != nil {
		log.Fatal(err)
	}

	if lastNonFlagArg == "" {
		encoder := json.NewEncoder(os.Stdout)
		err := encoder.Encode(res)
		if err != nil {
			log.Fatal(err)
		}

		return
	}

	sampleSize := 50
	shape, err := runner.GetArrayShape(firstNonFlagArg, res, sampleSize)
	if err != nil {
		log.Fatal(err)
	}

	p0 := runner.PanelInfo{
		ResultMeta: runner.PanelResult{
			Shape: *shape,
		},
		Id:   uuid.New().String(),
		Name: uuid.New().String(),
	}

	projectTmp, err := ioutil.TempFile("", "dsq-project")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(projectTmp.Name())
	project := &runner.ProjectState{
		Id: projectTmp.Name(),
		Pages: []runner.ProjectPage{
			{
				Panels: []runner.PanelInfo{p0},
			},
		},
	}
	connector, tmp, err := runner.MakeTmpSQLiteConnector()
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	project.Connectors = append(project.Connectors, *connector)

	query := lastNonFlagArg
	query = strings.ReplaceAll(query, inputTable, "DM_getPanel(0)")
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

	panelResultLoader := func(_, _ string, out interface{}) error {
		r := out.(*[]map[string]interface{})
		*r = res
		return nil
	}
	err = runner.EvalDatabasePanel(project, 0, panel, panelResultLoader)
	if err != nil {
		log.Fatal(err)
	}

	// Dump the result to stdout
	fd, err := os.Open(runner.GetPanelResultsFile(project.Id, panel.Id))
	if err != nil {
		log.Fatalf("Could not open results file: %s", err)
	}

	io.Copy(os.Stdout, fd)
}
