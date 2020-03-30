package cmd

import (
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update to latest release",
	Long:  `This command can be used to update to the latest release.`,
	Run: func(cmd *cobra.Command, args []string) {
		// init core
		initCore(true)

		// notify user command not implemented yet
		log.Warn("Command has not been implemented yet!")
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
