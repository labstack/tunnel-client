package cmd

import (
	"errors"
	"github.com/labstack/gommon/log"
	"github.com/labstack/tunnel-client/daemon"

	"github.com/spf13/cobra"
)

var name string
var protocol string
var startCmd = &cobra.Command{
	Use:   "start [address]",
	Short: "Start tunnel from the target address",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("requires a target address argument")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		c, err := getClient()
		if err != nil {
			log.Fatal(err)
		}
		rep := new(daemon.StartReply)
		s.Start()
		err = c.Call("Daemon.Start", &daemon.StartRequest{
			Name:     name,
			Address:  args[0],
			Protocol: daemon.Protocol(protocol),
		}, rep)
		if err != nil {
			log.Fatal(err)
		}
		s.Stop()
		psRPC()
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.PersistentFlags().StringVarP(&name, "configuration", "c", "", "configuration name from the console")
	startCmd.PersistentFlags().StringVarP(&protocol, "protocol", "p", daemon.ProtocolHTTP, "tunnel protocol (http, tcp, tls)")
}
