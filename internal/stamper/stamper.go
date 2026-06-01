// Package stamper holds the per-format line-based engines that mutate a file
// to a new version value. Used by `hanko stamp` (no-args, config-driven) and
// indirectly by `hanko seal`.
//
// Each engine returns the rewritten file plus a one-line change description,
// or refuses with a descriptive error. No AST parsing — the engines walk
// lines, matching the canonical "key on its own line, scalar value" shape.
// Files that don't match the canonical shape are refused rather than guessed
// at (see D-015 and the Helm engine's note in stamp_helm.go).
package stamper

import (
	"fmt"
	"regexp"
	"strings"
)

// Stamp applies the engine for `format` to `content`, returning the rewritten
// file and a one-line "what changed" description. Returns an error if the
// engine can't unambiguously determine where to write.
//
// `keys` is the list of dotted-path keys to update; all get set to `newVal`.
// For "plain" and "helm" formats, keys is ignored — "plain" replaces the
// whole file, "helm" has fixed keys (version + appVersion).
func Stamp(format string, content []byte, keys []string, newVal string) ([]byte, string, error) {
	switch format {
	case "nix":
		return stampNix(content, keys, newVal)
	case "yaml", "yml":
		return stampYAML(content, keys, newVal)
	case "helm":
		return stampHelm(content, newVal)
	case "toml":
		return stampTOML(content, keys, newVal)
	case "json":
		return stampJSON(content, keys, newVal)
	case "plain", "text":
		return stampPlain(content, newVal)
	default:
		return nil, "", fmt.Errorf("unknown stamp format %q (want: nix, yaml, helm, toml, json, plain)", format)
	}
}

// FormatHasFixedKeys reports whether a format determines its own keys (and so
// ignores user-supplied `keys:` in `.hanko.yaml`). Callers use this to decide
// whether to require keys before dispatching to Stamp.
func FormatHasFixedKeys(format string) bool {
	switch format {
	case "plain", "text", "helm":
		return true
	}
	return false
}

// --- nix --------------------------------------------------------------------

// nixKeyLineRE matches `<key> = "..."` with leading whitespace, mandatory
// semicolon, optional trailing comment.
func nixKeyLineRE(key string) *regexp.Regexp {
	return regexp.MustCompile(`^(\s*` + regexp.QuoteMeta(key) + `\s*=\s*)"([^"]*)"(\s*;\s*(?:#.*)?)\s*$`)
}

// stampNix follows D-015: replace every `<key> = "X";` line sharing the
// current value; refuse divergent values.
func stampNix(content []byte, keys []string, newVal string) ([]byte, string, error) {
	if len(keys) == 0 {
		return nil, "", fmt.Errorf("nix engine: at least one key required")
	}
	out := string(content)
	var descs []string
	for _, key := range keys {
		var err error
		var desc string
		out, desc, err = stampNixOne(out, key, newVal)
		if err != nil {
			return nil, "", err
		}
		descs = append(descs, desc)
	}
	return []byte(out), strings.Join(descs, ", "), nil
}

func stampNixOne(content, key, newVal string) (string, string, error) {
	re := nixKeyLineRE(key)
	lines := strings.Split(content, "\n")

	type hit struct {
		idx                  int
		prefix, old, trailer string
	}
	var hits []hit
	for i, line := range lines {
		m := re.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		hits = append(hits, hit{i, m[1], m[2], m[3]})
	}
	if len(hits) == 0 {
		return "", "", fmt.Errorf("nix engine: no `%s = \"...\";` attr found", key)
	}
	for _, h := range hits[1:] {
		if h.old != hits[0].old {
			return "", "", fmt.Errorf("nix engine: multiple `%s = \"...\";` attrs with different values (%q vs %q); hoist to a shared `let` binding", key, hits[0].old, h.old)
		}
	}
	for _, h := range hits {
		lines[h.idx] = h.prefix + `"` + newVal + `"` + h.trailer
	}
	return strings.Join(lines, "\n"), fmt.Sprintf("%s: %s → %s", key, hits[0].old, newVal), nil
}

// --- yaml -------------------------------------------------------------------

// yamlTopLevelKeyRE matches `<key>: <val>` at the start of a line (no
// indentation, so only top-level keys). Value may be quoted or bare;
// optional trailing comment is preserved.
func yamlTopLevelKeyRE(key string) *regexp.Regexp {
	return regexp.MustCompile(`^(` + regexp.QuoteMeta(key) + `)(\s*:\s*)("[^"]*"|'[^']*'|[^\s#]*)(\s*(?:#.*)?)\s*$`)
}

// stampYAML rewrites top-level scalar keys. Supports multiple keys (e.g.
// Chart.yaml's `version` + `appVersion`). Preserves whichever quote style
// the file used; defaults to bare for new values.
//
// Top-level only — nested keys need indentation-aware parsing which lives in
// a future AST engine. Refuses if the key isn't found.
func stampYAML(content []byte, keys []string, newVal string) ([]byte, string, error) {
	if len(keys) == 0 {
		return nil, "", fmt.Errorf("yaml engine: at least one key required")
	}
	lines := strings.Split(string(content), "\n")
	seen := map[string]bool{}
	var changes []string

	for i, line := range lines {
		for _, key := range keys {
			m := yamlTopLevelKeyRE(key).FindStringSubmatch(line)
			if m == nil {
				continue
			}
			matchedKey, sep, val, trailing := m[1], m[2], m[3], m[4]
			old := strings.Trim(val, `"'`)

			newQuoted := newVal
			switch {
			case strings.HasPrefix(val, `"`):
				newQuoted = `"` + newVal + `"`
			case strings.HasPrefix(val, `'`):
				newQuoted = `'` + newVal + `'`
			}
			lines[i] = matchedKey + sep + newQuoted + trailing
			changes = append(changes, fmt.Sprintf("%s: %s → %s", matchedKey, old, newVal))
			seen[matchedKey] = true
			break
		}
	}
	for _, key := range keys {
		if !seen[key] {
			return nil, "", fmt.Errorf("yaml engine: no top-level `%s:` key found", key)
		}
	}
	return []byte(strings.Join(lines, "\n")), strings.Join(changes, ", "), nil
}

// --- helm -------------------------------------------------------------------

// helmChartKeyRE matches Chart.yaml's `version:` or `appVersion:` lines:
// optional quotes around the value, optional trailing comment. The captured
// groups let us preserve the prefix and any trailing comment when rewriting.
var helmChartKeyRE = regexp.MustCompile(`^(version|appVersion)(\s*:\s*)("[^"]*"|'[^']*'|[^\s#]*)(\s*(?:#.*)?)\s*$`)

// stampHelm rewrites the top-level `version:` and `appVersion:` keys in a
// Helm Chart.yaml to the given semver. Fixed-key engine — user-supplied
// `keys:` are ignored. Both keys must be present; missing either is an error.
//
// Quoting policy: preserve the file's existing quote style for both keys;
// when a value is bare, leave `version` bare but force `appVersion` quoted —
// Helm's convention, since appVersion is a string that often looks numeric.
func stampHelm(content []byte, newVal string) ([]byte, string, error) {
	lines := strings.Split(string(content), "\n")
	seen := map[string]bool{}
	var changes []string

	for i, line := range lines {
		m := helmChartKeyRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		key, sep, val, trailing := m[1], m[2], m[3], m[4]
		old := strings.Trim(val, `"'`)

		newQuoted := newVal
		switch {
		case strings.HasPrefix(val, `"`):
			newQuoted = `"` + newVal + `"`
		case strings.HasPrefix(val, `'`):
			newQuoted = `'` + newVal + `'`
		case key == "appVersion":
			newQuoted = `"` + newVal + `"`
		}
		lines[i] = key + sep + newQuoted + trailing
		changes = append(changes, fmt.Sprintf("%s: %s → %s", key, old, newVal))
		seen[key] = true
	}

	if !seen["version"] {
		return nil, "", fmt.Errorf("helm engine: no top-level `version:` key found")
	}
	if !seen["appVersion"] {
		return nil, "", fmt.Errorf("helm engine: no top-level `appVersion:` key found")
	}
	return []byte(strings.Join(lines, "\n")), strings.Join(changes, ", "), nil
}

// --- toml -------------------------------------------------------------------

// tomlKeyLineRE matches `<key> = "..."` with optional leading whitespace.
// Value can be double- or single-quoted; trailing comment preserved.
func tomlKeyLineRE(key string) *regexp.Regexp {
	return regexp.MustCompile(`^(\s*` + regexp.QuoteMeta(key) + `\s*=\s*)("[^"]*"|'[^']*')(\s*(?:#.*)?)\s*$`)
}

// tomlHeaderRE matches `[section]` or `[[section]]` headers (single-bracket
// is what we care about; double-bracket is array-of-tables which we don't
// support yet).
var tomlHeaderRE = regexp.MustCompile(`^\s*\[([^\]]+)\]\s*(?:#.*)?$`)

// stampTOML walks the file in section-aware fashion: each key is a dotted
// path like `project.version`. The last segment is the key; everything
// before is the section header (empty = top-level).
//
// Line-based; canonical "key = value on its own line" shape only.
func stampTOML(content []byte, keys []string, newVal string) ([]byte, string, error) {
	if len(keys) == 0 {
		return nil, "", fmt.Errorf("toml engine: at least one key required")
	}
	lines := strings.Split(string(content), "\n")

	// For each key, compute (section, leafKey).
	type target struct {
		section, key string
		found        bool
		oldVal       string
	}
	targets := make([]*target, len(keys))
	for i, k := range keys {
		parts := strings.Split(k, ".")
		t := &target{key: parts[len(parts)-1]}
		if len(parts) > 1 {
			t.section = strings.Join(parts[:len(parts)-1], ".")
		}
		targets[i] = t
	}

	currentSection := ""
	for i, line := range lines {
		if h := tomlHeaderRE.FindStringSubmatch(line); h != nil {
			currentSection = strings.TrimSpace(h[1])
			continue
		}
		for _, t := range targets {
			if t.found {
				continue
			}
			if t.section != currentSection {
				continue
			}
			m := tomlKeyLineRE(t.key).FindStringSubmatch(line)
			if m == nil {
				continue
			}
			prefix, val, trailing := m[1], m[2], m[3]
			old := strings.Trim(val, `"'`)
			newQuoted := `"` + newVal + `"`
			if strings.HasPrefix(val, `'`) {
				newQuoted = `'` + newVal + `'`
			}
			lines[i] = prefix + newQuoted + trailing
			t.found = true
			t.oldVal = old
		}
	}

	var changes []string
	for i, t := range targets {
		if !t.found {
			return nil, "", fmt.Errorf("toml engine: key %q not found", keys[i])
		}
		changes = append(changes, fmt.Sprintf("%s: %s → %s", keys[i], t.oldVal, newVal))
	}
	return []byte(strings.Join(lines, "\n")), strings.Join(changes, ", "), nil
}

// --- json -------------------------------------------------------------------

// jsonKeyLineRE matches `  "<key>": "<val>"` with optional leading whitespace,
// optional trailing comma, optional `// ...` comment (some package.json
// variants tolerate them, even though strict JSON forbids).
func jsonKeyLineRE(key string) *regexp.Regexp {
	return regexp.MustCompile(`^(\s*"` + regexp.QuoteMeta(key) + `"\s*:\s*)"([^"]*)"(\s*,?\s*(?://.*)?)\s*$`)
}

// stampJSON rewrites top-level string-valued keys. Limited to the canonical
// "one key per line" shape; nested keys deferred.
//
// Refuses if the key isn't found. Note: technically there's no notion of
// "top-level" without parsing the JSON, but the regex's requirement that the
// line is just `"key": "value"` keeps us out of nested object trouble in
// practice for shapes like package.json.
func stampJSON(content []byte, keys []string, newVal string) ([]byte, string, error) {
	if len(keys) == 0 {
		return nil, "", fmt.Errorf("json engine: at least one key required")
	}
	lines := strings.Split(string(content), "\n")
	seen := map[string]bool{}
	var changes []string

	for i, line := range lines {
		for _, key := range keys {
			if seen[key] {
				continue
			}
			m := jsonKeyLineRE(key).FindStringSubmatch(line)
			if m == nil {
				continue
			}
			prefix, old, trailing := m[1], m[2], m[3]
			lines[i] = prefix + `"` + newVal + `"` + trailing
			changes = append(changes, fmt.Sprintf("%s: %s → %s", key, old, newVal))
			seen[key] = true
		}
	}
	for _, key := range keys {
		if !seen[key] {
			return nil, "", fmt.Errorf("json engine: key %q not found", key)
		}
	}
	return []byte(strings.Join(lines, "\n")), strings.Join(changes, ", "), nil
}

// --- plain ------------------------------------------------------------------

// stampPlain replaces the entire file contents with `newVal` plus a trailing
// newline (canonical for VERSION-style files). Preserves the trailing-newline
// convention regardless of what was there before.
func stampPlain(content []byte, newVal string) ([]byte, string, error) {
	old := strings.TrimRight(string(content), "\n")
	return []byte(newVal + "\n"), fmt.Sprintf("contents: %s → %s", old, newVal), nil
}
