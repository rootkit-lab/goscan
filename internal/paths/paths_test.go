package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestModeDevWithRepoRootEnv(t *testing.T) {
	t.Setenv("GOSCAN_MODE", "")
	t.Setenv("GOSCAN_REPO_ROOT", t.TempDir())
	// temp dir is not a repo — Mode still dev because GOSCAN_REPO_ROOT set
	if got := Mode(); got != ModeDev {
		t.Fatalf("Mode() = %q want dev", got)
	}
}

func TestModeProdExplicit(t *testing.T) {
	t.Setenv("GOSCAN_MODE", "prod")
	t.Setenv("GOSCAN_REPO_ROOT", "")
	if got := Mode(); got != ModeProd {
		t.Fatalf("Mode() = %q want prod", got)
	}
}

func TestDataRootOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOSCAN_DATA_DIR", dir)
	t.Setenv("GOSCAN_MODE", "prod")
	got, err := DataRoot()
	if err != nil {
		t.Fatal(err)
	}
	if got != dir {
		t.Fatalf("DataRoot() = %q want %q", got, dir)
	}
}

func TestDefaultDBPathUsesDataRoot(t *testing.T) {
	root := filepath.Join("tmp", "data")
	want := filepath.Join(root, "dominios.db")
	if got := DefaultDBPath(root); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestIsDevRepoPath(t *testing.T) {
	wd, _ := os.Getwd()
	root := filepath.Clean(filepath.Join(wd, "..", ".."))
	if !IsDevRepoPath(root) {
		t.Fatalf("expected dev repo path %s", root)
	}
	if IsDevRepoPath(t.TempDir()) {
		t.Fatal("temp dir should not be dev repo")
	}
}

func TestProdIgnoresDevSettings(t *testing.T) {
	wd, _ := os.Getwd()
	root := filepath.Clean(filepath.Join(wd, "..", ".."))
	t.Setenv("GOSCAN_MODE", "prod")
	t.Setenv("GOSCAN_DATA_DIR", "")
	t.Setenv("GOSCAN_REPO_ROOT", "")

	// Simulate bad settings via temp config
	tmp := t.TempDir()
	cfg := filepath.Join(tmp, "goscan")
	if err := os.MkdirAll(cfg, 0755); err != nil {
		t.Fatal(err)
	}
	body := "data_dir: " + root + "\nscan_dir: " + filepath.Join(root, "files") + "\n"
	if err := os.WriteFile(filepath.Join(cfg, "settings.yaml"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", tmp)

	got, err := DataRoot()
	if err != nil {
		t.Fatal(err)
	}
	wantBase, _ := userDataHome()
	want := filepath.Join(wantBase, "goscan", "data")
	if got != want {
		t.Fatalf("DataRoot() = %q want %q (dev settings ignored)", got, want)
	}
}
