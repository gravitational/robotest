package main

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/gravitational/configure/cstrings"
	"github.com/gravitational/robotest/lib/robotest"
	"github.com/gravitational/trace"
	"gopkg.in/alecthomas/kingpin.v2"
)

func main() {
	if err := run(); err != nil {
		log.Errorf(trace.DebugReport(err))
		os.Exit(255)
	}
}

func run() error {
	args, _ := cstrings.SplitAt(os.Args, "--")

	var (
		app = kingpin.New("robotest", "")

		cstart       = app.Command("start", "start the test")
		cstartConfig = cstart.Flag("config", "json string with configuration").Default("{}").String()

		cstop       = app.Command("stop", "stop the test")
		cstopConfig = cstop.Flag("config", "json string with configuration").Default("{}").String()
	)

	cmd, err := app.Parse(args[1:])
	if err != nil {
		return trace.Wrap(err)
	}

	switch cmd {
	case cstart.FullCommand():
		return robotest.Start(*cstartConfig)
	case cstop.FullCommand():
		return robotest.Stop(*cstopConfig)
	}

	return nil
}
