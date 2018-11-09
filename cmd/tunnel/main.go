package main

import (
	"github.com/labstack/gommon/log"
	"tunnel/client/cmd"
)

func main() {
	log.SetHeader("${time_rfc3339} ${level}")
	cmd.Execute()
}
