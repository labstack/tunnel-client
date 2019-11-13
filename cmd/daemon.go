package cmd

import (
  "errors"
  "github.com/labstack/tunnel-client/daemon"
  "github.com/radovskyb/watcher"
  "github.com/spf13/cobra"
  "github.com/spf13/viper"
  "io/ioutil"
  "os"
  "os/exec"
  "strconv"
  "time"
)

func startDaemon() {
  if viper.GetString("api_key") == "" {
    exit("To use tunnel you need an api key (https://tunnel.labstack.com) in $HOME/.tunnel/config.yaml")
  }
  start := true
  d, err := ioutil.ReadFile(viper.GetString("daemon_pid"))
  if err == nil {
    pid, _ := strconv.Atoi(string(d))
    if p, _ := os.FindProcess(pid); p != nil {
      start = false
    }
  }
  if start {
    exe, err := os.Executable()
    if err != nil {
      exit(err)
    }
    c := exec.Command(exe, "daemon", "start")
    c.SysProcAttr = sysProcAttr
    f, err := os.OpenFile(viper.GetString("log_file"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
      exit(err)
    }
    c.Stdout = f
    c.Stderr = f
    if err := c.Start(); err != nil {
      exit(err)
    }
    if err := ioutil.WriteFile(viper.GetString("daemon_pid"), []byte(strconv.Itoa(c.Process.Pid)), 0644); err != nil {
      exit(err)
    }

    // Wait for daemon to start
    w := watcher.New()
    w.SetMaxEvents(1)
    w.FilterOps(watcher.Create)
    go func() {
      e := <-w.Event
      if e.Name() == "daemon.addr" {
        w.Close()
      }
    }()
    w.Add(viper.GetString("root"))
    w.Start(50 * time.Millisecond)
  }
}

var daemonCmd = &cobra.Command{
  Use:   "daemon",
  Short: "Start/stop the tunnel daemon. It is automatically started as soon as the first command is executed.",
  Args: func(cmd *cobra.Command, args []string) error {
    if len(args) < 1 {
      return errors.New("requires an argument (start/stop)")
    }
    return nil
  },
  Run: func(cmd *cobra.Command, args []string) {
    if args[0] == "start" {
      daemon.Start()
    } else if args[0] == "stop" {
      defer os.Remove(viper.GetString("daemon_addr"))
      defer os.Remove(viper.GetString("daemon_pid"))
      d, _ := ioutil.ReadFile(viper.GetString("daemon_pid"))
      pid, _ := strconv.Atoi(string(d))
      p, _ := os.FindProcess(pid)
      p.Signal(os.Interrupt)
    }
  },
}

func init() {
  rootCmd.AddCommand(daemonCmd)
}
