package main

import (
	"github.com/labstack/gommon/log"
	"github.com/labstack/tunnel/cmd"
)

func main() {
	log.SetHeader("${time_rfc3339} ${level}")
	cmd.Execute()
}
