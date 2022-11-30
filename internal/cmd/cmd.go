package cmd

import (
	"flag"

	"github.com/galgotech/fhub-track/internal/log"
)

var logCmd = log.New("cmd")

type Cmd struct {
	Repository string
	Folder     string
}

func New() (*Cmd, int) {
	repository := flag.String("repository", "https://github.com/grafana/grafana.git", "repository")
	folder := flag.String("folder", "tmp/project", "folder out")
	help := flag.Bool("help", false, "Help")

	flag.Parse()

	cmd := &Cmd{
		Repository: *repository,
		Folder:     *folder,
	}

	logCmd.Debug("Arguments", "repository", cmd.Repository, "folder", cmd.Folder)

	if *repository == "" || *folder == "" {
		*help = true
	}

	if *help {
		flag.PrintDefaults()
		return nil, 1
	}

	return cmd, 0
}
