package cmd

import (
	"github.com/briandowns/spinner"
	"github.com/spf13/viper"
	"io/ioutil"
	"net/rpc"
	"time"
)

var (
	s = spinner.New(spinner.CharSets[32], 100*time.Millisecond)
)

func getClient() (*rpc.Client, error) {
	addr, _ := ioutil.ReadFile(viper.GetString("daemon_addr"))
	return rpc.Dial("tcp", string(addr))
}

