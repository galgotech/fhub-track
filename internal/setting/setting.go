package setting

import (
	"os"

	"github.com/galgotech/fhub-track/internal/cmd"
)

type Setting struct {
	TrackFolder string
}

func (s *Setting) Init(cmd *cmd.Cmd) error {
	if _, err := os.Stat(s.TrackFolder); err != nil {
		os.Mkdir(s.TrackFolder, 0775)
	}

	return nil
}

func New(cmd *cmd.Cmd) (*Setting, error) {
	setting := &Setting{
		TrackFolder: ".fhub-track",
	}

	err := setting.Init(cmd)
	if err != nil {
		return nil, err
	}

	return setting, nil
}
