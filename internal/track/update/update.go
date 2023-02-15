package update

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"path/filepath"

	"github.com/galgotech/fhub-track/internal/log"
	"github.com/galgotech/fhub-track/internal/setting"
	git "github.com/libgit2/git2go/v34"
)

const ContextLines = 5

// errorString is a trivial implementation of error.
type errorPatchConflict struct {
	patch string
}

func (e *errorPatchConflict) Error() string {
	return "patch conflict"
}

func New(setting *setting.Setting, src *git.Repository, dst *git.Repository) *Update {
	return &Update{
		setting: setting,
		src:     src,
		dst:     dst,
	}
}

var logTrack = log.New("track-update")

type Update struct {
	setting *setting.Setting
	src     *git.Repository
	dst     *git.Repository
}

func (t *Update) Run() error {
	logTrack.Debug("start update")

	headSrc, err := t.src.Head()
	if err != nil {
		logTrack.Error("head reference", "repo", "src")
		return err
	}
	headDst, err := t.dst.Head()
	if err != nil {
		logTrack.Error("head reference", "repo", "dst")
		return err
	}

	headCommitSrc, err := t.src.LookupCommit(headSrc.Target())
	if err != nil {
		logTrack.Error("head lookup commit", "repo", "src")
		return err
	}
	headCommitDst, err := t.dst.LookupCommit(headDst.Target())
	if err != nil {
		logTrack.Error("head lookup commit", "repo", "src")
		return err
	}

	headTreeSrc, err := headCommitSrc.Tree()
	if err != nil {
		logTrack.Error("head tree", "repo", "src")
		return err
	}
	headTreeDst, err := headCommitDst.Tree()
	if err != nil {
		logTrack.Error("head tree", "repo", "dst")
		return err
	}

	logTrack.Info("map objects")
	mapObjects, mapCommitsSrc, mapCommitsDst, err := t.MapObjects()
	if err != nil {
		return err
	}

	logTrack.Debug("objects", "count", len(mapObjects))

	logTrack.Info("load blob", "repo", "src")
	err = t.blob(t.src, mapCommitsSrc, headTreeSrc)
	if err != nil {
		return err
	}

	logTrack.Info("load blob", "repo", "dst")
	err = t.blob(t.dst, mapCommitsDst, headTreeDst)
	if err != nil {
		return err
	}

	for path, objectDst := range mapObjects {
		logTrack.Info("update", "path", path)
		objectSrc := objectDst.link
		err := t.updateObject(path, objectSrc, objectDst)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *Update) blob(repo *git.Repository, mapObjects mapCommitPath, newTree *git.Tree) error {
	for commitOid, mapPaths := range mapObjects {
		commit, err := repo.LookupCommit(commitOid)
		if err != nil {
			return err
		}

		oldTree, err := commit.Tree()
		if err != nil {
			return err
		}

		diff, err := repo.DiffTreeToTree(oldTree, newTree, &git.DiffOptions{
			Flags: git.DiffMinimal | git.DiffIncludeTypeChange | git.DiffIncludeTypeChangeTrees,
			// IgnoreSubmodules SubmoduleIgnore
			// Pathspec         []string
			// NotifyCallback   DiffNotifyCallback

			ContextLines: 5,
			// InterhunkLines uint32
			// IdAbbrev       uint16

			// MaxSize int
			OldPrefix: "a",
			NewPrefix: "b",
		})
		if err != nil {
			return err
		}

		// Rename detection (slow operation with many commits diff)
		// err = diff.FindSimilar(&git.DiffFindOptions{
		// 	Flags: git.DiffFindRenames | git.DiffFindCopies | git.DiffFindForUntracked,
		// })
		// if err != nil {
		// 	return err
		// }

		diff.ForEach(func(delta git.DiffDelta, progress float64) (git.DiffForEachHunkCallback, error) {
			var deltaOid *git.Oid
			var deltaPath string
			var deltaMode uint16
			var ancestorDeltaOid *git.Oid
			deleted := false

			switch true {
			case delta.Status == git.DeltaModified:
				deltaOid = delta.NewFile.Oid
				ancestorDeltaOid = delta.OldFile.Oid
				deltaPath = delta.NewFile.Path
				deltaMode = delta.NewFile.Mode
			case delta.Status == git.DeltaAdded:
				deltaOid = delta.NewFile.Oid
				ancestorDeltaOid = delta.NewFile.Oid
				deltaPath = delta.NewFile.Path
				deltaMode = delta.NewFile.Mode
			case delta.Status == git.DeltaDeleted:
				deltaOid = delta.OldFile.Oid
				ancestorDeltaOid = deltaOid
				deltaPath = delta.OldFile.Path
				deltaMode = delta.OldFile.Mode
				deleted = true

			case delta.Status == git.DeltaRenamed || delta.Status == git.DeltaCopied:
				// TODO: review path when renamed or copied
				deltaOid = delta.NewFile.Oid
				ancestorDeltaOid = delta.OldFile.Oid
				deltaPath = delta.NewFile.Path
				deltaMode = delta.NewFile.Mode
			// case git.DeltaUnmodified
			// case git.DeltaIgnored:
			// case git.DeltaUntracked:
			// case git.DeltaTypeChange:
			// case git.DeltaUnreadable:
			// case git.DeltaConflicted:
			default:
				return nil, fmt.Errorf("not implemented diff delta status '%s' '%s' '%s'", delta.Status, delta.NewFile.Path, delta.OldFile.Path)
			}

			if object, ok := mapPaths[deltaPath]; ok {
				blob, err := repo.LookupBlob(deltaOid)
				if err != nil {
					return nil, err
				}
				blobAncestor, err := repo.LookupBlob(ancestorDeltaOid)
				if err != nil {
					return nil, err
				}

				object.blob = blob
				object.mode = deltaMode
				object.deleted = deleted
				object.blobAncestor = blobAncestor
			}

			return nil, nil
		}, git.DiffDetailFiles)
	}
	return nil
}

func (t *Update) updateObject(path string, objectSrc, objectDst *object) (err error) {
	if objectDst.deleted {
		logTrack.Info("deleted", "path", path)
		return nil
	}
	if objectSrc.blob == nil {
		logTrack.Info("unmodified", "path", path, "repo", "src")
		return nil
	}
	if objectDst.blob == nil {
		logTrack.Info("unmodified", "path", path, "repo", "dst")
		return nil
	}

	ancestorFile := git.MergeFileInput{
		Path:     path,
		Mode:     0,
		Contents: []byte(objectSrc.blobAncestor.Contents()),
	}
	oursFile := git.MergeFileInput{
		Path:     path,
		Mode:     0,
		Contents: []byte(objectSrc.blob.Contents()),
	}
	theirsFile := git.MergeFileInput{
		Path:     path,
		Mode:     0,
		Contents: []byte(objectDst.blob.Contents()),
	}

	mergeResult, err := git.MergeFile(ancestorFile, oursFile, theirsFile, &git.MergeFileOptions{
		AncestorLabel: fmt.Sprintf("ancestor %s", objectDst.commit.String()),
		OurLabel:      fmt.Sprintf("src %s", objectSrc.commit.String()),
		TheirLabel:    fmt.Sprintf("dst %s", objectDst.commit.String()),
		Favor:         git.MergeFileFavorNormal,
		Flags:         git.MergeFileDiffPatience,
		//  MarkerSize    uint16
	})
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(t.setting.DstRepo, path), mergeResult.Contents, fs.FileMode(objectDst.mode))
	if err != nil {
		return err
	}
	return nil
}
