package track

import (
	"errors"
	"fmt"

	"github.com/go-git/go-git/v5"
)

func (t *Track) trackRenameObject(oldObject string, newObject string) error {
	status, err := t.dstWorkTree.Status()
	if err != nil {
		return err
	}

	for key, status := range status {
		if status.Staging == git.Added {
			return errors.New("unable to track files because they were added to the stage area")
		} else if status.Worktree == git.Modified && key == oldObject {
			return errors.New("unable to rename track files because they were modified in worktree")
		}
	}

	err = t.dstWorkTree.Filesystem.Rename(oldObject, newObject)
	if err != nil {
		return err
	}

	_, err = t.dstWorkTree.Remove(oldObject)
	if err != nil {
		return err
	}

	_, err = t.dstWorkTree.Add(newObject)
	if err != nil {
		return err
	}

	msg := fmt.Sprintf("rename:\n  %s -> %s", oldObject, newObject)
	err = t.commit(msg)
	if err != nil {
		return err
	}

	return nil
}
