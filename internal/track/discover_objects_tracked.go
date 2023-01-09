package track

import (
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type hashObjectsTracked struct {
	srcCommit plumbing.Hash
	dstCommit plumbing.Hash
	srcObject string
	dstObject string
	repos     string
}

type listObjectTracked = map[string]*hashObjectsTracked

func (t *Track) searchAllTrackedObjects() (listObjectTracked, error) {
	return t.searchObjects(t.dstHead)
}

func (t *Track) searchObjects(start *plumbing.Reference) (listObjectTracked, error) {
	commit, err := t.dstRepository.CommitObject(start.Hash())
	if err != nil {
		return nil, err
	}

	tracked := &listObjectTracked{}
	trackedMoved := map[string]string{}

	stackHash := []*object.Commit{commit}
	for len(stackHash) > 0 {
		err = commitIter(tracked, trackedMoved, commit)
		if err != nil {
			return nil, err
		}

		stackHash = stackHash[1:]

		for _, hash := range commit.ParentHashes {
			parentCommit, err := t.dstRepository.CommitObject(hash)
			if err != nil {
				return nil, err
			}
			stackHash = append(stackHash, parentCommit)
		}
	}

	return *tracked, nil
}

func commitIter(tracked *listObjectTracked, trackedMoved map[string]string, commit *object.Commit) error {
	lines := strings.Split(commit.Message, "\n")
	if lines[0] == "fhub-track" {
		var repo string
		var hash plumbing.Hash

		lastKey := ""
		for _, line := range lines[1:] {
			line = strings.TrimSpace(line)

			if key, ok := parseMessageKey(line); ok {
				lastKey = key
			} else if lastKey == "repo" {
				repo = line
			} else if lastKey == "hash" {
				hash = plumbing.NewHash(line)
			} else if lastKey == "files" {
				objects := strings.Split(line, ":")
				srcObject := objects[0]
				dstObject := objects[1]

				if val, ok := trackedMoved[dstObject]; ok {
					srcObject = val
					delete(trackedMoved, dstObject)
				}
				trackedObject := &hashObjectsTracked{
					srcCommit: hash,
					dstCommit: commit.Hash,
					srcObject: srcObject,
					dstObject: dstObject,
					repos:     repo,
				}
				(*tracked)[dstObject] = trackedObject
				logTrack.Debug("tracked object", "object.srcObject", trackedObject.srcObject, "object.dstObject", trackedObject.dstObject)

			} else if lastKey == "move" {
				objects := strings.Split(line, ":")
				srcObject := objects[0]
				dstObject := objects[1]

				if _, ok := (*tracked)[dstObject]; !ok {
					if _, ok := trackedMoved[dstObject]; !ok {
						return fmt.Errorf("object already moved %s", dstObject)
					}
					trackedMoved[dstObject] = srcObject
				}
			}
		}
	}
	return nil
}

func parseMessageKey(line string) (string, bool) {
	if line == "repo:" || line == "hash:" || line == "files:" || line == "rename:" {
		return line[:len(line)-1], true
	}
	return "", false
}
