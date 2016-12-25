package main

import (
	"os"
	"os/signal"
	"path/filepath"

	"chill/command"
	"chill/config"
	"chill/runner"
	log "chill/util"
)

func main() {
	conf := config.GetConfigs()

	abspath, _ := filepath.Abs(conf.Directory)

	patterns := conf.Patterns
	cmd := command.NewCommand(conf.Command)

	r := runner.NewRunner(abspath, patterns, cmd)

	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt)

		/// Block until a signal is received.
		s := <-ch
		log.Info("Got signal :%s", s.String())
		r.Exit()
	}()
	r.Start()
}
