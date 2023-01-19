package track

import (
	"fmt"

	git "github.com/libgit2/git2go/v34"
)

func (t *Track) status() error {
	status, err := t.dstRepositoryStatus()
	if err != nil {
		return err
	}

	fmt.Println(status)

	return nil
}

func (t *Track) dstRepositoryStatus() (*git.StatusList, error) {
	status, err := t.dstRepository.StatusList(&git.StatusOptions{
		Show:  git.StatusShowIndexAndWorkdir,
		Flags: git.StatusOptIncludeUntracked | git.StatusOptIncludeIgnored,
	})
	if err != nil {
		return nil, err
	}

	return status, nil
}

func (t *Track) dstRepositoryStatusEntryCount() (int, error) {
	status, err := t.dstRepositoryStatus()
	if err != nil {
		return 0, err
	}

	c, err := status.EntryCount()
	if err != nil {
		return 0, err
	}

	return c, nil
}
