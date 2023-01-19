package track

import (
	git "github.com/libgit2/git2go/v34"
)

type pathChunk struct {
	hash  *git.Oid
	delta git.DiffDelta
	patch *git.Patch
}

var pathChangesSrc = make(map[string][]pathChunk, 0)
var pathChangesDst = make(map[string][]pathChunk, 0)

func searchChanges(repository *git.Repository, path string, start, stop *git.Oid) ([]pathChunk, error) {
	commit, err := repository.LookupCommit(start)
	if err != nil {
		return nil, err
	}

	changes := []pathChunk{}
	for commit != nil {
		var parentTree *git.Tree
		parents := commitParents(commit)
		if len(parents) > 0 {
			parentTree, err = parents[0].Tree()
			if err != nil {
				return nil, err
			}
		}

		tree, err := commit.Tree()
		if err != nil {
			return nil, err
		}

		diff, err := repository.DiffTreeToTree(parentTree, tree, &git.DiffOptions{
			Flags: git.DiffNormal,
		})
		if err != nil {
			return nil, err
		}

		c, err := diff.NumDeltas()
		if err != nil {
			return nil, err
		}
		for i := 0; i < c; i++ {
			delta, err := diff.Delta(i)
			if err != nil {
				return nil, err
			}

			if delta.NewFile.Path == path {
				patch, err := diff.Patch(i)
				if err != nil {
					return nil, err
				}
				changes = append(changes, pathChunk{hash: start, delta: delta, patch: patch})
			}
		}

		if commit.Id().Equal(stop) {
			commit = nil
		} else if len(parents) > 0 {
			commit = parents[0]
		}
	}

	return changes, nil
}

func (t *Track) trackUpdate() error {
	srcHead, err := t.srcRepository.Head()
	if err != nil {
		return err
	}

	dstHead, err := t.dstRepository.Head()
	if err != nil {
		return err
	}

	srcHeadHash := srcHead.Target()
	dstHeadHash := dstHead.Target()

	trackObjects, err := t.searchAllTrackedObjects()
	if err != nil {
		return err
	}

	for path, trackObject := range trackObjects {
		// if path != "diff/diff.go" {
		// 	continue
		// }

		logTrack.Debug("start update", "path", path)
		if trackObject.commitSrc == srcHeadHash {
			logTrack.Debug("path alredy update", "path", path)
		}

		if !trackObject.commitSrc.Equal(srcHeadHash) {
			logTrack.Debug("start repository src", "commitSrc", trackObject.commitSrc.String(), "srcHeadHash", srcHeadHash.String())
			changes, err := searchChanges(t.srcRepository, path, srcHeadHash, trackObject.commitSrc)
			if err != nil {
				return err
			}
			pathChangesSrc[path] = changes
		}

		if !dstHeadHash.Equal(trackObject.commitDst) {
			logTrack.Debug("start repository dst", "commitDst", trackObject.commitDst.String(), "dstHeadHash", dstHeadHash.String())
			changes, err := searchChanges(t.dstRepository, path, dstHeadHash, trackObject.commitDst)
			if err != nil {
				return err
			}
			pathChangesDst[path] = changes
		}

		if len(pathChangesSrc[path]) > 0 {
			ps := pathChangesSrc[path]
			for i, j := 0, len(ps)-1; i < j; i, j = i+1, j-1 {
				ps[i], ps[j] = ps[j], ps[i]
			}

			for _, p := range ps {
				_, err := p.patch.String()
				if err != nil {
					return err
				}
				// fmt.Println(s)
				// break
			}
		}

		if len(pathChangesDst[path]) > 0 {
			ps := pathChangesDst[path]
			for i, j := 0, len(ps)-1; i < j; i, j = i+1, j-1 {
				ps[i], ps[j] = ps[j], ps[i]
			}

			for _, p := range ps {
				_, err := p.patch.String()
				if err != nil {
					return err
				}
				// fmt.Println(s)
				// break
			}
		}

		logTrack.Debug("changes find", "reposiotry", "pathChangesSrc", "len", len(pathChangesSrc[path]))
		logTrack.Debug("changes find", "reposiotry", "pathChangesDst", "len", len(pathChangesDst[path]))

		// os.Exit(0)

		// 	// srcContent, err := getCommitFileContent(t.srcRepository, tracked.srcCommit, tracked.srcObject)
		// 	// if err != nil {
		// 	// 	if errors.Is(err, object.ErrFileNotFound) {
		// 	// 		// TODO: go-git random issue. Random not found some file
		// 	// 		logTrack.Warn("File not found in src repository commit", "error", err.Error(), "hash", tracked.srcCommit.String(), "object", tracked.srcObject)
		// 	// 		continue
		// 	// 	} else {
		// 	// 		logTrack.Error("Read file contents src repository", "error", err.Error(), "hash", tracked.srcCommit.String(), "object", tracked.srcObject)
		// 	// 		return err
		// 	// 	}
		// 	// }

		// 	// dstContent, err := getCommitFileContent(t.dstRepository, tracked.dstCommit, tracked.dstObject)
		// 	// if err != nil {
		// 	// 	if errors.Is(err, object.ErrFileNotFound) {
		// 	// 		// TODO: go-git random issue. Random not found some file
		// 	// 		logTrack.Warn("File not found in dst repository commit", "error", err.Error(), "hash", tracked.dstCommit.String(), "object", tracked.dstObject)
		// 	// 		continue
		// 	// 	} else {
		// 	// 		logTrack.Error("Read file contents dst repository", "error", err.Error(), "hash", tracked.dstCommit.String(), "object", tracked.dstObject)
		// 	// 		return err
		// 	// 	}
		// 	// }

		// 	// if path == "pkg/services/hooks/hooks.go" {
		// 	// 	for hash, status := range detail.commits {
		// 	// 		for _, s := range status {
		// 	// 			fmt.Println(path, hash.String(), s)
		// 	// 		}
		// 	// 	}
		// 	// 	for src, dst := range detail.pairing {
		// 	// 		fmt.Println(src, dst)
		// 	// 	}
		// 	// }
	}

	// return nil

	// for _, tracked := range trackObjects {

	// 	fmt.Println(tracked)
	// 	fmt.Println("-----------")
	// 	continue

	// 	// srcContent, err := getCommitFileContent(t.srcRepository, tracked.srcCommit, tracked.srcObject)
	// 	// if err != nil {
	// 	// 	if errors.Is(err, object.ErrFileNotFound) {
	// 	// 		// TODO: go-git random issue. Random not found some file
	// 	// 		logTrack.Warn("File not found in src repository commit", "error", err.Error(), "hash", tracked.srcCommit.String(), "object", tracked.srcObject)
	// 	// 		continue
	// 	// 	} else {
	// 	// 		logTrack.Error("Read file contents src repository", "error", err.Error(), "hash", tracked.srcCommit.String(), "object", tracked.srcObject)
	// 	// 		return err
	// 	// 	}
	// 	// }

	// 	// dstContent, err := getCommitFileContent(t.dstRepository, tracked.dstCommit, tracked.dstObject)
	// 	// if err != nil {
	// 	// 	if errors.Is(err, object.ErrFileNotFound) {
	// 	// 		// TODO: go-git random issue. Random not found some file
	// 	// 		logTrack.Warn("File not found in dst repository commit", "error", err.Error(), "hash", tracked.dstCommit.String(), "object", tracked.dstObject)
	// 	// 		continue
	// 	// 	} else {
	// 	// 		logTrack.Error("Read file contents dst repository", "error", err.Error(), "hash", tracked.dstCommit.String(), "object", tracked.dstObject)
	// 	// 		return err
	// 	// 	}
	// 	// }

	// 	// contentDiff := diff.Strings(srcContent, dstContent)
	// 	// if len(contentDiff) > 0 {
	// 	// 	fmt.Println(contentDiff)
	// 	// 	break
	// 	// }
	// }

	return nil
}

// func getCommitFileContent(repository *git.Repository, oid *git.Oid, path string) (string, error) {
// 	c, err := repository.LookupCommit(oid)
// 	if err != nil {
// 		return "", err
// 	}

// 	f, err := c.File(path)
// 	if err != nil {
// 		return "", err
// 	}

// 	content, err := f.Contents()
// 	if err != nil {
// 		return "", err
// 	}

// 	return content, nil
// }
