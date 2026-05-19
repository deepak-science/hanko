package main

import "github.com/dmallubhotla/hanko/cmd"

// Build-time stamps. Defaults apply for `go build` / `go run`; nix injects
// real values via `-X main.version=…` ldflags.
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	cmd.SetBuildInfo(version, commit, date)
	cmd.Execute()
}
