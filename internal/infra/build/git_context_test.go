// Where: cli/internal/infra/build/git_context_test.go
// What: Tests for git context resolution env.
// Why: Ensure worktree and standard clones resolve build contexts correctly.
package build

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type gitRunnerStub struct {
	outputs map[string]string
}

func (g gitRunnerStub) RunOutput(_ context.Context, _, name string, args ...string) ([]byte, error) {
	key := name + " " + strings.Join(args, " ")
	if out, ok := g.outputs[key]; ok {
		return []byte(out), nil
	}
	return nil, nil
}

func TestResolveGitContextStandardRepo(t *testing.T) {
	root := t.TempDir()
	gitDir := filepath.Join(root, ".git")
	if err := writeTestFileWithDirs(gitDir, "HEAD", "ref: refs/heads/main\n"); err != nil {
		t.Fatal(err)
	}
	if err := writeTestFileWithDirs(gitDir, "objects/placeholder", ""); err != nil {
		t.Fatal(err)
	}

	runner := gitRunnerStub{
		outputs: map[string]string{
			"git rev-parse --show-toplevel":  root,
			"git rev-parse --git-dir":        ".git",
			"git rev-parse --git-common-dir": ".git",
		},
	}

	ctx, err := resolveGitContext(t.Context(), runner, root)
	if err != nil {
		t.Fatalf("resolveGitContext: %v", err)
	}
	if ctx.GitDir != gitDir {
		t.Fatalf("unexpected gitdir: %s", ctx.GitDir)
	}
	if ctx.GitCommon != gitDir {
		t.Fatalf("unexpected git common: %s", ctx.GitCommon)
	}
}

func TestResolveGitContextWorktreeGitDirFile(t *testing.T) {
	root := filepath.Join(t.TempDir(), "worktree")
	gitCommon := filepath.Join(filepath.Dir(root), ".git")
	gitDir := filepath.Join(gitCommon, "worktrees", "w1")

	if err := writeTestFileWithDirs(gitDir, "HEAD", "ref: refs/heads/main\n"); err != nil {
		t.Fatal(err)
	}
	if err := writeTestFileWithDirs(gitCommon, "objects/placeholder", ""); err != nil {
		t.Fatal(err)
	}

	if err := writeTestFileWithDirs(root, ".git", "gitdir: ../.git/worktrees/w1\n"); err != nil {
		t.Fatal(err)
	}

	runner := gitRunnerStub{
		outputs: map[string]string{
			"git rev-parse --show-toplevel":  root,
			"git rev-parse --git-dir":        ".git",
			"git rev-parse --git-common-dir": "../.git",
		},
	}

	ctx, err := resolveGitContext(t.Context(), runner, root)
	if err != nil {
		t.Fatalf("resolveGitContext: %v", err)
	}
	if ctx.GitDir != gitDir {
		t.Fatalf("unexpected gitdir: %s", ctx.GitDir)
	}
	if ctx.GitCommon != gitCommon {
		t.Fatalf("unexpected git common: %s", ctx.GitCommon)
	}
}

func TestResolveGitContextMissingRequiredPaths(t *testing.T) {
	tests := []struct {
		name         string
		headContent  string
		writeObjects bool
	}{
		{
			name:         "missing HEAD",
			headContent:  "",
			writeObjects: true,
		},
		{
			name:         "missing objects",
			headContent:  "ref: refs/heads/main\n",
			writeObjects: false,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			gitDir := filepath.Join(root, ".git")
			if tc.headContent != "" {
				if err := writeTestFileWithDirs(gitDir, "HEAD", tc.headContent); err != nil {
					t.Fatal(err)
				}
			}
			if tc.writeObjects {
				if err := writeTestFileWithDirs(gitDir, "objects/placeholder", ""); err != nil {
					t.Fatal(err)
				}
			}

			runner := gitRunnerStub{
				outputs: map[string]string{
					"git rev-parse --show-toplevel":  root,
					"git rev-parse --git-dir":        ".git",
					"git rev-parse --git-common-dir": ".git",
				},
			}

			_, err := resolveGitContext(t.Context(), runner, root)
			if err == nil {
				t.Fatalf("expected error for %s", tc.name)
			}
		})
	}
}

func writeTestFileWithDirs(root, relPath, content string) error {
	path := filepath.Join(root, relPath)
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o600)
}
