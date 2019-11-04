package cmd

import (
	"errors"
	"github.com/labstack/tunnel-client/daemon"

	"github.com/spf13/cobra"
)

var force bool
var rmCmd = &cobra.Command{
	Use:   "rm [name]",
	Short: "Remove connection by name",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("requires a connection name")
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
		rep := new(daemon.RMReply)
		s.Start()
		defer s.Stop()
		err = c.Call("Server.RM", daemon.RMRequest{
			Name:    args[0],
			Force: force,
		}, rep)
		if err != nil {
			exit(err)
		}
		psRPC()
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
	rmCmd.PersistentFlags().BoolVarP(&force, "force", "f", false, "force remove a connection")
}
