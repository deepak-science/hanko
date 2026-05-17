package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var stampCmd = &cobra.Command{
	Use:     "stamp",
	Short:   "Stamp computed version/labels onto a build artifact",
	GroupID: "stamp",
}

var stampDockerCmd = &cobra.Command{
	Use:   "docker [image]",
	Short: "Apply OCI labels (org.opencontainers.image.*) to a built image",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("not implemented: see ROADMAP.md milestone M3")
	},
}

var stampHelmCmd = &cobra.Command{
	Use:   "helm [chart-dir]",
	Short: "Set version and appVersion in a chart's Chart.yaml",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("not implemented: see ROADMAP.md milestone M3")
	},
}

var stampGoCmd = &cobra.Command{
	Use:   "go-ldflags",
	Short: "Emit -ldflags for stamping a Go binary",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("not implemented: see ROADMAP.md milestone M3")
	},
}

func init() {
	stampCmd.AddCommand(stampDockerCmd, stampHelmCmd, stampGoCmd)
	rootCmd.AddCommand(stampCmd)
}
