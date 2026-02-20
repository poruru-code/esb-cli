package command

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

var (
	errTestError      = errors.New("test error")
	errSomeOtherError = errors.New("some other error")
)

func TestExitWithError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var buf bytes.Buffer
	code := exitWithError(&buf, errTestError)

	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	want := "✗ test error\n"
	if got := buf.String(); got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}

func TestHandleParseError_GenericError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var buf bytes.Buffer
	code := handleParseError([]string{"build"}, errSomeOtherError, Dependencies{}, &buf)

	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
	output := buf.String()
	if !contains(output, "✗ some other error") {
		t.Errorf("expected error to be printed: %s", output)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
