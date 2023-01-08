package track

import (
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5"
)

func (t *Track) commit(msg string) error {
	remotes := []string{}
	for key, remote := range t.vendorConfig.Remotes {
		remotes = append(remotes, fmt.Sprintf("%s:%s", key, strings.Join(remote.URLs, ",")))
	}

	msg = fmt.Sprintf(
		"fhub-track\nrepo:\n  %s\nhash:\n  %s\n%s",
		strings.Join(remotes, "\n  "), t.vendorHash.Hash().String(), msg,
	)

	commitHash, err := t.trackObjectsWorkTree.Commit(msg, &git.CommitOptions{All: false})
	if err != nil {
		return err
	}

	logTrack.Debug("Commit message", "message", msg, "hash", commitHash.String())

	trackLog, err := t.trackObjects.Log(&git.LogOptions{})
	if err != nil {
		return err
	}

	commit, err := trackLog.Next()
	if err != nil {
		return err
	}
	fmt.Println(commit)

	return nil
}