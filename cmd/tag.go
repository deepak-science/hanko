package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var tagPush bool

var tagCmd = &cobra.Command{
	Use:     "tag",
	Short:   "Create a git tag for the computed version",
	GroupID: "stamp",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("not implemented: see ROADMAP.md milestone M2")
	},
}

func init() {
	tagCmd.Flags().BoolVar(&tagPush, "push", false, "push the tag to origin after creating it")
	rootCmd.AddCommand(tagCmd)
}
