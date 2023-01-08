package setting

import (
	"os"
)

type Setting struct {
	RootPath string
	SrcRepo  string
	DstRepo  string

	TrackObject string
}

func (s *Setting) Init() error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	s.RootPath = dir

	return nil
}

func New() (*Setting, error) {
	setting := &Setting{}

	err := setting.Init()
	if err != nil {
		return nil, err
	}

	return setting, nil
}
