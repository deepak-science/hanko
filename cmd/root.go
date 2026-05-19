package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/dmallubhotla/hanko/internal/logging"
	"github.com/spf13/cobra"
)

var (
	verbose    bool
	repoPath   string
	logCleanup func()
)

var rootCmd = &cobra.Command{
	Use:   "hanko",
	Short: "Hanko — stamp versions and labels onto your build artifacts",
	Long: `Hanko computes versions from git history and stamps them onto
build artifacts (binaries, container images, helm charts, OS packages).
A drop-in replacement for GitVersion and friends.`,
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cleanup, err := logging.Init()
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not initialize logging: %v\n", err)
		} else {
			logCleanup = cleanup
		}

		slog.Info("hanko invoked",
			"command", cmd.CommandPath(),
			"args", args,
			"repo", repoPath,
		)
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if logCleanup != nil {
			logCleanup()
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// SetBuildInfo wires the build-time stamps onto the root command, exposing
// them via `hanko --version`. Called from main(), where the linker can set
// the vars via `-X main.version=…` ldflags.
func SetBuildInfo(version, commit, date string) {
	rootCmd.Version = fmt.Sprintf("%s (commit %s, built %s)", version, commit, date)
}

func init() {
	rootCmd.AddGroup(
		&cobra.Group{ID: "compute", Title: "Compute Commands:"},
		&cobra.Group{ID: "stamp", Title: "Stamp Commands:"},
	)

	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose/debug output")
	rootCmd.PersistentFlags().StringVar(&repoPath, "repo", ".", "path to the git repository")
}
