package cmd

import (
	"flag"

	"github.com/galgotech/fhub-track/internal/log"
)

var logCmd = log.New("cmd")

type Cmd struct {
	Repository string
	WorkTree   string

	Init        bool
	Status      bool
	Track       string
	TrackUpdate bool
}

func New() (*Cmd, int) {
	cmd := &Cmd{}

	flag.StringVar(&cmd.Repository, "repository", "", "repository to track")
	flag.StringVar(&cmd.WorkTree, "work-tree", "", "Work tree path")

	flag.BoolVar(&cmd.Init, "init", false, "Init")
	flag.StringVar(&cmd.Track, "track", "", "Track objects")
	flag.BoolVar(&cmd.Status, "status", false, "Status objects")
	flag.BoolVar(&cmd.TrackUpdate, "track-update", false, "Update objects")

	help := flag.Bool("help", false, "Help")

	flag.Parse()

	logCmd.Debug("Arguments", "cmd", cmd)

	if cmd.Repository == "" {
		*help = true
	}

	if *help {
		flag.PrintDefaults()
		return nil, 1
	}

	return cmd, 0
}
