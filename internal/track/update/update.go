package update

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
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

	headCommitOidSrc := headSrc.Target()
	headCommitOidDst := headDst.Target()

	logTrack.Info("map objects")
	mapObjects, mapCommitsSrc, mapCommitsDst, err := t.MapObjects()
	if err != nil {
		return err
	}

	logTrack.Debug("objects", "count", len(mapObjects))

	logTrack.Info("load blob", "repo", "src")
	err = t.blob(t.src, mapCommitsSrc, headCommitOidSrc)
	if err != nil {
		return err
	}

	logTrack.Info("load blob", "repo", "dst")
	err = t.blob(t.dst, mapCommitsDst, headCommitOidDst)
	if err != nil {
		return err
	}

	for _, objectDst := range mapObjects {
		logTrack.Info("update", "path", objectDst.path)
		objectSrc := objectDst.link
		err := t.updateObject(objectSrc, objectDst)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *Update) blob(repo *git.Repository, mapObjects mapCommitPath, headCommitOid *git.Oid) error {
	headCommit, err := repo.LookupCommit(headCommitOid)
	if err != nil {
		logTrack.Error("head lookup commit", "repo", "src")
		return err
	}
	headTree, err := headCommit.Tree()
	if err != nil {
		logTrack.Error("head tree", "repo", "src")
		return err
	}

	for commitOid, mapPaths := range mapObjects {
		logTrack.Info("blob current commit", "repoPath", repo.Path(), "commit", commitOid, "head", headCommitOid.String())

		oid, err := git.NewOid(commitOid)
		if err != nil {
			return err
		}

		commit, err := repo.LookupCommit(oid)
		if err != nil {
			return err
		}
		oldTree, err := commit.Tree()
		if err != nil {
			return err
		}

		diff, err := repo.DiffTreeToTree(oldTree, headTree, &git.DiffOptions{
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
		err = diff.FindSimilar(&git.DiffFindOptions{
			Flags: git.DiffFindRenames | git.DiffFindCopies | git.DiffFindForUntracked,
		})
		if err != nil {
			return err
		}
		diff.ForEach(func(delta git.DiffDelta, progress float64) (git.DiffForEachHunkCallback, error) {
			switch true {
			case delta.Status == git.DeltaModified:
				if object, ok := mapPaths[delta.OldFile.Path]; ok {
					object.mode = delta.OldFile.Mode
					object.blob = delta.OldFile.Oid

					object.head.commit = headCommitOid.String()
					object.head.path = delta.NewFile.Path
					object.head.mode = delta.NewFile.Mode
					object.head.blob = delta.NewFile.Oid
				}

			case delta.Status == git.DeltaAdded:
				// Ignore added files
				break

			case delta.Status == git.DeltaDeleted:
				if object, ok := mapPaths[delta.OldFile.Path]; ok {
					object.mode = delta.OldFile.Mode
					object.blob = delta.OldFile.Oid

					object.head = nil
				}

			case delta.Status == git.DeltaRenamed || delta.Status == git.DeltaCopied:
				if object, ok := mapPaths[delta.OldFile.Path]; ok {
					object.mode = delta.OldFile.Mode
					object.blob = delta.OldFile.Oid

					object.head.commit = headCommitOid.String()
					object.head.path = delta.NewFile.Path
					object.head.mode = delta.NewFile.Mode
					object.head.blob = delta.NewFile.Oid
				}

			// case git.DeltaUnmodified
			// case git.DeltaIgnored:
			// case git.DeltaUntracked:
			// case git.DeltaTypeChange:
			// case git.DeltaUnreadable:
			// case git.DeltaConflicted:
			default:
				return nil, fmt.Errorf("not implemented diff delta status '%s' '%s' '%s'", delta.Status, delta.NewFile.Path, delta.OldFile.Path)
			}

			return nil, nil
		}, git.DiffDetailFiles)
	}
	return nil
}

func (t *Update) updateObject(objectSrc, objectDst *object) (err error) {
	path := objectDst.path
	if objectDst.head == nil {
		logTrack.Info("deleted", "path", path)
		return nil
	}

	if objectSrc.head == nil {
		pathRemove := filepath.Join(t.setting.DstRepo, path)
		logTrack.Info("deleting object", "path", pathRemove)
		err := os.Remove(pathRemove)
		if err != nil {
			return err
		}
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

	blobAncestor, err := t.src.LookupBlob(objectSrc.blob)
	if err != nil {
		return err
	}
	oursBlob, err := t.src.LookupBlob(objectSrc.head.blob)
	if err != nil {
		return err
	}
	theirsBlob, err := t.dst.LookupBlob(objectDst.head.blob)
	if err != nil {
		return err
	}

	ancestorFile := git.MergeFileInput{
		Path:     path,
		Mode:     0,
		Contents: blobAncestor.Contents(),
	}
	oursFile := git.MergeFileInput{
		Path:     path,
		Mode:     0,
		Contents: oursBlob.Contents(),
	}
	theirsFile := git.MergeFileInput{
		Path:     path,
		Mode:     0,
		Contents: theirsBlob.Contents(),
	}

	mergeResult, err := git.MergeFile(ancestorFile, oursFile, theirsFile, &git.MergeFileOptions{
		AncestorLabel: fmt.Sprintf("ancestor %s", objectDst.commit),
		OurLabel:      fmt.Sprintf("src %s", objectSrc.commit),
		TheirLabel:    fmt.Sprintf("dst %s", objectDst.commit),
		Favor:         git.MergeFileFavorNormal,
		Flags:         git.MergeFileDiffPatience,
		//  MarkerSize    uint16
	})
	if err != nil {
		return err
	}

	// var pathSave string
	// if objectSrc.path != objectSrc.head.path {
	// 	// contents = fmt.Sprintf("%b->%b%b", objectDst.path, objectDst.link, contents)
	// } else {
	// 	pathSave = filepath.Join(t.setting.DstRepo, path)
	// }

	path = filepath.Join(t.setting.DstRepo, path)
	err = ioutil.WriteFile(path, mergeResult.Contents, fs.FileMode(objectDst.mode))
	if err != nil {
		return err
	}
	return nil
}
