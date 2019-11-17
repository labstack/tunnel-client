package cmd

import (
	"fmt"
	"github.com/briandowns/spinner"
	"github.com/spf13/viper"
	"io/ioutil"
	"net/rpc"
	"os"
	"time"
)

var (
	s = spinner.New(spinner.CharSets[32], 50*time.Millisecond)
)

func exit(i interface{}) {
	fmt.Println(i)
	os.Exit(1)
}

func getClient() (*rpc.Client, error) {
	addr, _ := ioutil.ReadFile(viper.GetString("daemon_addr"))
	return rpc.Dial("tcp", string(addr))
}
