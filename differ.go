package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-openapi/loads"
	"github.com/go-swagger/go-swagger/cmd/swagger/commands/diff"
)

func isSameLeastVersion(folder string, newVersionPath string) (bool, *diff.SpecDifferences, error) {
	s1, err := loads.Spec(newVersionPath)

	if err != nil {
		return false, nil, err
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
		return false, nil, err
	}

	if leastVersion == "" {
		return false, nil, nil
	}
	s2, err := loads.Spec(leastVersion)
	if err != nil {
		return false, nil, err
	}

	diffReport, err := diff.Compare(s1.Spec(), s2.Spec())
	if err != nil {
		return false, nil, err
	}

	input, err, _ := diffReport.ReportAllDiffs(true)
	var result []byte
	input.Read(result)
	fmt.Println(string(result))
	if string(result) == "[]\n" {
		return true, nil, nil
	}
	if err != nil {
		return false, nil, err
	}

	return false, &diffReport, nil
}
