package cmd

import (
	"errors"

	"github.com/labstack/gommon/log"
	"github.com/labstack/tunnel-client/daemon"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop tunnel by name",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("requires a tunnel name")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		c, err := getClient()
		if err != nil {
			log.Fatal(err)
		}
		defer c.Close()
		rep := new(daemon.StopReply)
		err = c.Call("Daemon.Stop", daemon.StopRequest{
			Name: args[0],
		}, rep)
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}