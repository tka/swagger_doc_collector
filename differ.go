package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/tufin/oasdiff/diff"
	"github.com/tufin/oasdiff/load"
	"github.com/tufin/oasdiff/report"
)

func diffConfig() *diff.Config {
	return diff.NewConfig()
}
func newApiLoader() *openapi3.Loader {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	return loader
}
func isSameLeastVersion(folder string, apiBody []byte) (bool, error) {
	loader := openapi3.NewLoader()
	s1, err := loader.LoadFromData(apiBody)
	if err != nil {
		return false, err
	}

	var leastVersion string
	err = filepath.Walk(folder, func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			return nil
		}
		leastVersion = path
		return nil
	})
	if err != nil {
		return false, err
	}

	if leastVersion == "" {
		return false, nil
	}
	s2, err := load.From(loader, leastVersion)
	if err != nil {
		return false, err
	}

	diffReport, err := diff.Get(diffConfig(), s1, s2)

	if err != nil {
		return false, err
	}
	if diffReport.Empty() {
		return true, nil
	}
	return false, nil
}
func removeDuplicates(folder string) error {
	versions := make([]string, 0)
	err := filepath.Walk(folder, func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			return nil
		}
		fmt.Println(path)
		versions = append(versions, path)
		return nil
	})
	if err != nil {
		return err
	}
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	s1, err := load.From(loader, versions[0])
	if err != nil {
		return err
	}
	s2, err := load.From(loader, versions[1])
	if err != nil {
		return err
	}
	diffReport, err := diff.Get(diffConfig(), s1, s2)
	fmt.Println("diffReport", diffReport)
	if err != nil {
		return err
	}
	html, err := report.GetHTMLReportAsString(diffReport)
	if err != nil {
		return err
	}
	fmt.Println(html)
	return nil
}
