package cmd

import (
  "errors"
  "fmt"
  "net"
  "os"
  "path/filepath"
  "strings"

  "github.com/labstack/tunnel-client/daemon"
  "github.com/spf13/cobra"

  "github.com/mitchellh/go-homedir"
  "github.com/spf13/viper"
)

var configuration string
var protocol string
var rootCmd = &cobra.Command{
  Use:   "tunnel [address]",
  Short: "Tunnel lets you expose local servers to the internet securely",
  Long:  "Signup @ https://tunnel.labstack.com to get an api key and get started",
  Args: func(cmd *cobra.Command, args []string) error {
    if len(args) < 1 {
      return errors.New("requires a target address")
    }
    return nil
  },
  Run: func(cmd *cobra.Command, args []string) {
    s.Start()
    startDaemon()
    c, err := getClient()
    if err != nil {
      exit(err)
    }
    defer c.Close()
    rep := new(daemon.ConnectReply)
    addr := args[0]
    _, _, err = net.SplitHostPort(addr)
    if err != nil && strings.Contains(err.Error(), "missing port") {
      addr = ":" + addr
    }
    err = c.Call("Server.Connect", &daemon.ConnectRequest{
      Configuration: configuration,
      Address:       addr,
      Protocol:      daemon.Protocol(protocol),
    }, rep)
    if err != nil {
      exit(err)
    }
    s.Stop()
    psRPC()
  },
}

func Execute() {
  if err := rootCmd.Execute(); err != nil {
    exit(err)
  }
}

func init() {
  cobra.OnInitialize(initialize)
}

func initialize() {
  // Create directories
  dir, err := homedir.Dir()
  if err != nil {
    fmt.Printf("failed to find the home directory: %v", err)
  }
  root := filepath.Join(dir, ".tunnel")
  if err = os.MkdirAll(root, 0755); err != nil {
    fmt.Printf("failed to create root directory: %v", err)
  }
  if _, err := os.OpenFile(filepath.Join(root, "config.yaml"), os.O_RDONLY|os.O_CREATE, 0644); err != nil {
    fmt.Printf("failed to create config file: %v", err)
  }

  // Config
  viper.AutomaticEnv()
  viper.Set("root", root)
  viper.Set("log_file", filepath.Join(root, "daemon.log"))
  viper.Set("daemon_pid", filepath.Join(root, "daemon.pid"))
  viper.Set("daemon_addr", filepath.Join(root, "daemon.addr"))
  viper.Set("hostname", "labstack.me")
  viper.Set("port", 22)
  viper.Set("remote_port", 80)
  viper.Set("api_url", "https://tunnel.labstack.com/api/v1")
  if dev := viper.GetString("DC") == "dev"; dev {
    viper.Set("hostname", "labstack.d")
    viper.Set("port", 2200)
    viper.Set("remote_port", 8000)
    viper.Set("api_url", "http://tunnel.labstack.d/api/v1")
    viper.SetConfigName("config.dev")
  } else {
    viper.SetConfigName("config")
  }
  viper.AddConfigPath(root)
  viper.ReadInConfig()
  viper.WatchConfig()
}

func init() {
  rootCmd.PersistentFlags().StringVarP(&configuration, "configuration", "c", "",
    "configuration name from the console")
  rootCmd.PersistentFlags().StringVarP(&protocol, "protocol", "p", string(daemon.ProtocolHTTP),
    "connection protocol (http, tcp, tls)")
}
