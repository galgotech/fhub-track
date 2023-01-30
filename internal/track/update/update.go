package update

import (
	"os"

	"github.com/galgotech/fhub-track/internal/log"
	git "github.com/libgit2/git2go/v34"
)

func New(src *git.Repository, dst *git.Repository) *Update {
	return &Update{
		src: src,
		dst: dst,
	}
}

type Update struct {
	src *git.Repository
	dst *git.Repository
}

var logTrack = log.New("track-update")

func (t *Update) Run() error {
	logTrack.Debug("start")

	srcHead, err := t.src.Head()
	if err != nil {
		return err
	}

	dstHead, err := t.dst.Head()
	if err != nil {
		return err
	}

	srcHeadOid := srcHead.Target()
	dstHeadOid := dstHead.Target()

	objects, err := t.SearchObjects()
	if err != nil {
		return err
	}

	logTrack.Debug("objects", "count", len(objects))

	for path, object := range objects {
		if path != "pkg/api/api.go" {
			continue
		}

		/////
		objectsPatchSrc, err := searchPatchByPath(t.src, object.pathSrc, srcHeadOid, object.commitSrc)
		if err != nil {
			return err
		}
		if len(objectsPatchSrc) == 0 {
			logTrack.Debug("repository dst alredy update")
			continue
		}
		logTrack.Debug("searchPatchByPath", "repository", "src", "start", srcHeadOid, "stop", object.commitSrc, "len", len(objectsPatchSrc))

		/////
		objectsPatchDst, err := searchPatchByPath(t.dst, path, dstHeadOid, object.commitDst)
		if err != nil {
			return err
		}
		logTrack.Debug("searchPatchByPath", "repository", "dst", "start", dstHeadOid, "stop", object.commitDst, "len", len(objectsPatchDst))
		err = t.updateObject(path, objectsPatchDst, objectsPatchDst)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *Update) updateObject(path string, patchsSrc, patchsDst []patchByPath) (err error) {
	// ancestor commit
	ancestorTree, ancestorContent, err := treeAndPathContent(t.dst, patchsDst[0].commit.TreeId(), path)
	if err != nil {
		return err
	}

	for _, patchSrc := range patchsSrc[:1] {
		ancestorTree, ancestorContent, err = t.applyPath(path, ancestorContent, patchSrc, ancestorTree)
		if err != nil {
			return err
		}

		for _, patchDst := range patchsDst[1:] {
			ancestorTree, ancestorContent, err = t.applyPath(path, ancestorContent, patchDst, ancestorTree)
			if err != nil {
				return err
			}
		}

		// ie, _ := indexMerge.EntryByPath(path, 0)
		// c, _ := blobContent(t.dst, ancestorTree, path)
		// fmt.Println(string(c))
		// os.Exit(0)
	}

	os.Exit(0)

	return nil
}

func (t *Update) applyPath(path string, content []byte, patch patchByPath, tree *git.Tree) (*git.Tree, []byte, error) {
	opts := &git.ApplyOptions{
		ApplyHunkCallback: func(dh *git.DiffHunk) (apply bool, err error) {
			return true, nil
		},
		ApplyDeltaCallback: func(dd *git.DiffDelta) (apply bool, err error) {
			return true, nil
		},
	}

	diff, err := prepareDiff(t.dst, content, patch)
	if err != nil {
		return nil, nil, err
	}

	index, err := t.dst.ApplyToTree(diff, tree, opts)
	if err != nil {
		return nil, nil, err
	}

	treeOid, err := index.WriteTreeTo(t.dst)
	if err != nil {
		return nil, nil, err
	}

	return treeAndPathContent(t.dst, treeOid, path)
}

func treeAndPathContent(repo *git.Repository, treeOid *git.Oid, path string) (*git.Tree, []byte, error) {
	tree, err := repo.LookupTree(treeOid)
	if err != nil {
		return nil, nil, err
	}

	treeEntry, err := tree.EntryByPath(path)
	if err != nil {
		return nil, nil, err
	}

	blob, err := repo.LookupBlob(treeEntry.Id)
	if err != nil {
		return nil, nil, err
	}

	return tree, blob.Contents(), nil

}

func prepareDiff(repository *git.Repository, ancestorContent []byte, patch patchByPath) (*git.Diff, error) {
	theirsPatch, err := recountPatch(ancestorContent, patch.patch)
	if err != nil {
		return nil, err
	}
	theirsDiff, err := git.DiffFromBuffer([]byte(theirsPatch), repository)
	if err != nil {
		return nil, err
	}

	return theirsDiff, nil
}
