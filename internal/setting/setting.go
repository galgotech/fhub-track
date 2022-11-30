package setting

import "os"

type Setting struct {
	TrackFolder string
}

func (s *Setting) Init() {
	if _, err := os.Stat(s.TrackFolder); err != nil {
		os.Mkdir(s.TrackFolder, 0775)
	}
}

func New() *Setting {
	setting := &Setting{
		TrackFolder: ".fhub-track",
	}

	setting.Init()

	return setting
}
