package track

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/diff"
	git "github.com/libgit2/git2go/v34"
)

type objectsTracked struct {
	commitSrc *git.Oid
	commitDst *git.Oid
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

	return t.searchObjects(head.Target())
}

func (t *Track) searchObjects(start *git.Oid) (mapObjectTrack, error) {
	commit, err := t.dstRepository.LookupCommit(start)
	if err != nil {
		return nil, err
	}

	tracked := &mapObjectTrack{}
	stackCommit := []*git.Commit{commit}
	for len(stackCommit) > 0 {
		commit := stackCommit[0]
		parents := commitParents(commit)
		err = t.commitIter(tracked, commit, parents)
		if err != nil {
			return nil, err
		}

		stackCommit = append(stackCommit[1:], parents...)
	}

	return *tracked, nil
}

func (t *Track) commitIter(tracked *mapObjectTrack, commit *git.Commit, parents []*git.Commit) error {
	commitTree, err := commit.Tree()
	if err != nil {
		return err
	}

	var commitParentTree *git.Tree
	if len(parents) > 0 {
		commitParentTree, err = parents[0].Tree()
		if err != nil {
			return err
		}
	}

	diffTree, err := t.dstRepository.DiffTreeToTree(commitParentTree, commitTree, &git.DiffOptions{
		Flags: git.DiffNormal,
	})
	if err != nil {
		return err
	}

	err = diffTree.FindSimilar(&git.DiffFindOptions{
		Flags: git.DiffFindRenames | git.DiffFindCopies | git.DiffFindForUntracked,
	})
	if err != nil {
		return err
	}

	nDetals, err := diffTree.NumDeltas()
	if err != nil {
		return err
	}

	for i := 0; i < nDetals; i++ {
		delta, err := diffTree.GetDelta(i)
		if err != nil {
			return err
		}

		path := delta.NewFile.Path
		if _, ok := (*tracked)[path]; !ok {
			(*tracked)[path] = &objectsTracked{
				deleted: delta.Status == git.DeltaDeleted,
			}
		}
	}

	lines := strings.Split(strings.TrimSpace(commit.Message()), "\n")
	if len(lines) < 3 {
		return nil
	}

	if lines[0] == "fhub-track" || lines[1] == "" {
		var repo string
		var oid *git.Oid

		lastKey := ""
		for _, line := range lines[2:] {
			line = strings.TrimSpace(line)

			if key, ok := parseMessageKey(line); ok {
				lastKey = key
			} else if lastKey == "repo" {
				repo = line
			} else if lastKey == "hash" {
				var err error
				oid, err = git.NewOid(line)
				if err != nil {
					return err
				}
			} else if lastKey == "files" {
				objects := strings.Split(line, ":")
				if len(objects) != 2 {
					return fmt.Errorf("invalid line '%s'", line)
				}

				if objectTracked, ok := (*tracked)[objects[1]]; ok {
					if objectTracked.commitSrc == nil {
						objectTracked.repo = repo
						objectTracked.commitSrc = oid
						objectTracked.commitDst = commit.Id()
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
