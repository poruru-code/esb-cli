package command

import (
	"os"
	"testing"
)

func setWorkingDir(t *testing.T, dir string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(prev); err != nil {
			t.Fatalf("restore cwd %s: %v", prev, err)
		}
	})
}

func recordSelectCall(calls *[]imageRuntimeSelectCall, title string, options []string) {
	*calls = append(*calls, imageRuntimeSelectCall{
		title:   title,
		options: append([]string{}, options...),
	})
}

func popQueuedSelection(selectValues *[]string, emptyErr error) (string, error) {
	if len(*selectValues) == 0 {
		return "", emptyErr
	}
	value := (*selectValues)[0]
	*selectValues = (*selectValues)[1:]
	return value, nil
}
