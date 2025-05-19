package main

import (
	"github.com/oiler-backup/base/logger"
	"github.com/oiler-backup/cli/cmd"
)

func main() {
	log, err := logger.GetLogger(logger.PRODUCTION)
	if err != nil {
		panic(err)
	}
	cmd.Execute(log)
}
