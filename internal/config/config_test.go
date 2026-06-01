package config

import (
	"os"
	"path/filepath"
	"strings"
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
	if cfg.InitialVersion != Defaults().InitialVersion {
		t.Errorf("InitialVersion = %q, want default %q", cfg.InitialVersion, Defaults().InitialVersion)
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

func TestDefaults_passValidation(t *testing.T) {
	// Defaults() are hard-coded; if validate() rejects them we've drifted
	// between the validator and the defaults table. Important invariant.
	d := Defaults()
	if err := validate(d); err != nil {
		t.Errorf("Defaults() failed validation: %v", err)
	}
}

func TestLoad_rejectsUnknownKey(t *testing.T) {
	dir := t.TempDir()
	yaml := "tag-prefex: \"^v?(.+)$\"\n"
	if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for unknown key, got nil")
	}
	if !strings.Contains(err.Error(), "tag-prefex") {
		t.Errorf("error should mention the offending key, got: %v", err)
	}
}

func TestLoad_validationErrors(t *testing.T) {
	cases := []struct {
		name      string
		yaml      string
		wantInErr string
	}{
		{
			name:      "invalid on-shallow",
			yaml:      "on-shallow: maybe\n",
			wantInErr: "on-shallow",
		},
		{
			name:      "invalid bump-strategy",
			yaml:      "bump-strategy: vibes\n",
			wantInErr: "bump-strategy",
		},
		{
			name:      "invalid tag-prefix regex",
			yaml:      `tag-prefix: "([unclosed"` + "\n",
			wantInErr: "tag-prefix",
		},
		{
			name: "branch missing regex",
			yaml: `branches:
  - name: x
    increment: patch
`,
			wantInErr: "branches[0].regex",
		},
		{
			name: "branch invalid regex",
			yaml: `branches:
  - name: x
    regex: "([unclosed"
    increment: patch
`,
			wantInErr: "branches[0].regex",
		},
		{
			name: "branch invalid increment",
			yaml: `branches:
  - name: x
    regex: ".*"
    increment: epic
`,
			wantInErr: "branches[0].increment",
		},
		{
			name: "branch invalid bump-strategy",
			yaml: `branches:
  - name: x
    regex: ".*"
    bump-strategy: vibes
`,
			wantInErr: "branches[0].bump-strategy",
		},
		{
			name: "stamp-target missing path",
			yaml: `stamp-targets:
  - format: toml
    key: version
`,
			wantInErr: "stamp-targets[0].path",
		},
		{
			name: "stamp-target missing format",
			yaml: `stamp-targets:
  - path: Cargo.toml
    key: package.version
`,
			wantInErr: "stamp-targets[0].format",
		},
		{
			name: "stamp-target invalid format",
			yaml: `stamp-targets:
  - path: foo.ini
    format: ini
    key: version
`,
			wantInErr: "stamp-targets[0].format",
		},
		{
			name: "stamp-target both key and keys",
			yaml: `stamp-targets:
  - path: Chart.yaml
    format: yaml
    key: version
    keys: [version, appVersion]
`,
			wantInErr: "stamp-targets[0]",
		},
		{
			name: "stamp-target neither key nor keys",
			yaml: `stamp-targets:
  - path: Chart.yaml
    format: yaml
`,
			wantInErr: "stamp-targets[0]",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte(tc.yaml), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := Load(dir)
			if err == nil {
				t.Fatalf("expected validation error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantInErr) {
				t.Errorf("error %q should contain %q", err.Error(), tc.wantInErr)
			}
		})
	}
}

func TestLoad_fixedKeysFormatsAcceptOmittedKeys(t *testing.T) {
	// plain, text, and helm engines determine their own keys (plain replaces
	// the whole file; helm always rewrites version + appVersion). Config
	// validation must NOT require `key:` or `keys:` for these formats.
	cases := map[string]string{
		"plain": `stamp-targets:
  - path: VERSION
    format: plain
`,
		"text": `stamp-targets:
  - path: VERSION
    format: text
`,
		"helm": `stamp-targets:
  - path: charts/demo/Chart.yaml
    format: helm
`,
	}
	for name, yaml := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte(yaml), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(dir); err != nil {
				t.Errorf("format %q with no keys should pass validation, got: %v", name, err)
			}
		})
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
