package stamper

import (
	"strings"
	"testing"
)

func TestStamp_nixSingleKey(t *testing.T) {
	in := `{
  packages.default = mkDerivation {
    version = "0.1.0";
  };
}
`
	out, desc, err := Stamp("nix", []byte(in), []string{"version"}, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `version = "1.0.0";`) {
		t.Errorf("not rewritten:\n%s", out)
	}
	if !strings.Contains(desc, "version: 0.1.0 → 1.0.0") {
		t.Errorf("desc = %q", desc)
	}
}

func TestStamp_yamlMultipleKeys(t *testing.T) {
	in := `apiVersion: v2
name: demo
version: 0.5.0
appVersion: "0.5.0"
`
	out, desc, err := Stamp("yaml", []byte(in), []string{"version", "appVersion"}, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, "version: 1.2.3\n") {
		t.Errorf("bare version not rewritten:\n%s", s)
	}
	if !strings.Contains(s, `appVersion: "1.2.3"`) {
		t.Errorf("quoted appVersion not rewritten:\n%s", s)
	}
	if !strings.Contains(desc, "version:") || !strings.Contains(desc, "appVersion:") {
		t.Errorf("desc = %q", desc)
	}
}

func TestStamp_yamlRefusesMissingKey(t *testing.T) {
	in := "apiVersion: v2\nname: demo\nversion: 0.5.0\n"
	_, _, err := Stamp("yaml", []byte(in), []string{"version", "appVersion"}, "1.2.3")
	if err == nil {
		t.Fatal("expected error for missing appVersion")
	}
	if !strings.Contains(err.Error(), "appVersion") {
		t.Errorf("error should name the missing key: %s", err)
	}
}

func TestStamp_tomlNestedKey(t *testing.T) {
	in := `[project]
name = "demo"
version = "0.1.0"
description = "test"

[project.urls]
home = "https://example.invalid"
`
	out, desc, err := Stamp("toml", []byte(in), []string{"project.version"}, "9.9.9")
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, `version = "9.9.9"`) {
		t.Errorf("project.version not rewritten:\n%s", s)
	}
	if !strings.Contains(s, `home = "https://example.invalid"`) {
		t.Errorf("unrelated url key was touched:\n%s", s)
	}
	if !strings.Contains(desc, "project.version: 0.1.0 → 9.9.9") {
		t.Errorf("desc = %q", desc)
	}
}

func TestStamp_tomlIgnoresKeyInOtherSection(t *testing.T) {
	// A `version` key in [tool.poetry] shouldn't be touched when we asked
	// for [project] section.
	in := `[project]
name = "demo"
version = "0.1.0"

[tool.poetry]
version = "0.2.0"
`
	out, _, err := Stamp("toml", []byte(in), []string{"project.version"}, "9.9.9")
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, `version = "9.9.9"`) {
		t.Errorf("project.version not rewritten:\n%s", s)
	}
	if !strings.Contains(s, `version = "0.2.0"`) {
		t.Errorf("tool.poetry.version was wrongly touched:\n%s", s)
	}
}

func TestStamp_tomlTopLevelKey(t *testing.T) {
	// Cargo.toml has its version inside [package]; but some configs put
	// the version at the top before any section. Confirm top-level lookup
	// works when section is empty.
	in := `version = "0.1.0"
edition = "2021"
`
	out, _, err := Stamp("toml", []byte(in), []string{"version"}, "9.9.9")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `version = "9.9.9"`) {
		t.Errorf("top-level version not rewritten:\n%s", out)
	}
}

func TestStamp_jsonSingleKey(t *testing.T) {
	in := `{
  "name": "demo",
  "version": "0.1.0",
  "description": "test"
}
`
	out, desc, err := Stamp("json", []byte(in), []string{"version"}, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"version": "1.0.0"`) {
		t.Errorf("not rewritten:\n%s", out)
	}
	if !strings.Contains(desc, "version: 0.1.0 → 1.0.0") {
		t.Errorf("desc = %q", desc)
	}
}

func TestStamp_jsonRefusesMissingKey(t *testing.T) {
	in := `{
  "name": "demo"
}
`
	_, _, err := Stamp("json", []byte(in), []string{"version"}, "1.0.0")
	if err == nil {
		t.Fatal("expected error for missing version key")
	}
}

func TestStamp_plainOverwritesEntireFile(t *testing.T) {
	in := "0.1.0\n"
	out, desc, err := Stamp("plain", []byte(in), nil, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "1.2.3\n" {
		t.Errorf("plain output = %q, want %q", string(out), "1.2.3\n")
	}
	if !strings.Contains(desc, "0.1.0 → 1.2.3") {
		t.Errorf("desc = %q", desc)
	}
}

func TestStamp_plainHandlesMissingNewline(t *testing.T) {
	in := "0.1.0"
	out, _, err := Stamp("plain", []byte(in), nil, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "1.2.3\n" {
		t.Errorf("plain output = %q, want trailing newline", string(out))
	}
}

func TestStamp_unknownFormat(t *testing.T) {
	_, _, err := Stamp("xml", []byte("<v>0.1.0</v>"), []string{"v"}, "1.0.0")
	if err == nil {
		t.Fatal("expected error for unknown format")
	}
	if !strings.Contains(err.Error(), "unknown stamp format") {
		t.Errorf("error message: %s", err)
	}
}

func TestStamp_nixDivergentValuesRefused(t *testing.T) {
	// D-015: same multi-derivation refusal logic as cmd/stamp_nix.
	in := `{
  a = mkDerivation { version = "0.1.0"; };
  b = mkDerivation { version = "0.2.0"; };
}
`
	_, _, err := Stamp("nix", []byte(in), []string{"version"}, "9.9.9")
	if err == nil {
		t.Fatal("expected error for divergent versions")
	}
}

func TestStamp_helmRewritesBothKeys(t *testing.T) {
	in := `apiVersion: v2
name: demo
version: 0.0.0
appVersion: "0.0.0"
`
	out, desc, err := Stamp("helm", []byte(in), nil, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	want := `apiVersion: v2
name: demo
version: 1.2.3
appVersion: "1.2.3"
`
	if string(out) != want {
		t.Errorf("output:\n%s\nwant:\n%s", out, want)
	}
	if !strings.Contains(desc, "version:") || !strings.Contains(desc, "appVersion:") {
		t.Errorf("desc = %q", desc)
	}
}

func TestStamp_helmForcesQuotedAppVersionWhenBare(t *testing.T) {
	// Helm convention: appVersion is a string. If the file declared it bare,
	// rewrite it quoted to keep YAML parsers from misreading numeric-looking
	// values.
	in := `apiVersion: v2
name: demo
version: 0.0.0
appVersion: 0.0.0
`
	out, _, err := Stamp("helm", []byte(in), nil, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, `appVersion: "1.2.3"`) {
		t.Errorf("expected appVersion to come back quoted:\n%s", s)
	}
	if !strings.Contains(s, "version: 1.2.3\n") {
		t.Errorf("expected version to stay bare:\n%s", s)
	}
}

func TestStamp_helmPreservesCommentsAndOrder(t *testing.T) {
	in := `# top comment
apiVersion: v2
appVersion: "0.5.0"   # bumped by hanko
name: demo
version: 0.5.0  # also bumped
type: application
`
	out, _, err := Stamp("helm", []byte(in), nil, "9.9.9")
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, `# top comment`) {
		t.Errorf("top comment lost:\n%s", s)
	}
	if !strings.Contains(s, `# bumped by hanko`) {
		t.Errorf("trailing comment lost:\n%s", s)
	}
	if !strings.Contains(s, `appVersion: "9.9.9"`) || !strings.Contains(s, `version: 9.9.9`) {
		t.Errorf("keys not rewritten:\n%s", s)
	}
}

func TestStamp_helmIdempotent(t *testing.T) {
	in := "apiVersion: v2\nversion: 1.2.3\nappVersion: \"1.2.3\"\n"
	out, _, err := Stamp("helm", []byte(in), nil, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != in {
		t.Errorf("expected no-op when version unchanged:\nin:  %q\nout: %q", in, out)
	}
}

func TestStamp_helmRefusesMissingKey(t *testing.T) {
	cases := map[string]string{
		"missing version":    "apiVersion: v2\nname: demo\nappVersion: \"0.0.0\"\n",
		"missing appVersion": "apiVersion: v2\nname: demo\nversion: 0.0.0\n",
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			_, _, err := Stamp("helm", []byte(in), nil, "1.0.0")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestFormatHasFixedKeys(t *testing.T) {
	for _, f := range []string{"helm", "plain", "text"} {
		if !FormatHasFixedKeys(f) {
			t.Errorf("FormatHasFixedKeys(%q) = false, want true", f)
		}
	}
	for _, f := range []string{"yaml", "yml", "toml", "json", "nix", "bogus"} {
		if FormatHasFixedKeys(f) {
			t.Errorf("FormatHasFixedKeys(%q) = true, want false", f)
		}
	}
}

func TestStamp_nixMultipleKeysIndependently(t *testing.T) {
	// User explicitly asks for two different keys; each gets its own
	// match-and-update pass.
	in := `{
  version = "0.1.0";
  appVersion = "0.1.0";
}
`
	out, _, err := Stamp("nix", []byte(in), []string{"version", "appVersion"}, "9.9.9")
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, `version = "9.9.9";`) || !strings.Contains(s, `appVersion = "9.9.9";`) {
		t.Errorf("expected both keys rewritten:\n%s", s)
	}
}

func TestStamp_nixPreservesCommentsAndOrder(t *testing.T) {
	in := `{
  # top-level comment
  outputs = _: {
    pkg = mkDerivation {
      version = "0.5.0"; # bumped by hanko
      pname = "demo";
    };
  };
}
`
	out, _, err := Stamp("nix", []byte(in), []string{"version"}, "9.9.9")
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, "# top-level comment") {
		t.Errorf("top comment lost:\n%s", s)
	}
	if !strings.Contains(s, "# bumped by hanko") {
		t.Errorf("trailing comment lost:\n%s", s)
	}
	if !strings.Contains(s, `version = "9.9.9";`) {
		t.Errorf("version not rewritten:\n%s", s)
	}
}

func TestStamp_nixMultipleMatchingValuesAllReplaced(t *testing.T) {
	// D-015: when a flake has multiple derivations sharing the same value
	// for a key, every matching line is rewritten.
	in := `{
  packages.first = mkDerivation {
    pname = "first";
    version = "0.1.0";
  };
  packages.second = mkDerivation {
    pname = "second";
    version = "0.1.0";
  };
}
`
	out, _, err := Stamp("nix", []byte(in), []string{"version"}, "9.9.9")
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if got := strings.Count(s, `version = "9.9.9";`); got != 2 {
		t.Errorf("expected 2 lines rewritten, got %d:\n%s", got, s)
	}
	if strings.Contains(s, `version = "0.1.0";`) {
		t.Errorf("no version lines should remain at the old value:\n%s", s)
	}
}

func TestStamp_nixIdempotent(t *testing.T) {
	in := `{
  version = "1.2.3";
}
`
	out, _, err := Stamp("nix", []byte(in), []string{"version"}, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != in {
		t.Errorf("expected no-op when value unchanged:\nin:  %q\nout: %q", in, out)
	}
}

func TestStamp_nixErrorsIfMissing(t *testing.T) {
	in := `{
  pname = "demo";
}
`
	_, _, err := Stamp("nix", []byte(in), []string{"version"}, "1.0.0")
	if err == nil {
		t.Error("expected error when key is absent, got nil")
	}
}

func TestStamp_nixSkipsNonStringValueLines(t *testing.T) {
	// A `version` attr whose value isn't a string literal (function call,
	// path, let-binding ref) shouldn't match the engine — the regex
	// requires `"..."`. The string-valued sibling still gets rewritten.
	in := `{
  version = pkgs.lib.fileContents ./VERSION;
  package = mkDerivation {
    version = "0.1.0";
  };
}
`
	out, _, err := Stamp("nix", []byte(in), []string{"version"}, "9.9.9")
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, `version = pkgs.lib.fileContents`) {
		t.Errorf("function-valued version was altered:\n%s", s)
	}
	if !strings.Contains(s, `version = "9.9.9";`) {
		t.Errorf("string-valued version not rewritten:\n%s", s)
	}
}
