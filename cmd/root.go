package cmd

import (
	"fmt"
	"net/http"
	"os"

	"github.com/go-resty/resty"
	"github.com/labstack/gommon/log"
	"github.com/labstack/tunnel"
	"github.com/labstack/tunnel/util"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	err        error
	configFile string
	name       string
	tcp        bool
	tls        bool
	user       string
	rootCmd    = &cobra.Command{
		Use:   "tunnel",
		Short: "Tunnel lets you expose local servers to the internet securely",
		Long:  ``,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			c := &tunnel.Config{
				Host:       "labstack.me:22",
				RemoteHost: "0.0.0.0",
				RemotePort: 80,
			}
			e := new(tunnel.Error)

			if name != "" {
				key := viper.GetString("api_key")
				if key == "" {
					log.Fatalf("Failed to find api key in the config")
				}

				// Find config
				res, err := resty.R().
					SetAuthToken(key).
					SetHeader("Content-Type", "application/json").
					SetResult(c).
					SetError(e).
					SetHeader("User-Agent", "labstack/tunnel").
					Get(fmt.Sprintf("https://api.labstack.com/tunnel/configs/%s", name))
				if err != nil {
					log.Fatalf("Failed to the find tunnel: %v", err)
				} else if res.StatusCode() != http.StatusOK {
					log.Fatalf("Failed to the find tunnel: %s", e.Message)
				}
				if c.Protocol == "tcp" {
					tcp = true
				} else if c.Protocol == "tls" {
					tls = true
				}

				user = fmt.Sprintf("key=%s,name=%s", key, name)
				c.Host += ":22"
			} else if tls {
				user = "tls=true"
			}

			c.User = user
			c.TargetHost, c.TargetPort, err = util.SplitHostPort(args[0])
			if err != nil {
				log.Fatalf("Failed to parse target address: %v", err)
			}
			if tcp || tls {
				c.RemotePort = 0
			}
			tunnel.Create(c)
		},
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "config file (default is $HOME/.tunnel.yaml)")
	rootCmd.PersistentFlags().StringVarP(&name, "name", "n", "", "config name from the dashboard")
	rootCmd.PersistentFlags().BoolVarP(&tcp, "tcp", "", false, "tcp tunnel")
	rootCmd.PersistentFlags().BoolVarP(&tls, "tls", "", false, "tls tunnel")
}

// initConfig reads in config file and ENV variables if set
func initConfig() {
	if configFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(configFile)
	} else {
		// Find home directory
		home, err := homedir.Dir()
		if err != nil {
			log.Fatalf("Failed to find the home directory: %v", err)
		}

		// Search config in home directory with name ".tunnel" (without extension)
		viper.AddConfigPath(home)
		viper.SetConfigName(".tunnel")
	}
	viper.AutomaticEnv() // Read in environment variables that match
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}
}
