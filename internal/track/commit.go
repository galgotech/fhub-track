package track

import (
	"fmt"
	"strings"

	git "github.com/libgit2/git2go/v34"
)

func (t *Track) commit(msg string, tree *git.Tree, parents ...*git.Commit) error {
	remotesName, err := t.srcRepository.Remotes.List()
	if err != nil {
		return err
	}

	remotes := []string{}
	for _, remoteName := range remotesName {
		remote, err := t.srcRepository.Remotes.Lookup(remoteName)
		if err != nil {
			return err
		}

		remotes = append(remotes, fmt.Sprintf("%s:%s", remoteName, remote.Url()))
	}

	head, err := t.srcRepository.Head()
	if err != nil {
		return err
	}

	msg = fmt.Sprintf(
		"fhub-track\n\nrepo:\n  %s\nhash:\n  %s\n%s",
		strings.Join(remotes, "\n  "), head.Target().String(), msg,
	)

	signature, err := t.dstRepository.DefaultSignature()
	if err != nil {
		return err
	}

	oid, err := t.dstRepository.CreateCommit("HEAD", signature, signature, msg, tree, parents...)
	if err != nil {
		return err
	}

	logTrack.Debug("Commit message", "message", msg, "hash", oid.String())

	// trackLog, err := t.dstRepository.Log(&git.LogOptions{})
	// if err != nil {
	// 	return err
	// }

	// commit, err := trackLog.Next()
	// if err != nil {
	// 	return err
	// }
	// fmt.Println(commit)

	return nil
}

func commitParents(commit *git.Commit) []*git.Commit {
	parentCount := commit.ParentCount()
	parents := make([]*git.Commit, parentCount)
	for i := uint(0); i < parentCount; i++ {
		parents[i] = commit.Parent(i)
	}
	return parents
}

func commitFiles(commit *git.Commit) ([]*git.TreeEntry, error) {
	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	c := tree.EntryCount()
	entries := []*git.TreeEntry{}
	for i := uint64(0); i < c; i++ {
		entry := tree.EntryByIndex(i)
		entries = append(entries, entry)
	}
	return entries, nil
}
