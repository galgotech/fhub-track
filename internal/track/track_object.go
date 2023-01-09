package track

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

type excludeObjects struct {
	include map[string]bool
}

// Pattern defines a single gitignore pattern.
func (i *excludeObjects) Match(path []string, isDir bool) gitignore.MatchResult {
	pathJoin := filepath.Join(path...)
	if _, ok := i.include[pathJoin]; ok {
		return gitignore.Include
	}
	return gitignore.Exclude
}

func (t *Track) trackObject(srcObject, dstObject string) error {
	logTrack.Info("start track object", "srcObject", srcObject, "dstObject", dstObject)

	status, err := t.dstWorkTree.Status()
	if err != nil {
		return err
	}

	for _, status := range status {
		if status.Staging != git.Untracked {
			return errors.New("the destination repository has files untracked")
		} else if status.Staging != git.Unmodified {
			return errors.New("the destination repository has files in the staging area")
		}
	}

	allSrcObjects, err := t.searchObjectsInWorkTree(srcObject)
	if err != nil {
		return err
	}

	allDstObjects := t.renameObjectsToDst(allSrcObjects, srcObject, dstObject)

	t.initExcludeFiles(allDstObjects, t.dstWorkTree)

	err = t.copyObject(allSrcObjects, allDstObjects)
	if err != nil {
		return err
	}

	err = t.dstWorkTree.AddWithOptions(&git.AddOptions{All: true})
	if err != nil {
		return err
	}

	status, err = t.dstWorkTree.Status()
	if err != nil {
		return err
	}

	if !status.IsClean() {
		zipObjects, err := zipObjects(allSrcObjects, allDstObjects)
		if err != nil {
			return err
		}

		msg := fmt.Sprintf("files:\n  %s", strings.Join(zipObjects, "\n  "))
		err = t.commit(msg)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *Track) initExcludeFiles(objects []string, workTree *git.Worktree) {
	excludeObjects := &excludeObjects{
		include: map[string]bool{".": true},
	}

	for _, object := range objects {
		objectBase := object
		for objectBase != "." {
			excludeObjects.include[objectBase] = true
			objectBase = filepath.Dir(objectBase)
		}
		excludeObjects.include[object] = true
	}

	workTree.Excludes = append(workTree.Excludes, excludeObjects)
}

func (t *Track) searchObjectsInWorkTree(object string) ([]string, error) {
	allObjects := []string{}

	objectInfo, err := t.srcWorkTree.Filesystem.Stat(object)
	if err != nil {
		return nil, err
	}

	if !objectInfo.IsDir() {
		return allObjects, nil
	}

	subObjectsInfo, err := t.srcWorkTree.Filesystem.ReadDir(object)
	if err != nil {
		return nil, err
	}

	for _, subObjectInfo := range subObjectsInfo {
		object := filepath.Join(object, subObjectInfo.Name())
		// Walk folders recursively
		if subObjectInfo.IsDir() {
			newObjects, err := t.searchObjectsInWorkTree(object)
			if err != nil {
				return nil, err
			}

			allObjects = append(allObjects, newObjects...)
		} else {
			allObjects = append(allObjects, object)
		}
	}

	return allObjects, nil
}

func (t *Track) renameObjectsToDst(objects []string, srcObject, dstObject string) []string {
	dstObjects := make([]string, len(objects))
	srcObject = filepath.Clean(srcObject)
	dstObject = filepath.Clean(dstObject)
	for i, object := range objects {
		dstObjects[i] = filepath.Clean(strings.Replace(object, srcObject, dstObject, 1))
	}
	return dstObjects
}

func zipObjects(allSrcObjects, allDstObjects []string) ([]string, error) {
	if len(allSrcObjects) != len(allDstObjects) {
		return nil, errors.New("allSrcObjects and allDstObjects have different length")
	}

	zipObjects := make([]string, len(allSrcObjects))
	for i := 0; i < len(allSrcObjects); i++ {
		zipObjects[i] = fmt.Sprintf("%s:%s", allSrcObjects[i], allDstObjects[i])
	}

	return zipObjects, nil
}

func (t *Track) copyObject(allSrcObjects, allDstObjects []string) error {
	if len(allSrcObjects) != len(allDstObjects) {
		return errors.New("allSrcObjects and allDstObjects have different length")
	}

	for i := 0; i < len(allSrcObjects); i++ {
		srcObject := allSrcObjects[i]
		dstObject := allDstObjects[i]

		logTrack.Debug("copy object", "src", srcObject, "dst", dstObject)

		fileRead, err := t.srcWorkTree.Filesystem.Open(srcObject)
		if err != nil {
			return err
		}
		defer fileRead.Close()

		fileWrite, err := t.dstWorkTree.Filesystem.Create(dstObject)
		if err != nil {
			return err
		}
		defer fileWrite.Close()

		_, err = io.Copy(fileWrite, fileRead)
		if err != nil {
			return err
		}
	}

	return nil
}
