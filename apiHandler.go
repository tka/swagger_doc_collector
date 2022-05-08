package main

import (
	"os"
	"path"
	"path/filepath"
	"reflect"

	"github.com/go-openapi/loads"
	"github.com/go-swagger/go-swagger/cmd/swagger/commands/diff"
	"github.com/labstack/echo/v4"
	"gopkg.in/yaml.v2"
)

func listDocsHandler(e echo.Context) error {
	var result = make(map[string][]string)
	for _, doc := range config.Docs {
		filepath.Walk(doc.Path(), func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			} else {

				result[doc.Name] = append(result[doc.Name], filepath.Base(path))
			}
			return nil
		})
	}

	return e.JSON(200, map[string]interface{}{
		"docs": result,
	})
}

func docDiffDetailHandler(e echo.Context) error {
	docName := e.QueryParam("name")
	docVersion1 := e.QueryParam("version1")
	docVersion2 := e.QueryParam("version2")

	var doc Doc
	for _, d := range config.Docs {
		if docName == d.Name {
			doc = d
			break
		}
	}

	s1, err := loads.Spec(path.Join(doc.Path(), filepath.Base(docVersion1)))
	if err != nil {
		return e.JSON(500, "load version 1 error "+err.Error())
	}

	s2, err := loads.Spec(path.Join(doc.Path(), filepath.Base(docVersion2)))
	if err != nil {
		return e.JSON(500, "load version 2 error "+err.Error())
	}
	diffs, err := diff.Compare(s1.Spec(), s2.Spec())

	if err != nil {
		return e.JSON(500, "get diff error "+err.Error())

	}
	input, err, _ := diffs.ReportAllDiffs(false)
	var result = make([]byte, 1024*20) // read 20k diff
	input.Read(result)

	if err != nil {
		return e.JSON(500, "gen report error "+err.Error())
	}
	return e.JSON(200, map[string]string{"diff": string(result)})
}

func printYAML(output interface{}) (string, error) {
	if reflect.ValueOf(output).IsNil() {
		return "", nil
	}

	bytes, err := yaml.Marshal(output)
	if err != nil {
		return "", err
	}
	return string(bytes), nil

}
