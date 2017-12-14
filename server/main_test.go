package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runTests(m *testing.M) int {
	defer func() {
		// remove any log files in the current direct that are as a result of testing
		currentDirectory, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to get current directory: %s\n", err)
			return
		}

		err = filepath.Walk(
			currentDirectory,
			func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if info != nil && info.Mode().IsRegular() && strings.HasSuffix(info.Name(), ".log") {
					if err := os.Remove(path); err != nil {
						fmt.Fprintf(os.Stderr, "Unable to remove log file after test: %s\n", err)
					}
				}

				return nil
			},
		)
	}()

	return m.Run()
}

func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}
