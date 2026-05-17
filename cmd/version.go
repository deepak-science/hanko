package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dmallubhotla/hanko/internal/gitinfo"
	"github.com/dmallubhotla/hanko/internal/version"
	"github.com/spf13/cobra"
)

var versionFormat string

var versionCmd = &cobra.Command{
	Use:     "version",
	Short:   "Compute the current version from git history",
	GroupID: "compute",
	RunE: func(cmd *cobra.Command, args []string) error {
		info, err := gitinfo.Read(repoPath)
		if err != nil {
			return fmt.Errorf("read git info: %w", err)
		}

		v, err := version.Compute(info)
		if err != nil {
			return fmt.Errorf("compute version: %w", err)
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
		default:
			return fmt.Errorf("unknown format %q (want: semver, full, json, env)", versionFormat)
		}
		return nil
	},
}

func init() {
	versionCmd.Flags().StringVarP(&versionFormat, "format", "f", "semver", "output format: semver, full, json, env")
	rootCmd.AddCommand(versionCmd)
}
