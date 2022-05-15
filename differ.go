package main

import (
	"os"
	"path/filepath"

	"github.com/go-openapi/loads"
	"github.com/go-swagger/go-swagger/cmd/swagger/commands/diff"
)

func isSameLeastVersion(folder string, newSpec *loads.Document) (bool, *diff.SpecDifferences, error) {

	var leastVersion string
	err := filepath.Walk(folder, func(path string, f os.FileInfo, err error) error {
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
	oldSpec, err := loads.Spec(leastVersion)
	if err != nil {
		return false, nil, err
	}

	diffReport, err := diff.Compare(oldSpec.Spec(), newSpec.Spec())
	if err != nil {
		return false, nil, err
	}

	numDiffs := len(diffReport)
	if numDiffs == 0 {
		return true, nil, err
	}
	if err != nil {
		return false, nil, err
	}

	return false, &diffReport, nil
}
