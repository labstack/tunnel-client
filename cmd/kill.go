package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var killCmd = &cobra.Command{
	Use:   "kill",
	Short: "Stop the tunnel daemon",
	Run: func(cmd *cobra.Command, args []string) {
		pid := viper.GetInt("daemon_pid")
		p, err := os.FindProcess(pid)
		if err != nil {
			log.Fatal(err)
		}
		p.Kill()
		os.Remove(viper.GetString("daemon_pid"))
		os.Remove(viper.GetString("daemon_addr"))
	},
}

func init() {
	rootCmd.AddCommand(killCmd)
}
