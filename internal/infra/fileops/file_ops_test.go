package fileops

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyDirVariants(t *testing.T) {
	tests := []struct {
		name string
		run  func(src, dst string) error
	}{
		{
			name: "copy",
			run:  CopyDir,
		},
		{
			name: "link or copy",
			run:  CopyDirLinkOrCopy,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			src := filepath.Join(t.TempDir(), "src")
			dst := filepath.Join(t.TempDir(), "dst")
			writeFixtureTree(t, src)

			if err := tc.run(src, dst); err != nil {
				t.Fatalf("copy variant failed: %v", err)
			}
			assertFixtureTree(t, dst)
		})
	}
}

func writeFixtureTree(t *testing.T, root string) {
	t.Helper()
	writeFixtureFile(t, filepath.Join(root, "root.txt"), "root", 0o640)
	writeFixtureFile(t, filepath.Join(root, "nested", "a.txt"), "alpha", 0o600)
	writeFixtureFile(t, filepath.Join(root, "nested", "deeper", "b.txt"), "beta", 0o644)
}

func writeFixtureFile(t *testing.T, path, content string, perm os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), perm); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func assertFixtureTree(t *testing.T, root string) {
	t.Helper()
	assertFile(t, filepath.Join(root, "root.txt"), "root", 0o640)
	assertFile(t, filepath.Join(root, "nested", "a.txt"), "alpha", 0o600)
	assertFile(t, filepath.Join(root, "nested", "deeper", "b.txt"), "beta", 0o644)
}

func assertFile(t *testing.T, path, wantContent string, wantPerm os.FileMode) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(data) != wantContent {
		t.Fatalf("content mismatch for %s: got %q want %q", path, string(data), wantContent)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != wantPerm {
		t.Fatalf("perm mismatch for %s: got %o want %o", path, got, wantPerm)
	}
}
