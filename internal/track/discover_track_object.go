package track

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/utils/merkletrie"
)

type objectsTracked struct {
	commitSrc plumbing.Hash
	commitDst plumbing.Hash
	pathSrc   string
	deleted   bool
	repo      string
}

type mapObjectTrack = map[string]*objectsTracked

func (t *Track) searchAllTrackedObjects() (mapObjectTrack, error) {
	head, err := t.dstRepository.Head()
	if err != nil {
		return nil, err
	}

	return t.searchObjects(head.Hash())
}

func (t *Track) searchObjects(start plumbing.Hash) (mapObjectTrack, error) {
	tracked := &mapObjectTrack{}
	stackCommit := []plumbing.Hash{start}
	for len(stackCommit) > 0 {
		commit, err := t.dstRepository.CommitObject(stackCommit[0])
		if err != nil {
			return nil, err
		}

		parents := commit.ParentHashes
		err = t.commitIter(tracked, commit, parents)
		if err != nil {
			return nil, err
		}

		stackCommit = append(stackCommit[1:], parents...)
	}

	return *tracked, nil
}

func (t *Track) commitIter(tracked *mapObjectTrack, commit *object.Commit, parents []plumbing.Hash) error {
	if len(parents) > 0 {
		commitTree, err := t.dstRepository.TreeObject(commit.TreeHash)
		if err != nil {
			return err
		}

		commitParent, err := t.dstRepository.CommitObject(parents[0])
		if err != nil {
			return err
		}

		commitParentTree, err := t.dstRepository.TreeObject(commitParent.TreeHash)
		if err != nil {
			return err
		}

		diff, err := commitParentTree.Diff(commitTree)
		if err != nil {
			return err
		}

		for _, f := range diff {
			action, err := f.Action()
			if err != nil {
				return err
			}

			path := ""
			if action == merkletrie.Insert || action == merkletrie.Modify {
				path = f.To.Name
			} else if action == merkletrie.Delete {
				path = f.From.Name
			}

			if _, ok := (*tracked)[path]; !ok {
				(*tracked)[path] = &objectsTracked{
					deleted: action == merkletrie.Delete,
				}
			}
		}

	} else {
		files, err := commit.Files()
		if err != nil {
			return err
		}

		files.ForEach(func(f *object.File) error {
			if _, ok := (*tracked)[f.Name]; !ok {
				(*tracked)[f.Name] = &objectsTracked{
					deleted: false,
				}
			}
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
				objects := strings.Split(line, ":")
				if len(objects) != 2 {
					return fmt.Errorf("invalid line '%s'", line)
				}

				if objectTracked, ok := (*tracked)[objects[1]]; ok {
					if objectTracked.commitSrc.IsZero() {
						objectTracked.repo = repo
						objectTracked.commitSrc = hash
						objectTracked.commitDst = commit.Hash
						objectTracked.pathSrc = objects[0]
					}
				} else {
					return errors.New("path not found")
				}

			} else if lastKey == "move" {
				panic("implement move")
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

func filePatchStatus(from, to diff.File) (string, string, error) {
	path := ""
	status := ""

	if from == nil && to != nil {
		status = "add"
		path = to.Path()
	} else if from != nil && to == nil {
		status = "delete"
		path = from.Path()
	} else if from != nil && to != nil {
		path = to.Path()
		if from.Path() != to.Path() {
			status = "rename"
		} else {
			status = "change"
		}
	} else {
		return "", "", errors.New("fail get patches")
	}

	return path, status, nil
}

// func addTrack(tracked *mapObjectTracked, line, status, repo string, srcHash, dstHash plumbing.Hash) error {
// 	objects := strings.Split(line, ":")
// 	if len(objects) != 2 {
// 		return fmt.Errorf("invalid %s line '%s'", status, line)
// 	}

// 	prepareObjectTracked(tracked, dstHash, objects[1], status)

// 	(*tracked)[objects[0]].pairing = append((*tracked)[objects[0]].pairing, pairing{srcHash, dstHash})
// 	// (*tracked)[objects[0]].repos = repo

// 	return nil
// }

func prepareObjectTracked(tracked *mapObjectTrack, hash plumbing.Hash, path, status string) {
	// objectTracked, ok := (*tracked)[path]
	// if !ok {
	// 	objectTracked = &objectsTracked{
	// 		commits: map[plumbing.Hash][]string{},
	// 		pairing: []pairing{},
	// 	}
	// }

	// if _, ok := objectTracked.commits[hash]; !ok {
	// 	objectTracked.commits[hash] = []string{}
	// }

	// objectTracked.commits[hash] = append(objectTracked.commits[hash], status)

	// (*tracked)[path] = objectTracked
}
