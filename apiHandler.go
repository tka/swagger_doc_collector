package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
	"github.com/tufin/oasdiff/diff"
	"github.com/tufin/oasdiff/load"
	"github.com/tufin/oasdiff/report"
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

	loader := openapi3.NewLoader()
	s1, err := load.From(loader, path.Join(doc.Path(), filepath.Base(docVersion1)))
	if err != nil {
		return e.JSON(500, "load version 1 error "+err.Error())
	}
	s2, err := load.From(loader, path.Join(doc.Path(), filepath.Base(docVersion2)))
	if err != nil {
		return e.JSON(500, "load version 2 error "+err.Error())
	}

	diffReport, err := diff.Get(diffConfig(), s1, s2)
	if err != nil {
		return e.JSON(500, "get diff error "+err.Error())

	}
	html, err := report.GetHTMLReportAsString(diffReport)
	if err != nil {
		return e.JSON(500, "gen report error "+err.Error())
	}
	fmt.Println(html)
	return e.JSON(200, map[string]string{"diff": html})
}
