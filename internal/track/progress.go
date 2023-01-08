package track

import "github.com/galgotech/fhub-track/internal/log"

type logGitProgess struct {
	log log.Logger
}

func (l *logGitProgess) Write(s []byte) (int, error) {
	return len(s), nil
}
