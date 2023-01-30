package utils

import (
	"fmt"
	"strings"

	git "github.com/libgit2/git2go/v34"
)

func Commit(repo *git.Repository, msg string, tree *git.Tree, parents ...*git.Commit) (*git.Oid, error) {
	remotesName, err := repo.Remotes.List()
	if err != nil {
		return nil, err
	}

	remotes := []string{}
	for _, remoteName := range remotesName {
		remote, err := repo.Remotes.Lookup(remoteName)
		if err != nil {
			return nil, err
		}

		remotes = append(remotes, fmt.Sprintf("%s:%s", remoteName, remote.Url()))
	}

	head, err := repo.Head()
	if err != nil {
		return nil, err
	}

	msg = fmt.Sprintf(
		"fhub-track\n\nrepo:\n  %s\nhash:\n  %s\n%s",
		strings.Join(remotes, "\n  "), head.Target().String(), msg,
	)

	signature, err := repo.DefaultSignature()
	if err != nil {
		return nil, err
	}

	oid, err := repo.CreateCommit("HEAD", signature, signature, msg, tree, parents...)
	if err != nil {
		return nil, err
	}

	return oid, nil
}

func CommitParents(commit *git.Commit) []*git.Commit {
	parentCount := commit.ParentCount()
	parents := make([]*git.Commit, parentCount)
	for i := uint(0); i < parentCount; i++ {
		parents[i] = commit.Parent(i)
	}
	return parents
}

func CommitFiles(commit *git.Commit) ([]*git.TreeEntry, error) {
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
