package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"net"
	"os"
	"time"
)

var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Ping remote host",
	Run: func(cmd *cobra.Command, args []string) {
		host := viper.GetString("host")
		conn, err := net.DialTimeout("tcp", host, 5*time.Second)
		if err != nil {
			os.Exit(1)
		}
		defer conn.Close()
	},
}

func init() {
	rootCmd.AddCommand(pingCmd)
}
