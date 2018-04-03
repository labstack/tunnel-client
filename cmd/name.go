package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// nameCmd represents the name command
var nameCmd = &cobra.Command{
	Use:   "name",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("name called")
	},
}

func init() {
	rootCmd.AddCommand(nameCmd)
}
