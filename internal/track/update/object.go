package update

import (
	"errors"
	"fmt"
	"strings"

	"github.com/galgotech/fhub-track/internal/track/utils"
	git "github.com/libgit2/git2go/v34"
)

type object struct {
	commitSrc *git.Oid
	commitDst *git.Oid
	pathSrc   string
	deleted   bool
	repo      string
}

type mapObject = map[string]*object

func (t *Update) SearchObjects() (mapObject, error) {
	head, err := t.dst.Head()
	if err != nil {
		return nil, err
	}

	commit, err := t.dst.LookupCommit(head.Target())
	if err != nil {
		return nil, err
	}

	tracked := &mapObject{}
	stackCommit := []*git.Commit{commit}
	for len(stackCommit) > 0 {
		commit := stackCommit[0]
		parents := utils.CommitParents(commit)
		err = t.commitIter(tracked, commit, parents)
		if err != nil {
			return nil, err
		}

		stackCommit = append(stackCommit[1:], parents...)
	}

	for path, track := range *tracked {
		if track.repo == "" {
			delete(*tracked, path)
		}
	}

	return *tracked, nil
}

func (t *Update) commitIter(tracked *mapObject, commit *git.Commit, parents []*git.Commit) error {
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

	diffTree, err := t.dst.DiffTreeToTree(commitParentTree, commitTree, &git.DiffOptions{
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
			(*tracked)[path] = &object{
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
