package cmd

import (
	"fmt"
	"log"
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
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initialize)
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func initialize() {
	// Create directories
	dir, err := homedir.Dir()
	if err != nil {
		log.Fatalf("failed to find the home directory: %v", err)
	}
	root := filepath.Join(dir, ".tunnel")

	// Add to viper
	viper.Set("root", root)
	viper.Set("log_file", filepath.Join(root, "daemon.log"))
	viper.Set("daemon_pid", filepath.Join(root, "daemon.pid"))
	viper.Set("daemon_addr", filepath.Join(root, "daemon.addr"))

	// Search config in home directory with name ".config" (without extension).
	viper.AddConfigPath(root)
	viper.SetConfigName("config")

	// Load config
	viper.AutomaticEnv()
	viper.ReadInConfig()

	startDaemon()
}
