package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	versionFormat string
	versionBump   string
)

var versionCmd = &cobra.Command{
	Use:     "version",
	Short:   "Compute the current version from git history",
	GroupID: "compute",
	RunE: func(cmd *cobra.Command, args []string) error {
		switch versionBump {
		case "", "patch", "minor", "major", "none":
			// valid
		default:
			return fmt.Errorf("unknown --bump %q (want: patch, minor, major, none)", versionBump)
		}
		v, err := resolveVersion(versionBump)
		if err != nil {
			return err
		}

		if verbose {
			fmt.Fprint(os.Stderr, v.Decision.Format())
		}

		switch versionFormat {
		case "semver":
			fmt.Println(v.SemVer)
		case "full":
			fmt.Println(v.FullSemVer)
		case "json":
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(v)
		case "env":
			for k, val := range v.AsEnv() {
				fmt.Printf("%s=%s\n", k, val)
			}
		case "gha":
			// Field names match cicd's resolve-version composite action so
			// hanko is a drop-in replacement. See docs/design-decisions.md D-006.
			for k, val := range v.AsGHA() {
				fmt.Printf("%s=%s\n", k, val)
			}
		default:
			return fmt.Errorf("unknown format %q (want: semver, full, json, env, gha)", versionFormat)
		}
		return nil
	},
}

func init() {
	versionCmd.Flags().StringVarP(&versionFormat, "format", "f", "semver", "output format: semver, full, json, env, gha")
	versionCmd.Flags().StringVar(&versionBump, "bump", "", "force a bump direction for this invocation: patch | minor | major | none (overrides bump-strategy)")
	rootCmd.AddCommand(versionCmd)
}
