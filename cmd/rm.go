package cmd

import (
	"errors"
	"github.com/labstack/gommon/log"
	"github.com/labstack/tunnel-client/daemon"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm",
	Short: "Remove tunnel by name",
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
		rep := new(daemon.RMReply)
		err = c.Call("Daemon.RM", daemon.RMRequest{
			Name: args[0],
		}, rep)
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
}
