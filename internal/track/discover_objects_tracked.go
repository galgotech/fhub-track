package track

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type pairing struct {
	src plumbing.Hash
	dst plumbing.Hash
}

type objectsTracked struct {
	commits map[plumbing.Hash][]string
	pairing []pairing
	moved   []string
	repos   string
}

type mapObjectTracked = map[string]*objectsTracked

func (t *Track) searchAllTrackedObjects() (mapObjectTracked, error) {
	return t.searchObjects(t.dstHead)
}

func (t *Track) searchObjects(start *plumbing.Reference) (mapObjectTracked, error) {
	commit, err := t.dstRepository.CommitObject(start.Hash())
	if err != nil {
		return nil, err
	}

	tracked := &mapObjectTracked{}
	stackCommit := []*object.Commit{commit}
	for len(stackCommit) > 0 {
		commit = stackCommit[0]
		parents := make([]*object.Commit, len(commit.ParentHashes))
		for i, hash := range commit.ParentHashes {
			commit, err := t.dstRepository.CommitObject(hash)
			if err != nil {
				return nil, err
			}
			parents[i] = commit
		}
		stackCommit = append(stackCommit[1:], parents...)

		err = commitIter(tracked, commit, parents)
		if err != nil {
			return nil, err
		}
	}

	return *tracked, nil
}

func commitIter(tracked *mapObjectTracked, commit *object.Commit, parents []*object.Commit) error {
	if len(parents) > 0 {
		patch, err := parents[0].Patch(commit)
		if err != nil {
			return err
		}

		filesPatches := patch.FilePatches()
		for _, f := range filesPatches {
			pathFrom, pathTo, status, err := filePatchStatus(f.Files())
			if err != nil {
				return err
			}

			prepareObjectTracked(tracked, commit.Hash, pathFrom, status)
			prepareObjectTracked(tracked, commit.Hash, pathTo, status)
		}
	} else {
		files, err := commit.Files()
		if err != nil {
			return err
		}

		files.ForEach(func(f *object.File) error {
			prepareObjectTracked(tracked, commit.Hash, f.Name, "add")
			return nil
		})
	}

	lines := strings.Split(strings.TrimSpace(commit.Message), "\n")
	if len(lines) < 3 {
		return nil
	}

	if lines[0] == "fhub-track" || lines[1] == "" {
		var repo string
		var hash plumbing.Hash

		lastKey := ""
		for _, line := range lines[2:] {
			line = strings.TrimSpace(line)

			if key, ok := parseMessageKey(line); ok {
				lastKey = key
			} else if lastKey == "repo" {
				repo = line
			} else if lastKey == "hash" {
				hash = plumbing.NewHash(line)
			} else if lastKey == "files" {
				err := addTrack(tracked, line, "track", repo, hash, commit.Hash)
				if err != nil {
					return err
				}

			} else if lastKey == "move" {
				panic("implement move")
				err := addTrack(tracked, line, "move", repo, hash, commit.Hash)
				if err != nil {
					return err
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

func filePatchStatus(from, to diff.File) (string, string, string, error) {
	pathFrom := ""
	pathTo := ""
	status := ""

	if from == nil && to != nil {
		status = "add"
		pathTo = to.Path()
		pathFrom = to.Path()
	} else if from != nil && to == nil {
		status = "delete"
		pathFrom = from.Path()
		pathTo = from.Path()
	} else if from != nil && to != nil {
		pathFrom = from.Path()
		pathTo = from.Path()
		if from.Path() != to.Path() {
			status = "rename"
		} else {
			status = "change"
		}
	} else {
		return "", "", "", errors.New("fail get patches")
	}

	return pathFrom, pathTo, status, nil
}

func addTrack(tracked *mapObjectTracked, line, status, repo string, srcHash, dstHash plumbing.Hash) error {
	objects := strings.Split(line, ":")
	if len(objects) != 2 {
		return fmt.Errorf("invalid %s line '%s'", status, line)
	}

	prepareObjectTracked(tracked, dstHash, objects[0], status)
	prepareObjectTracked(tracked, dstHash, objects[1], status)

	(*tracked)[objects[0]].pairing = append((*tracked)[objects[0]].pairing, pairing{srcHash, dstHash})
	(*tracked)[objects[0]].repos = repo

	return nil
}

func prepareObjectTracked(tracked *mapObjectTracked, hash plumbing.Hash, path, status string) {
	objectTracked, ok := (*tracked)[path]
	if !ok {
		objectTracked = &objectsTracked{
			commits: map[plumbing.Hash][]string{},
			moved:   []string{},
			pairing: []pairing{},
		}
	}

	if commits, ok := objectTracked.commits[hash]; ok {
		if len(commits) == 0 || commits[len(commits)-1] != status {
			objectTracked.commits[hash] = append(objectTracked.commits[hash], status)
		}
	} else {
		objectTracked.commits[hash] = []string{status}
	}

	if len(objectTracked.moved) == 0 || objectTracked.moved[len(objectTracked.moved)-1] != path {
		objectTracked.moved = append(objectTracked.moved, path)
	}

	(*tracked)[path] = objectTracked
}
