package track

import (
	"bytes"

	"github.com/go-git/go-git/v5/plumbing/format/diff"
	git "github.com/libgit2/git2go/v34"
)

type pathChunk struct {
	hash      *git.Oid
	filePatch diff.FilePatch
}

func (p pathChunk) Message() string {
	return "commit:" + p.hash.String()
}

func (p pathChunk) FilePatches() []diff.FilePatch {
	return []diff.FilePatch{p.filePatch}
}

func (p pathChunk) String() (string, error) {
	buf := bytes.NewBuffer(nil)
	ue := diff.NewUnifiedEncoder(buf, diff.DefaultContextLines)
	err := ue.Encode(p)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

var pathChangesSrc = make(map[string][]pathChunk, 0)
var pathChangesDst = make(map[string][]pathChunk, 0)

// func searchChanges(repository *git.Repository, path string, start, stop *git.Oid) ([]pathChunk, error) {
// 	changes := []pathChunk{}
// 	hashStack := []*git.Oid{start}
// 	for start != stop && len(hashStack) > 0 {
// 		commit, err := repository.LookupCommit(start)
// 		if err != nil {
// 			return nil, err
// 		}

// 		parents := commit.ParentHashes
// 		if len(parents) > 0 {
// 			commitParent, err := repository.LookupCommit(parents[0])
// 			if err != nil {
// 				return nil, err
// 			}

// 			patch, err := commitParent.Patch(commit)
// 			if err != nil {
// 				return nil, err
// 			}

// 			for _, f := range patch.FilePatches() {
// 				pathPatch := ""
// 				moved := false
// 				from, to := f.Files()
// 				if from != nil && to != nil {
// 					pathPatch = to.Path()
// 					if from.Path() != to.Path() {
// 						moved = true
// 					}
// 				} else if to != nil {
// 					pathPatch = to.Path()
// 				} else if from != nil {
// 					pathPatch = from.Path()
// 				}

// 				if pathPatch == path {
// 					if moved {
// 						panic("trac changes from move file not implementend")
// 					}
// 					changes = append(changes, pathChunk{hash: start, filePatch: f})
// 				}
// 			}
// 		}

// 		hashStack = append(hashStack[1:], parents...)
// 		start = hashStack[0]
// 	}

// 	return changes, nil
// }

func (t *Track) trackUpdate() error {
	return nil
	// srcHead, err := t.srcRepository.Head()
	// if err != nil {
	// 	return err
	// }

	// dstHead, err := t.dstRepository.Head()
	// if err != nil {
	// 	return err
	// }

	// srcHeadHash := srcHead.Target()
	// dstHeadHash := dstHead.Target()

	// trackObjects, err := t.searchAllTrackedObjects()
	// if err != nil {
	// 	return err
	// }

	// for path, trackObject := range trackObjects {
	// 	if path != "pkg/api/api.go" {
	// 		continue
	// 	}

	// 	logTrack.Debug("start update", "path", path)
	// 	if trackObject.commitSrc == srcHeadHash {
	// 		logTrack.Debug("path alredy update", "path", path)
	// 	}

	// 	if trackObject.commitSrc != srcHeadHash {
	// 		logTrack.Debug("start repository src", "commitSrc", trackObject.commitSrc.String(), "srcHeadHash", srcHeadHash.String())
	// 		changes, err := searchChanges(t.srcRepository, path, srcHeadHash, trackObject.commitSrc)
	// 		if err != nil {
	// 			return err
	// 		}
	// 		pathChangesSrc[path] = changes
	// 	}

	// 	if dstHeadHash != trackObject.commitDst {
	// 		logTrack.Debug("start repository dst", "commitDst", trackObject.commitDst.String(), "dstHeadHash", dstHeadHash.String())
	// 		changes, err := searchChanges(t.dstRepository, path, dstHeadHash, trackObject.commitDst)
	// 		if err != nil {
	// 			return err
	// 		}
	// 		pathChangesDst[path] = changes
	// 	}

	// 	if len(pathChangesSrc[path]) > 0 {
	// 		fmt.Println("------------- src")

	// 		ps := pathChangesSrc[path]
	// 		for i, j := 0, len(ps)-1; i < j; i, j = i+1, j-1 {
	// 			ps[i], ps[j] = ps[j], ps[i]
	// 		}

	// 		for _, p := range ps {
	// 			s, err := p.String()
	// 			if err != nil {
	// 				return err
	// 			}
	// 			fmt.Println(s)
	// 			break
	// 		}
	// 	}

	// 	if len(pathChangesDst[path]) > 0 {
	// 		fmt.Println("------------- dst")

	// 		ps := pathChangesDst[path]
	// 		for i, j := 0, len(ps)-1; i < j; i, j = i+1, j-1 {
	// 			ps[i], ps[j] = ps[j], ps[i]
	// 		}

	// 		for _, p := range ps {
	// 			s, err := p.String()
	// 			if err != nil {
	// 				return err
	// 			}
	// 			fmt.Println(s)
	// 			break
	// 		}
	// 	}

	// 	logTrack.Debug("changes find", "reposiotry", "pathChangesSrc", "len", len(pathChangesSrc[path]))
	// 	logTrack.Debug("changes find", "reposiotry", "pathChangesDst", "len", len(pathChangesDst[path]))

	// 	os.Exit(0)

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
	// }

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
