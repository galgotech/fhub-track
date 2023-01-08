package track

import (
	"errors"
	"fmt"

	"github.com/go-git/go-git/v5"
)

func (t *Track) trackRenameObject(trackRenameObject string) error {
	trackSrc, trackDst, err := splitTrackObject(trackRenameObject)
	if err != nil {
		return err
	}

	status, err := t.trackObjectsWorkTree.Status()
	if err != nil {
		return err
	}

	for key, status := range status {
		if status.Staging == git.Added {
			return errors.New("unable to track files because they were added to the stage area")
		} else if status.Worktree == git.Modified && key == trackSrc {
			return errors.New("unable to rename track files because they were modified in worktree")
		}
	}

	err = t.trackObjectsWorkTree.Filesystem.Rename(trackSrc, trackDst)
	if err != nil {
		return err
	}

	_, err = t.trackObjectsWorkTree.Remove(trackSrc)
	if err != nil {
		return err
	}

	_, err = t.trackObjectsWorkTree.Add(trackDst)
	if err != nil {
		return err
	}

	msg := fmt.Sprintf("rename:\n  %s", trackRenameObject)
	err = t.commit(msg)
	if err != nil {
		return err
	}

	return nil
}
