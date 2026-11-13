// Package smoke holds end-to-end CLI tests: they run hanko as a subprocess against throwaway git repos and assert on stdout, stderr, exit codes and repo side effects.
// Unit tests under internal/ cover the library layer; these cover command-line plumbing, exit codes and idempotency.
//
// By default the test binary re-execs itself as hanko (see TestMain), so `go test ./test/smoke/...` needs no prior build.
// Set HANKO=/path/to/hanko to test a real build instead, e.g. the nix-built binary in the flake's smoke check.
//
// The tests live in _test.go files; this file exists so the package has a non-test file for `go build ./...`.
package smoke
