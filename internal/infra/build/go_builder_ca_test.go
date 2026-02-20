package build

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/poruru-code/esb-cli/internal/constants"
	"github.com/poruru-code/esb-cli/internal/infra/envutil"
	"github.com/poruru-code/esb-cli/internal/meta"
)

func TestResolveRootCAPathPrefersHostCACertPath(t *testing.T) {
	t.Setenv("ENV_PREFIX", meta.EnvPrefix)

	caPath := filepath.Join(t.TempDir(), "custom-root-ca.crt")
	writeTestFile(t, caPath, "host-ca")

	caKey, err := envutil.HostEnvKey(constants.HostSuffixCACertPath)
	if err != nil {
		t.Fatal(err)
	}
	certDirKey, err := envutil.HostEnvKey(constants.HostSuffixCertDir)
	if err != nil {
		t.Fatal(err)
	}

	certDir := t.TempDir()
	writeTestFile(t, filepath.Join(certDir, meta.RootCACertFilename), "cert-dir-ca")

	t.Setenv(caKey, caPath)
	t.Setenv(certDirKey, certDir)
	t.Setenv("CAROOT", t.TempDir())

	got, err := resolveRootCAPath()
	if err != nil {
		t.Fatalf("resolve root CA path: %v", err)
	}
	if got != caPath {
		t.Fatalf("expected %q, got %q", caPath, got)
	}
}

func TestResolveRootCAPathUsesCertDirWhenCACertPathMissing(t *testing.T) {
	t.Setenv("ENV_PREFIX", meta.EnvPrefix)

	caKey, err := envutil.HostEnvKey(constants.HostSuffixCACertPath)
	if err != nil {
		t.Fatal(err)
	}
	certDirKey, err := envutil.HostEnvKey(constants.HostSuffixCertDir)
	if err != nil {
		t.Fatal(err)
	}

	certDir := t.TempDir()
	want := filepath.Join(certDir, meta.RootCACertFilename)
	writeTestFile(t, want, "cert-dir-ca")

	t.Setenv(caKey, "")
	t.Setenv(certDirKey, certDir)
	t.Setenv("CAROOT", t.TempDir())

	got, err := resolveRootCAPath()
	if err != nil {
		t.Fatalf("resolve root CA path: %v", err)
	}
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestResolveRootCAPathUsesCAROOTWhenHostEnvMissing(t *testing.T) {
	t.Setenv("ENV_PREFIX", meta.EnvPrefix)

	caKey, err := envutil.HostEnvKey(constants.HostSuffixCACertPath)
	if err != nil {
		t.Fatal(err)
	}
	certDirKey, err := envutil.HostEnvKey(constants.HostSuffixCertDir)
	if err != nil {
		t.Fatal(err)
	}

	caRoot := t.TempDir()
	want := filepath.Join(caRoot, meta.RootCACertFilename)
	writeTestFile(t, want, "caroot-ca")

	t.Setenv(caKey, "")
	t.Setenv(certDirKey, "")
	t.Setenv("CAROOT", caRoot)

	got, err := resolveRootCAPath()
	if err != nil {
		t.Fatalf("resolve root CA path: %v", err)
	}
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestResolveRootCAPathFallsBackToRepoRoot(t *testing.T) {
	t.Setenv("ENV_PREFIX", meta.EnvPrefix)

	caKey, err := envutil.HostEnvKey(constants.HostSuffixCACertPath)
	if err != nil {
		t.Fatal(err)
	}
	certDirKey, err := envutil.HostEnvKey(constants.HostSuffixCertDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(caKey, "")
	t.Setenv(certDirKey, "")
	t.Setenv("CAROOT", "")

	repoRoot := t.TempDir()
	writeComposeFiles(t, repoRoot, "docker-compose.containerd.yml")
	want := filepath.Join(repoRoot, meta.HomeDir, "certs", meta.RootCACertFilename)
	writeTestFile(t, want, "repo-root-ca")

	nested := filepath.Join(repoRoot, "nested", "dir")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested dir: %v", err)
	}
	setWorkingDir(t, nested)

	got, err := resolveRootCAPath()
	if err != nil {
		t.Fatalf("resolve root CA path: %v", err)
	}
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestResolveRootCAPathReturnsHelpfulErrorWhenMissing(t *testing.T) {
	t.Setenv("ENV_PREFIX", meta.EnvPrefix)

	caKey, err := envutil.HostEnvKey(constants.HostSuffixCACertPath)
	if err != nil {
		t.Fatal(err)
	}

	missing := filepath.Join(t.TempDir(), "missing-root-ca.crt")
	t.Setenv(caKey, missing)

	_, err = resolveRootCAPath()
	if err == nil {
		t.Fatalf("expected error for missing root CA")
	}
	if !strings.Contains(err.Error(), "root CA not found at "+missing) {
		t.Fatalf("expected missing path in error, got %q", err)
	}
	if !strings.Contains(err.Error(), "mise run setup") {
		t.Fatalf("expected setup hint in error, got %q", err)
	}
}

func TestEnsureRootCAPathRejectsDirectory(t *testing.T) {
	dir := t.TempDir()

	_, err := ensureRootCAPath(dir)
	if err == nil {
		t.Fatalf("expected directory error")
	}
	if !strings.Contains(err.Error(), "root CA path is a directory") {
		t.Fatalf("expected directory error, got %q", err)
	}
}

func TestResolveRootCAFingerprint(t *testing.T) {
	t.Setenv("ENV_PREFIX", meta.EnvPrefix)

	caKey, err := envutil.HostEnvKey(constants.HostSuffixCACertPath)
	if err != nil {
		t.Fatal(err)
	}
	caPath := filepath.Join(t.TempDir(), "rootCA.crt")
	const body = "fingerprint-source"
	writeTestFile(t, caPath, body)
	t.Setenv(caKey, caPath)

	got, err := resolveRootCAFingerprint()
	if err != nil {
		t.Fatalf("resolve root CA fingerprint: %v", err)
	}

	sum := sha256.Sum256([]byte(body))
	want := hex.EncodeToString(sum[:4])
	if got != want {
		t.Fatalf("expected fingerprint %q, got %q", want, got)
	}
}

func TestExpandHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "plain path", in: "/tmp/cert.crt", want: "/tmp/cert.crt"},
		{name: "home only", in: "~", want: home},
		{name: "home child", in: "~/certs/rootCA.crt", want: filepath.Join(home, "certs", "rootCA.crt")},
		{name: "tilde with username stays as-is", in: "~other/cert.crt", want: "~other/cert.crt"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := expandHome(tc.in)
			if got != tc.want {
				t.Fatalf("expandHome(%q): expected %q, got %q", tc.in, tc.want, got)
			}
		})
	}
}
