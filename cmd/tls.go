package cmd

import (
	"log"

	"github.com/labstack/tunnel"
	"github.com/labstack/tunnel/util"
	"github.com/spf13/cobra"
)

var (
	tlsCmd = &cobra.Command{
		Use:   "tls",
		Short: "Forward TLS traffic from internet to a target address",
		// Long: ``,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			t := &tunnel.Tunnel{
				Protocol:   "tls",
				RemoteHost: "0.0.0.0",
				RemotePort: 0,
			}
			t.TargetHost, t.TargetPort, err = util.SplitHostPort(args[0])
			if err != nil {
				log.Fatalf("Failed to parse target address %v\n", err)
			}
			t.Create()
		},
	}
)

func init() {
	rootCmd.AddCommand(tlsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// tlsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// tlsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
