package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_missingFileReturnsDefaults(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Source != "" {
		t.Errorf("expected empty Source for missing file, got %q", cfg.Source)
	}
	if cfg.TagPrefix != Defaults().TagPrefix {
		t.Errorf("TagPrefix = %q, want default %q", cfg.TagPrefix, Defaults().TagPrefix)
	}
	if len(cfg.Branches) != len(Defaults().Branches) {
		t.Errorf("Branches len = %d, want %d", len(cfg.Branches), len(Defaults().Branches))
	}
}

func TestLoad_emptyFileMatchesDefaults(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Source == "" {
		t.Errorf("expected Source to be set for present file")
	}
	if cfg.TagPrefix != Defaults().TagPrefix {
		t.Errorf("empty file should yield defaults; TagPrefix = %q", cfg.TagPrefix)
	}
}

func TestLoad_partialFileMerges(t *testing.T) {
	dir := t.TempDir()
	yaml := `
tag-prefix: "^release-(.+)$"
on-shallow: warn
`
	if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TagPrefix != "^release-(.+)$" {
		t.Errorf("TagPrefix = %q, want override", cfg.TagPrefix)
	}
	if cfg.OnShallow != "warn" {
		t.Errorf("OnShallow = %q, want override", cfg.OnShallow)
	}
	// Untouched keys should retain defaults.
	if cfg.Mode != Defaults().Mode {
		t.Errorf("Mode = %q, want default %q", cfg.Mode, Defaults().Mode)
	}
	if len(cfg.Branches) != len(Defaults().Branches) {
		t.Errorf("Branches should fall back to defaults; got len %d", len(cfg.Branches))
	}
}

func TestLoad_branchesReplaceDefaults(t *testing.T) {
	dir := t.TempDir()
	yaml := `
branches:
  - name: trunk
    regex: '^trunk$'
    is-mainline: true
    increment: patch
`
	if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Branches) != 1 {
		t.Fatalf("Branches len = %d, want 1 (full replacement)", len(cfg.Branches))
	}
	if cfg.Branches[0].Name != "trunk" {
		t.Errorf("Branches[0].Name = %q, want trunk", cfg.Branches[0].Name)
	}
	if !cfg.Branches[0].IsMainline {
		t.Errorf("Branches[0].IsMainline should be true")
	}
}

func TestLoad_dirtySuffixCanBeExplicitlyFalse(t *testing.T) {
	dir := t.TempDir()
	yaml := "dirty-suffix: false\n"
	if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DirtySuffix == nil {
		t.Fatal("DirtySuffix should be set (pointer)")
	}
	if *cfg.DirtySuffix != false {
		t.Errorf("DirtySuffix = %v, want false", *cfg.DirtySuffix)
	}
}

func TestLoad_malformedYAMLErrors(t *testing.T) {
	dir := t.TempDir()
	yaml := "tag-prefix: [this is not a string\n"
	if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Error("expected parse error, got nil")
	}
}

func TestLoad_walksUpToFindConfig(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ConfigFileName), []byte("on-shallow: ignore\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(nested)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OnShallow != "ignore" {
		t.Errorf("OnShallow = %q, expected to be picked up from parent dir", cfg.OnShallow)
	}
}
