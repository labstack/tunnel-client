package cmd

import (
	"errors"
	"fmt"
	"github.com/labstack/tunnel-client/daemon"
	"github.com/spf13/cobra"
)

var configuration string
var protocol string
var connectCmd = &cobra.Command{
	Use:   "connect [address]",
	Short: "Start a new connection from the target address",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("requires a target address")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		startDaemon()
		c, err := getClient()
		if err != nil {
			fmt.Println(err)
			return
		}
		defer c.Close()
		rep := new(daemon.ConnectReply)
		s.Start()
		err = c.Call("Server.Connect", &daemon.ConnectRequest{
			Configuration: configuration,
			Address:       args[0],
			Protocol:      daemon.Protocol(protocol),
		}, rep)
		if err != nil {
			fmt.Println(err)
		} else {
			s.Stop()
			psRPC()
		}
	},
}

func init() {
	rootCmd.AddCommand(connectCmd)
	connectCmd.PersistentFlags().StringVarP(&configuration, "configuration", "c", "",
		"configuration name from the console")
	connectCmd.PersistentFlags().StringVarP(&protocol, "protocol", "p", daemon.ProtocolHTTPS,
		"connection protocol (https, tcp, tls)")
}
