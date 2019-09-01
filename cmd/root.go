package cmd

import (
	"fmt"
	"github.com/labstack/gommon/log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "Tunnel lets you expose local servers to the internet securely",
	Long:  "Signup @ https://tunnel.labstack.com to get an api key and get started",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
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
	viper.Set("host", "labstack.me:22")
	viper.Set("api_url", "https://tunnel.labstack.com/api/v1")
	if dev := viper.GetString("DC") == "dev"; dev {
		viper.Set("host", "localhost:2200")
		viper.Set("api_url", "http://tunnel.labstack.d/api/v1")
		viper.SetConfigName("config.dev")
	} else {
		viper.SetConfigName("config")
	}
	viper.AddConfigPath(root)
	viper.ReadInConfig()
	viper.WatchConfig()
}
