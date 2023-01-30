package update

import (
	"github.com/galgotech/fhub-track/internal/track/utils"
	git "github.com/libgit2/git2go/v34"
)

type patchByPath struct {
	commit *git.Commit
	patch  *git.Patch
}

func searchPatchByPath(repository *git.Repository, path string, start, stop *git.Oid) ([]patchByPath, error) {
	commit, err := repository.LookupCommit(start)
	if err != nil {
		return nil, err
	}

	patchs := []patchByPath{}
	for commit != nil {
		var parentTree *git.Tree
		parents := utils.CommitParents(commit)
		if len(parents) > 0 {
			parentTree, err = parents[0].Tree()
			if err != nil {
				return nil, err
			}
		}

		tree, err := commit.Tree()
		if err != nil {
			return nil, err
		}

		diff, err := repository.DiffTreeToTree(parentTree, tree, &git.DiffOptions{
			Flags:        git.DiffPatience,
			ContextLines: 5,
			OldPrefix:    "a",
			NewPrefix:    "b",
		})
		if err != nil {
			return nil, err
		}

		c, err := diff.NumDeltas()
		if err != nil {
			return nil, err
		}

		for i := 0; i < c; i++ {
			delta, err := diff.Delta(i)
			if err != nil {
				return nil, err
			}

			if delta.NewFile.Path == path {
				patch, err := diff.Patch(i)
				if err != nil {
					return nil, err
				}
				patchs = append(patchs, patchByPath{commit: commit, patch: patch})
			}
		}

		if commit.Id().Equal(stop) {
			commit = nil
		} else if len(parents) > 0 {
			commit = parents[0]
		}
	}

	for i, j := 0, len(patchs)-1; i < j; i, j = i+1, j-1 {
		patchs[i], patchs[j] = patchs[j], patchs[i]
	}
	return patchs, nil
}
