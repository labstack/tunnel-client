package cmd

import (
	"net"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Ping remote host",
	Run: func(cmd *cobra.Command, args []string) {
		host := net.JoinHostPort(viper.GetString("hostname"), "22")
		conn, err := net.DialTimeout("tcp", host, 5*time.Second)
		if err != nil {
			exit(err)
		}
		defer conn.Close()
	},
}

func init() {
	rootCmd.AddCommand(pingCmd)
}
