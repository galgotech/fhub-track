package track

import (
	"errors"
	"fmt"
	"os"
)

func (t *Track) trackRenameObject(oldObject string, newObject string) error {
	c, err := t.dstRepositoryStatusEntryCount()
	if err != nil {
		return err
	}

	if c > 0 {
		return errors.New("unable to track files because they were in stage")
	}

	err = os.Rename(oldObject, newObject)
	if err != nil {
		return err
	}

	index, err := t.dstRepository.Index()
	if err != nil {
		return err
	}

	err = index.RemoveByPath(oldObject)
	if err != nil {
		return err
	}

	err = index.AddByPath(newObject)
	if err != nil {
		return err
	}

	msg := fmt.Sprintf("rename:\n  %s -> %s", oldObject, newObject)

	treeOid, err := index.WriteTree()
	if err != nil {
		return err
	}

	tree, err := t.dstRepository.LookupTree(treeOid)
	if err != nil {
		return err
	}

	head, err := t.dstRepository.Head()
	if err != nil {
		return err
	}

	commitHead, err := t.dstRepository.LookupCommit(head.Target())
	if err != nil {
		return err
	}

	err = t.commit(msg, tree, commitHead)
	if err != nil {
		return err
	}

	return nil
}
