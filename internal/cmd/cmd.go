package cmd

import (
	"flag"

	"github.com/galgotech/fhub-track/internal/log"
)

var logCmd = log.New("cmd")

type Cmd struct {
	WorkTreeSrc string
	WorkTreeDst string

	Init                bool
	Status              bool
	Track               string
	TrackRename         string
	TrackIgnoreModified bool
	TrackUpdate         bool
}

func New() (*Cmd, int) {
	cmd := &Cmd{}

	flag.StringVar(&cmd.WorkTreeSrc, "work-tree-src", "", "Work tree src")
	flag.StringVar(&cmd.WorkTreeDst, "work-tree-dst", "", "Work tree dst")

	flag.BoolVar(&cmd.Init, "init", false, "Init")
	flag.StringVar(&cmd.Track, "track", "", "Track objects")
	flag.StringVar(&cmd.TrackRename, "track-rename", "", "Track rename object")
	flag.BoolVar(&cmd.TrackIgnoreModified, "ignore-modified", false, "Track ignore modified files")
	flag.BoolVar(&cmd.Status, "status", false, "Status objects")
	flag.BoolVar(&cmd.TrackUpdate, "track-update", false, "Update objects")

	help := flag.Bool("help", false, "Help")

	flag.Parse()

	logCmd.Debug("Arguments", "cmd", cmd)

	if cmd.WorkTreeSrc == "" {
		*help = true
	}

	if *help {
		flag.PrintDefaults()
		return nil, 1
	}

	return cmd, 0
}
