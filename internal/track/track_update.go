package track

import (
	"errors"
	"fmt"

	"github.com/galgotech/gotools/diff"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func (t *Track) trackUpdate() error {
	trackedObjects, err := t.searchAllTrackedObjects()
	if err != nil {
		return err
	}

	for _, tracked := range trackedObjects {
		srcContent, err := getCommitFileContent(t.srcRepository, tracked.srcCommit, tracked.srcObject)
		if err != nil {
			if errors.Is(err, object.ErrFileNotFound) {
				// TODO: go-git random issue. Random not found some file
				logTrack.Warn("File not found in src repository commit", "error", err.Error(), "hash", tracked.srcCommit.String(), "object", tracked.srcObject)
				continue
			} else {
				logTrack.Error("Read file contents src repository", "error", err.Error(), "hash", tracked.srcCommit.String(), "object", tracked.srcObject)
				return err
			}
		}

		dstContent, err := getCommitFileContent(t.dstRepository, tracked.dstCommit, tracked.dstObject)
		if err != nil {
			if errors.Is(err, object.ErrFileNotFound) {
				// TODO: go-git random issue. Random not found some file
				logTrack.Warn("File not found in dst repository commit", "error", err.Error(), "hash", tracked.dstCommit.String(), "object", tracked.dstObject)
				continue
			} else {
				logTrack.Error("Read file contents dst repository", "error", err.Error(), "hash", tracked.dstCommit.String(), "object", tracked.dstObject)
				return err
			}
		}

		contentDiff := diff.Strings(srcContent, dstContent)
		if len(contentDiff) > 0 {
			fmt.Println(contentDiff)
			break
		}
	}

	return nil
}

func getCommitFileContent(repository *git.Repository, hash plumbing.Hash, path string) (string, error) {
	c, err := repository.CommitObject(hash)
	if err != nil {
		return "", err
	}

	f, err := c.File(path)
	if err != nil {
		return "", err
	}

	content, err := f.Contents()
	if err != nil {
		return "", err
	}

	return content, nil
}
