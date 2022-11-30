package main

import (
	"os"

	"github.com/galgotech/fhub-track/internal/cmd"
	"github.com/galgotech/fhub-track/internal/setting"
	"github.com/galgotech/fhub-track/internal/track"
)

func main() {
	cmd, exit := cmd.New()
	if exit != 0 {
		os.Exit(exit)
	}

	setting := setting.New()

	os.Exit(track.Run(cmd, setting))
}
