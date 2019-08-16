package cmd

import (
	"github.com/hako/durafmt"
	"github.com/jedib0t/go-pretty/table"
	"github.com/labstack/gommon/log"
	"github.com/labstack/tunnel-client/daemon"
	"github.com/spf13/cobra"
	"os"
	"time"
)

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "List all active tunnels",
	Run: func(cmd *cobra.Command, args []string) {
		psRPC()
	},
}

func psRPC() {
	c, err := getClient()
	if err != nil {
		log.Fatal(err)
	}
	req := new(daemon.PSRequest)
	rep := new(daemon.PSReply)
	err = c.Call("Daemon.PS", req, rep)
	if err != nil {
		log.Fatal(err)
	}
	tbl := table.NewWriter()
	tbl.SetOutputMirror(os.Stdout)
	tbl.AppendHeader(table.Row{"Name", "Target Address", "Remote URI", "Status", "Uptime"})
	for _, t := range rep.Tunnels {
		uptime := durafmt.ParseShort(time.Since(t.CreatedAt)).String()
		if t.CreatedAt.IsZero() {
			uptime = ""
		}
		tbl.AppendRow([]interface{}{t.Name, t.TargetAddress, t.RemoteURI, t.Status, uptime})
	}
	tbl.Render()
}

func init() {
	rootCmd.AddCommand(psCmd)
}
