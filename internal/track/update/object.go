package update

import (
	"fmt"
	"strings"

	"github.com/galgotech/fhub-track/internal/track/utils"
	git "github.com/libgit2/git2go/v34"
)

type baseObject struct {
	// repo    string
	path   string
	commit string

	mode uint16
	blob *git.Oid
}

type head struct {
	baseObject
}

type object struct {
	baseObject

	link *object
	head *head
}

type listPathObject = []*object
type mapPathObject = map[string]*object
type mapCommitPath = map[string]mapPathObject

func (t *Update) MapObjects() (listPathObject, mapCommitPath, mapCommitPath, error) {
	head, err := t.dst.Head()
	if err != nil {
		return nil, nil, nil, err
	}

	commit, err := t.dst.LookupCommit(head.Target())
	if err != nil {
		return nil, nil, nil, err
	}

	objects := &mapPathObject{}
	commitsSrc := &mapCommitPath{}
	commitsDst := &mapCommitPath{}
	stackCommit := []*git.Commit{commit}
	for len(stackCommit) > 0 {
		commit := stackCommit[0]
		parents := utils.CommitParents(commit)
		err = t.commitIter(objects, commitsSrc, commitsDst, commit)
		if err != nil {
			return nil, nil, nil, err
		}

		stackCommit = append(stackCommit[1:], parents...)
	}

	listObject := []*object{}
	for _, object := range *objects {
		listObject = append(listObject, object)
	}

	return listObject, *commitsSrc, *commitsDst, nil
}

func (t *Update) commitIter(objects *mapPathObject, commitsSrc, commitsDst *mapCommitPath, commitDst *git.Commit) error {
	lines := strings.Split(strings.TrimSpace(commitDst.Message()), "\n")
	if len(lines) < 3 {
		return nil
	}

	if lines[0] == "fhub-track" || lines[1] == "" {
		// var repo string
		var commitOidSrc string
		commitOidDst := commitDst.Id().String()

		lastKey := ""
		for _, line := range lines[2:] {
			line = strings.TrimSpace(line)

			if key, ok := parseMessageKey(line); ok {
				lastKey = key
			} else if lastKey == "repo" {
				// repo = line
			} else if lastKey == "hash" {
				var err error
				commitOidSrc = line
				if err != nil {
					return err
				}

			} else if lastKey == "files" {
				path := strings.Split(line, ":")
				if len(path) != 2 {
					return fmt.Errorf("invalid line '%s'", line)
				}

				if _, ok := (*commitsSrc)[commitOidSrc]; !ok {
					(*commitsSrc)[commitOidSrc] = map[string]*object{}
				}

				if _, ok := (*commitsDst)[commitOidDst]; !ok {
					(*commitsDst)[commitOidDst] = map[string]*object{}
				}

				// Add only the first time path find
				if _, ok := (*objects)[path[1]]; !ok {
					if _, ok := (*commitsSrc)[commitOidSrc][path[1]]; !ok {
						objSrc := &object{
							baseObject: baseObject{
								commit: commitOidSrc,
								path:   path[0],
								// repo:      repo,
							},
							head: &head{},
						}

						objDst := &object{
							baseObject: baseObject{
								commit: commitOidDst,
								path:   path[1],
								// repo:      repo,
							},
							head: &head{},
						}

						objSrc.link = objDst
						objDst.link = objSrc

						(*objects)[path[1]] = objDst
						(*commitsSrc)[commitOidSrc][path[1]] = objSrc
						(*commitsDst)[commitOidDst][path[1]] = objDst
					}
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
