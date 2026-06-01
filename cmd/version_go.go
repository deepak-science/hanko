package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var versionGoLdflagsPackage string

var versionGoLdflagsCmd = &cobra.Command{
	Use:   "go-ldflags",
	Short: "Emit -ldflags for stamping a Go binary at build time",
	Long: `Emit a single line of -X flags suitable for splicing into
` + "`go build -ldflags \"$(hanko version go-ldflags)\" ./...`" + `.

By default stamps three variables on package "main": version (full SemVer),
commit (full SHA), date (committer date of HEAD). Pass --package to stamp a
different import path.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		v, err := resolveVersion("")
		if err != nil {
			return err
		}
		pkg := versionGoLdflagsPackage
		parts := []string{
			fmt.Sprintf("-X %s.version=%s", pkg, v.SemVer),
			fmt.Sprintf("-X %s.commit=%s", pkg, v.Sha),
			fmt.Sprintf("-X %s.date=%s", pkg, v.CommitDate),
		}
		fmt.Println(strings.Join(parts, " "))
		return nil
	},
}

func init() {
	versionGoLdflagsCmd.Flags().StringVar(&versionGoLdflagsPackage, "package", "main", "Go import path of the package whose variables get stamped")
	versionCmd.AddCommand(versionGoLdflagsCmd)
}
