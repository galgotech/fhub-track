package track

import (
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type hashObjectsTracked struct {
	srcCommit plumbing.Hash
	dstCommit plumbing.Hash
	objects   []string
}

type listObjectTracked = []*hashObjectsTracked

func (t *Track) searchTrackedObjects() (listObjectTracked, error) {
	log, err := t.dstRepository.Log(&git.LogOptions{})
	if err != nil {
		return nil, err
	}

	parseMessageKey := func(line string) (string, bool) {
		if line == "repo:" || line == "hash:" || line == "files:" || line == "rename:" {
			return line[:len(line)-1], true
		}
		return "", false
	}

	tracked := listObjectTracked{}

	err = log.ForEach(func(commit *object.Commit) error {
		lines := strings.Split(commit.Message, "\n")
		if lines[0] == "fhub-track" {
			var repos, objects []string
			var hash string

			lastKey := ""
			for _, line := range lines[1:] {
				line = strings.TrimSpace(line)

				if key, ok := parseMessageKey(line); ok {
					lastKey = key
				} else if lastKey == "repo" {
					repos = append(repos, line)
				} else if lastKey == "hash" {
					hash = line
				} else if lastKey == "files" {
					objects = append(objects, line)
				} else if lastKey == "rename" {
					objects = strings.Split(":", line)
				}
			}

			tracked = append(tracked, &hashObjectsTracked{
				srcCommit: plumbing.NewHash(hash),
				dstCommit: commit.Hash,
				objects:   objects,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return tracked, nil
}
