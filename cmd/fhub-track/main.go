package main

import (
	"os"

	"github.com/galgotech/fhub-track/internal/cmd"
	"github.com/galgotech/fhub-track/internal/setting"
)

func main() {
	setting, err := setting.New()
	if err != nil {
		os.Exit(1)
	}

	err = cmd.New(setting)
	if err != nil {
		os.Exit(1)
	}

	// os.Exit(track.Run(cmd))
}
