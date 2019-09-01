package cmd

import (
	"errors"
	"github.com/labstack/tunnel-client/daemon"

	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start [id]",
	Short: "Start an existing connection by id",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("requires a connection id")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		startDaemon()
		c, err := getClient()
		if err != nil {
			exit(err)
		}
		defer c.Close()
		rep := new(daemon.StartReply)
		s.Start()
		defer s.Stop()
		err = c.Call("Server.Start", &daemon.StartRequest{
			ID: args[0],
		}, rep)
		if err != nil {
			exit(err)
		}
		psRPC()
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.PersistentFlags().StringVarP(&configuration, "configuration", "c", "",
		"configuration name from the console")
	startCmd.PersistentFlags().StringVarP(&protocol, "protocol", "p", daemon.ProtocolHTTPS,
		"connection protocol (https, tcp, tls)")
}
