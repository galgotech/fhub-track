package rename

import (
	"errors"
	"fmt"
	"os"

	"github.com/galgotech/fhub-track/internal/track/utils"
	git "github.com/libgit2/git2go/v34"
)

func New(src, dst *git.Repository) *Rename {
	return &Rename{src, dst}
}

type Rename struct {
	src, dst *git.Repository
}

func (t *Rename) Run(oldObject string, newObject string) error {
	c, err := utils.StatusEntryCount(t.dst)
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

	index, err := t.dst.Index()
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

	err = index.Write()
	if err != nil {
		return err
	}

	treeOid, err := index.WriteTree()
	if err != nil {
		return err
	}

	tree, err := t.dst.LookupTree(treeOid)
	if err != nil {
		return err
	}

	head, err := t.dst.Head()
	if err != nil {
		return err
	}

	commitHead, err := t.dst.LookupCommit(head.Target())
	if err != nil {
		return err
	}

	msg := fmt.Sprintf("rename:\n  %s -> %s", oldObject, newObject)
	_, err = utils.Commit(t.dst, msg, tree, commitHead)
	if err != nil {
		return err
	}

	return nil
}
