package cmd

import (
	"errors"
	"fmt"
	"github.com/labstack/tunnel-client/daemon"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop [id]",
	Short: "Stop connection by id",
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
			fmt.Println(err)
		} else {
			defer c.Close()
			rep := new(daemon.StopReply)
			s.Start()
			defer s.Stop()
			err = c.Call("Server.Stop", daemon.StopRequest{
				ID: args[0],
			}, rep)
			if err != nil {
				fmt.Println(err)
			} else {
				psRPC()
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
