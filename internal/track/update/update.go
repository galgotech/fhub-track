package update

import (
	"errors"
	"os"

	"github.com/galgotech/fhub-track/internal/log"
	git "github.com/libgit2/git2go/v34"
)

var ErrPatchConflict = errors.New("patch conflict")

func New(src *git.Repository, dst *git.Repository) *Update {
	return &Update{
		src: src,
		dst: dst,
	}
}

var logTrack = log.New("track-update")

type Update struct {
	src *git.Repository
	dst *git.Repository
}

func (t *Update) Run() error {
	logTrack.Debug("start")

	headSrc, err := t.src.Head()
	if err != nil {
		return err
	}

	headDst, err := t.dst.Head()
	if err != nil {
		return err
	}

	headOidSrc := headSrc.Target()
	headOidDst := headDst.Target()

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
		objectsPatchSrc, err := searchPatchByPath(t.src, object.pathSrc, headOidSrc, object.commitSrc)
		if err != nil {
			return err
		}
		if len(objectsPatchSrc) == 0 {
			logTrack.Debug("repository dst alredy update")
			continue
		}
		logTrack.Debug("searchPatchByPath", "path", path, "repository", "src", "start", headOidSrc.String(), "stop", object.commitSrc.String(), "len", len(objectsPatchSrc))

		/////
		objectsPatchDst, err := searchPatchByPath(t.dst, path, headOidDst, object.commitDst)
		if err != nil {
			return err
		}
		logTrack.Debug("searchPatchByPath", "path", path, "repository", "dst", "start", headOidDst.String(), "stop", object.commitDst.String(), "len", len(objectsPatchDst))

		err = t.updateObject(path, objectsPatchSrc, objectsPatchDst)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *Update) updateObject(path string, patchsSrc, patchsDst []patchByPath) (err error) {
	logTrack.Debug("first ancestor", "commit", patchsDst[0].commit.Id().String())

	ancestorTree, ancestor, err := t.treeAndPathContent(patchsDst[0].commit.TreeId(), path)
	if err != nil {
		return err
	}

	for i, patchSrc := range patchsSrc {
		logTrack.Debug("apply path src", "i", i, "commit", patchSrc.commit.Id().String())
		theirTree, theirAncestor, err := t.applyPath(path, ancestorTree, ancestor, patchSrc)
		if err != nil {
			return err
		}
		ancestorTree, ancestor = theirTree, theirAncestor

		for j, patchDst := range patchsDst[1:] {
			logTrack.Debug("apply path dst", "j", j, "commit", patchDst.commit.Id().String())
			oursTree, oursAncestor, err := t.applyPath(path, ancestorTree, ancestor, patchDst)
			if err != nil {
				if errors.Is(ErrPatchConflict, err) {
					// _, contentDst, _ := t.treeAndPathContent(patchDst.commit.TreeId(), path)
					// opts := &git.DiffOptions{
					// 	Flags:          git.DiffPatience,
					// 	ContextLines:   5,
					// 	InterhunkLines: 5,
					// 	OldPrefix:      "a",
					// 	NewPrefix:      "b",
					// }
					// callcack := func(delta git.DiffDelta, progress float64) (git.DiffForEachHunkCallback, error) {
					// 	return func(hunk git.DiffHunk) (git.DiffForEachLineCallback, error) {
					// 		return func(line git.DiffLine) error {
					// 			fmt.Print(line.Origin, line.Content)
					// 			return nil
					// 		}, nil
					// 	}, nil
					// }
					// err := git.DiffBuffers([]byte(ancestor), path, []byte(contentDst), path, opts, callcack, git.DiffDetailLines)
					// // fmt.Println(oldContent)

					// // fmt.Println(patchSrc.patch.String())
					// // fmt.Println("---")
					// // fmt.Println(patchDst.patch.String())
					return err
				}
				return err
			}
			ancestorTree, ancestor = oursTree, oursAncestor
		}

		// ie, _ := indexMerge.EntryByPath(path, 0)
		// c, _ := blobContent(t.dst, ancestorTree, path)
		// fmt.Println(string(c))
		os.Exit(0)
	}

	os.Exit(0)

	return nil
}

func (t *Update) applyPath(path string, tree *git.Tree, content string, patch patchByPath) (*git.Tree, string, error) {
	opts := &git.ApplyOptions{
		ApplyHunkCallback: func(dh *git.DiffHunk) (apply bool, err error) {
			return true, nil
		},
		ApplyDeltaCallback: func(dd *git.DiffDelta) (apply bool, err error) {
			return true, nil
		},
	}

	diff, err := t.prepareDiff(content, patch)
	if err != nil {
		return nil, "", err
	}

	index, err := t.dst.ApplyToTree(diff, tree, opts)
	if err != nil {
		return nil, "", err
	}

	treeOid, err := index.WriteTreeTo(t.dst)
	if err != nil {
		return nil, "", err
	}

	return t.treeAndPathContent(treeOid, path)
}

func (t *Update) treeAndPathContent(treeOid *git.Oid, path string) (*git.Tree, string, error) {
	tree, err := t.dst.LookupTree(treeOid)
	if err != nil {
		return nil, "", err
	}

	treeEntry, err := tree.EntryByPath(path)
	if err != nil {
		return nil, "", err
	}

	blob, err := t.dst.LookupBlob(treeEntry.Id)
	if err != nil {
		return nil, "", err
	}

	return tree, string(blob.Contents()), nil
}

func (t *Update) prepareDiff(ancestor string, patch patchByPath) (*git.Diff, error) {
	theirsPatch, err := recountPatch(ancestor, patch.patch)
	if err != nil {
		return nil, err
	}
	theirsDiff, err := git.DiffFromBuffer([]byte(theirsPatch), t.dst)
	if err != nil {
		return nil, err
	}

	return theirsDiff, nil
}
