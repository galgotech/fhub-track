package utils

import git "github.com/libgit2/git2go/v34"

func StatusEntryCount(repo *git.Repository) (int, error) {
	status, err := Status(repo)
	if err != nil {
		return 0, err
	}

	c, err := status.EntryCount()
	if err != nil {
		return 0, err
	}

	return c, nil
}

func Status(repo *git.Repository) (*git.StatusList, error) {
	status, err := repo.StatusList(&git.StatusOptions{
		Show:  git.StatusShowIndexAndWorkdir,
		Flags: git.StatusOptIncludeUntracked | git.StatusOptIncludeIgnored,
	})
	if err != nil {
		return nil, err
	}

	return status, nil
}
