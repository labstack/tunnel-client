package cmd

import (
	"github.com/labstack/gommon/log"
	"github.com/labstack/tunnel-client/daemon"
	"github.com/mitchellh/go-ps"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func startDaemon() {
	start := true
	d, err := ioutil.ReadFile(viper.GetString("daemon_pid"))
	if err == nil {
		pid, _ := strconv.Atoi(string(d))
		if p, _ := ps.FindProcess(pid); p != nil {
			start = false
		}
	}
	if start {
		e, err := os.Executable()
		if err != nil {
			log.Fatal(err)
		}
		c := exec.Command(e, "daemon")
		c.SysProcAttr = sysProcAttr
		f, err := os.OpenFile(viper.GetString("log_file"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		c.Stdout = f
		c.Stderr = f
		if err := c.Start(); err != nil {
			log.Fatal(err)
		}
		if err := ioutil.WriteFile(viper.GetString("daemon_pid"), []byte(strconv.Itoa(c.Process.Pid)), 0644); err != nil {
			log.Fatal(err)
		}
		time.Sleep(time.Second) // Let the daemon start
	}
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Start the tunnel daemon. It is automatically started as soon as the first command is executed.",
	Run: func(cmd *cobra.Command, args []string) {
		daemon.Start()
	},
}

func init() {
	rootCmd.AddCommand(daemonCmd)
}
