package build

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestBuildxHint(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{name: "empty output", output: "", want: ""},
		{name: "non ecr output", output: "dial tcp timeout", want: ""},
		{
			name:   "ecr 403 output includes hint",
			output: "failed to fetch from public.ecr.aws with 403 forbidden",
			want:   "Hint: public.ecr.aws denied the request.",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := buildxHint(tc.output)
			if !strings.Contains(got, tc.want) {
				t.Fatalf("expected %q to contain %q", got, tc.want)
			}
		})
	}
}

func TestRunBakeCommandVerboseUsesRun(t *testing.T) {
	runner := &bakeExecRunner{}

	err := runBakeCommand(context.Background(), runner, "/repo", []string{"buildx", "bake"}, true)
	if err != nil {
		t.Fatalf("run bake command: %v", err)
	}
	if runner.runCalls != 1 {
		t.Fatalf("expected Run to be called once, got %d", runner.runCalls)
	}
	if runner.runOutputCalls != 0 {
		t.Fatalf("RunOutput must not be called in verbose mode, got %d", runner.runOutputCalls)
	}
}

func TestRunBakeCommandVerbosePropagatesRunError(t *testing.T) {
	runErr := errors.New("run failed")
	runner := &bakeExecRunner{runErr: runErr}

	err := runBakeCommand(context.Background(), runner, "/repo", []string{"buildx", "bake"}, true)
	if !errors.Is(err, runErr) {
		t.Fatalf("expected run error to propagate, got %v", err)
	}
}

func TestRunBakeCommandNonVerboseSuccess(t *testing.T) {
	runner := &bakeExecRunner{output: []byte("ok")}

	err := runBakeCommand(context.Background(), runner, "/repo", []string{"buildx", "bake"}, false)
	if err != nil {
		t.Fatalf("run bake command: %v", err)
	}
	if runner.runCalls != 0 {
		t.Fatalf("Run must not be called in non-verbose mode, got %d", runner.runCalls)
	}
	if runner.runOutputCalls != 1 {
		t.Fatalf("expected RunOutput to be called once, got %d", runner.runOutputCalls)
	}
}

func TestRunBakeCommandNonVerboseReturnsRawErrorOnEmptyOutput(t *testing.T) {
	runErr := errors.New("bake failed")
	runner := &bakeExecRunner{
		output:    []byte("   \n\t"),
		outputErr: runErr,
	}

	err := runBakeCommand(context.Background(), runner, "/repo", []string{"buildx", "bake"}, false)
	if !errors.Is(err, runErr) {
		t.Fatalf("expected raw error for empty output, got %v", err)
	}
}

func TestRunBakeCommandNonVerboseWrapsErrorWithOutput(t *testing.T) {
	runErr := errors.New("bake failed")
	runner := &bakeExecRunner{
		output:    []byte("failed to solve: network error"),
		outputErr: runErr,
	}

	err := runBakeCommand(context.Background(), runner, "/repo", []string{"buildx", "bake"}, false)
	if err == nil {
		t.Fatalf("expected wrapped error")
	}
	if !errors.Is(err, runErr) {
		t.Fatalf("expected wrapped original error, got %v", err)
	}
	if !strings.Contains(err.Error(), "buildx bake failed:") {
		t.Fatalf("expected bake failure prefix, got %q", err)
	}
	if !strings.Contains(err.Error(), "failed to solve: network error") {
		t.Fatalf("expected output message in error, got %q", err)
	}
	if strings.Contains(err.Error(), "Hint: public.ecr.aws denied the request.") {
		t.Fatalf("did not expect ECR hint, got %q", err)
	}
}

func TestRunBakeCommandNonVerboseAddsECRHint(t *testing.T) {
	runErr := errors.New("bake failed")
	runner := &bakeExecRunner{
		output:    []byte("ERROR: denied: 403 from public.ecr.aws"),
		outputErr: runErr,
	}

	err := runBakeCommand(context.Background(), runner, "/repo", []string{"buildx", "bake"}, false)
	if err == nil {
		t.Fatalf("expected wrapped error")
	}
	if !errors.Is(err, runErr) {
		t.Fatalf("expected wrapped original error, got %v", err)
	}
	if !strings.Contains(err.Error(), "Hint: public.ecr.aws denied the request.") {
		t.Fatalf("expected ECR hint in error, got %q", err)
	}
}

type bakeExecRunner struct {
	runErr         error
	output         []byte
	outputErr      error
	runCalls       int
	runOutputCalls int
}

func (r *bakeExecRunner) Run(_ context.Context, _, _ string, _ ...string) error {
	r.runCalls++
	return r.runErr
}

func (r *bakeExecRunner) RunOutput(_ context.Context, _, _ string, _ ...string) ([]byte, error) {
	r.runOutputCalls++
	return r.output, r.outputErr
}

func (r *bakeExecRunner) RunQuiet(_ context.Context, _, _ string, _ ...string) error {
	return nil
}
