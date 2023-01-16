package track

import (
	"fmt"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type commitLink struct {
	commit *object.Commit
	next   *commitLink
	prev   *commitLink
}

type pathPatch struct {
	commit *object.Commit
	patch  diff.Patch
}

func (l *commitLink) ForEach(f func(*object.Commit, *object.Commit) error) {
	start := l.prev
	for start != nil {
		f(start.next.commit, start.commit)
		start = start.prev
	}
}

type ChangeByPath struct {
	head       *plumbing.Reference
	repository *git.Repository
	stop       plumbing.Hash
	link       *commitLink
	paths      map[string][]pathPatch
}

func (t *ChangeByPath) threeCommit(stop plumbing.Hash) error {
	hash := t.head.Hash()
	commit, err := t.repository.CommitObject(hash)
	if err != nil {
		return err
	}

	t.link = &commitLink{
		commit: commit,
	}

	stackCommit := []plumbing.Hash{hash}
	for stop != hash {
		hash = stackCommit[0]
		commit, err := t.repository.CommitObject(hash)
		if err != nil {
			return err
		}

		t.link.next = &commitLink{
			commit: commit,
			next:   nil,
			prev:   t.link,
		}
		t.link = t.link.next

		stackCommit = append(stackCommit[1:], commit.ParentHashes...)
	}

	return nil
}

func (t *ChangeByPath) findPathChange() {
	t.link.ForEach(func(c1, c2 *object.Commit) error {
		patch, err := c1.Patch(c2)
		if err != nil {
			return err
		}

		for _, f := range patch.FilePatches() {
			from, to := f.Files()
			if from != nil && to != nil {
				t.addPathChange(from.Path(), c1, patch)
			} else if from == nil && to != nil {
				t.addPathChange(to.Path(), c1, patch)
			} else if from != nil && to != nil {
				if from.Path() == to.Path() {
					t.addPathChange(from.Path(), c1, patch)
				} else {
					panic("rename not implemented")
				}
			}
		}

		for k := range t.paths {
			fmt.Println(k)
		}

		return nil
	})
}

func (t *ChangeByPath) addPathChange(path string, commit *object.Commit, patch diff.Patch) {
	if _, ok := t.paths[path]; !ok {
		t.paths[path] = []pathPatch{}
	}

	t.paths[path] = append(t.paths[path], pathPatch{
		commit: commit,
		patch:  patch,
	})
}

func (t *Track) trackUpdate() error {
	trackedObjects, err := t.searchAllTrackedObjects()
	if err != nil {
		return err
	}

	for path, detail := range trackedObjects {
		fmt.Println(path)
		for _, pair := range detail.pairing {
			changeByPath := ChangeByPath{
				head:       t.srcHead,
				repository: t.srcRepository,
				stop:       pair.src,
				paths:      map[string][]pathPatch{},
			}

			err := changeByPath.threeCommit(pair.src)
			if err != nil {
				return err
			}

			changeByPath.findPathChange()

			// linkSrc.ForEach(func(commit *object.Commit) {
			// 	fmt.Println(commit.Hash.String())
			// })

			commitSrc, err := t.srcRepository.CommitObject(pair.src)
			if err != nil {
				return err
			}

			patch, err := commitSrc.Patch(changeByPath.link.prev.commit)
			if err != nil {
				return err
			}
			fmt.Println(patch.String())

			os.Exit(0)

			commitDst, err := t.dstRepository.CommitObject(pair.dst)
			if err != nil {
				return err
			}

			fmt.Println(commitSrc.Hash.String(), commitDst.Hash.String())

		}

		os.Exit(0)

		// srcContent, err := getCommitFileContent(t.srcRepository, tracked.srcCommit, tracked.srcObject)
		// if err != nil {
		// 	if errors.Is(err, object.ErrFileNotFound) {
		// 		// TODO: go-git random issue. Random not found some file
		// 		logTrack.Warn("File not found in src repository commit", "error", err.Error(), "hash", tracked.srcCommit.String(), "object", tracked.srcObject)
		// 		continue
		// 	} else {
		// 		logTrack.Error("Read file contents src repository", "error", err.Error(), "hash", tracked.srcCommit.String(), "object", tracked.srcObject)
		// 		return err
		// 	}
		// }

		// dstContent, err := getCommitFileContent(t.dstRepository, tracked.dstCommit, tracked.dstObject)
		// if err != nil {
		// 	if errors.Is(err, object.ErrFileNotFound) {
		// 		// TODO: go-git random issue. Random not found some file
		// 		logTrack.Warn("File not found in dst repository commit", "error", err.Error(), "hash", tracked.dstCommit.String(), "object", tracked.dstObject)
		// 		continue
		// 	} else {
		// 		logTrack.Error("Read file contents dst repository", "error", err.Error(), "hash", tracked.dstCommit.String(), "object", tracked.dstObject)
		// 		return err
		// 	}
		// }

		// if path == "pkg/services/hooks/hooks.go" {
		// 	for hash, status := range detail.commits {
		// 		for _, s := range status {
		// 			fmt.Println(path, hash.String(), s)
		// 		}
		// 	}
		// 	for src, dst := range detail.pairing {
		// 		fmt.Println(src, dst)
		// 	}
		// }
	}

	fmt.Println(len(trackedObjects))

	return nil

	for _, tracked := range trackedObjects {

		fmt.Println(tracked)
		fmt.Println("-----------")
		continue

		// srcContent, err := getCommitFileContent(t.srcRepository, tracked.srcCommit, tracked.srcObject)
		// if err != nil {
		// 	if errors.Is(err, object.ErrFileNotFound) {
		// 		// TODO: go-git random issue. Random not found some file
		// 		logTrack.Warn("File not found in src repository commit", "error", err.Error(), "hash", tracked.srcCommit.String(), "object", tracked.srcObject)
		// 		continue
		// 	} else {
		// 		logTrack.Error("Read file contents src repository", "error", err.Error(), "hash", tracked.srcCommit.String(), "object", tracked.srcObject)
		// 		return err
		// 	}
		// }

		// dstContent, err := getCommitFileContent(t.dstRepository, tracked.dstCommit, tracked.dstObject)
		// if err != nil {
		// 	if errors.Is(err, object.ErrFileNotFound) {
		// 		// TODO: go-git random issue. Random not found some file
		// 		logTrack.Warn("File not found in dst repository commit", "error", err.Error(), "hash", tracked.dstCommit.String(), "object", tracked.dstObject)
		// 		continue
		// 	} else {
		// 		logTrack.Error("Read file contents dst repository", "error", err.Error(), "hash", tracked.dstCommit.String(), "object", tracked.dstObject)
		// 		return err
		// 	}
		// }

		// contentDiff := diff.Strings(srcContent, dstContent)
		// if len(contentDiff) > 0 {
		// 	fmt.Println(contentDiff)
		// 	break
		// }
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
