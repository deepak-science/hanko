package smoke

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/dmallubhotla/hanko/cmd"
	"github.com/dmallubhotla/hanko/internal/testrepo"
)

const reexecEnv = "HANKO_SMOKE_REEXEC"

// hankoBin is the executable the tests invoke as hanko.
var hankoBin string

func TestMain(m *testing.M) {
	if os.Getenv(reexecEnv) == "1" {
		cmd.Execute()
		os.Exit(0)
	}
	home, err := setup()
	if err != nil {
		fmt.Fprintln(os.Stderr, "smoke setup:", err)
		os.Exit(2)
	}
	code := m.Run()
	_ = os.RemoveAll(home)
	os.Exit(code)
}

// setup picks the hanko binary and isolates the environment, returning the temp HOME to clean up.
func setup() (string, error) {
	env := map[string]string{}
	if bin := os.Getenv("HANKO"); bin != "" {
		// go test runs in the package dir, so a relative HANKO is read from the module root, where it was typed.
		if !filepath.IsAbs(bin) {
			root, err := moduleRoot()
			if err != nil {
				return "", err
			}
			bin = filepath.Join(root, bin)
		}
		hankoBin = bin
	} else {
		self, err := os.Executable()
		if err != nil {
			return "", err
		}
		hankoBin = self
		env[reexecEnv] = "1"
	}

	// Isolate git and hanko's log dir from the developer's home and global gitconfig.
	home, err := os.MkdirTemp("", "hanko-smoke-home-")
	if err != nil {
		return "", err
	}
	env["HOME"] = home
	env["XDG_CACHE_HOME"] = filepath.Join(home, ".cache")
	env["GIT_CONFIG_NOSYSTEM"] = "1"
	for k, v := range env {
		if err := os.Setenv(k, v); err != nil {
			return home, err
		}
	}
	return home, nil
}

func moduleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.mod above the working directory")
		}
		dir = parent
	}
}

type result struct {
	stdout, stderr, combined string
	code                     int
}

// lockedBuffer lets stdout and stderr interleave into one buffer, like `2>&1`.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

// hanko runs `hanko --repo dir args...` and returns its output and exit code; it only fails the test if the process can't be started.
func hanko(t *testing.T, dir string, args ...string) result {
	t.Helper()
	c := exec.Command(hankoBin, append([]string{"--repo", dir}, args...)...)
	var stdout, stderr bytes.Buffer
	var combined lockedBuffer
	c.Stdout = io.MultiWriter(&stdout, &combined)
	c.Stderr = io.MultiWriter(&stderr, &combined)
	err := c.Run()
	code := 0
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("running hanko %v: %v", args, err)
		}
		code = exitErr.ExitCode()
	}
	return result{
		stdout:   trim(stdout.String()),
		stderr:   trim(stderr.String()),
		combined: trim(combined.buf.String()),
		code:     code,
	}
}

// ok runs hanko, fails the test fatally on a non-zero exit, and returns stdout.
func ok(t *testing.T, dir string, args ...string) string {
	t.Helper()
	r := hanko(t, dir, args...)
	if r.code != 0 {
		t.Fatalf("hanko %v: exit %d\n%s", args, r.code, r.combined)
	}
	return r.stdout
}

// fails runs hanko, asserts it exits 1, and returns the combined output.
func fails(t *testing.T, dir string, args ...string) string {
	t.Helper()
	r := hanko(t, dir, args...)
	if r.code != 1 {
		t.Errorf("hanko %v: want exit 1, got exit %d\n%s", args, r.code, r.combined)
	}
	return r.combined
}

// trim mirrors shell command substitution, which strips trailing newlines.
func trim(s string) string { return strings.TrimRight(s, "\n") }

// git runs git in the repo and returns trimmed output.
func git(r *testrepo.Builder, args ...string) string { return trim(r.Git(args...)) }

func eq(t *testing.T, name, want, got string) {
	t.Helper()
	if want != got {
		t.Errorf("%s:\n  want: %q\n  got:  %q", name, want, got)
	}
}

func contains(t *testing.T, name, needle, haystack string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("%s:\n  want substring: %q\n  in:             %q", name, needle, haystack)
	}
}

// lineWithPrefix returns the first line of the repo file starting with prefix, or "" if none.
func lineWithPrefix(t *testing.T, r *testrepo.Builder, file, prefix string) string {
	t.Helper()
	for _, l := range strings.Split(r.ReadFile(file), "\n") {
		if strings.HasPrefix(l, prefix) {
			return l
		}
	}
	return ""
}

func lines(s ...string) string { return strings.Join(s, "\n") }

// touch creates an empty file at path.
func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
}
