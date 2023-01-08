package track

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
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

func (t *Track) trackMultipeObject(objects []string) error {
	status, err := t.dstWorkTree.Status()
	if err != nil {
		return err
	}

	for _, status := range status {
		if status.Staging != git.Unmodified {
			return errors.New("the destination repository has files in the staging area")
		}
	}

	allObjects := []string{}
	for _, object := range objects {
		newObjects, err := t.searchObjectsInWorkTree(object)
		if err != nil {
			return err
		}
		allObjects = append(allObjects, newObjects...)
	}

	t.initExcludeFiles(allObjects, t.dstWorkTree)

	err = t.copyObject(objects, objects)
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
		msg := fmt.Sprintf("files:\n  %s", strings.Join(objects, "\n  "))
		err := t.commit(msg)
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
		excludeObjects.include[object] = true
	}

	workTree.Excludes = append(workTree.Excludes, excludeObjects)
}

func (t *Track) searchObjectsInWorkTree(object string) ([]string, error) {
	allObjects := []string{object}

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
		// Walk folders recursively
		if subObjectInfo.IsDir() {
			object := filepath.Join(object, subObjectInfo.Name())
			newObjects, err := t.searchObjectsInWorkTree(object)
			if err != nil {
				return nil, err
			}

			allObjects = append(allObjects, newObjects...)
		}
	}

	return allObjects, nil
}

func (t *Track) searchSrcWorkTreeFiles(trackSrc, trackDst string, includes *excludeObjects) ([]string, []string, error) {
	trackSrcInfo, err := t.srcWorkTree.Filesystem.Stat(trackSrc)
	if err != nil {
		return nil, nil, err
	}

	var files []fs.FileInfo
	if trackSrcInfo.IsDir() {
		files, err = t.srcWorkTree.Filesystem.ReadDir(trackSrc)
		if err != nil {
			return nil, nil, err
		}
	} else {
		trackSrc = filepath.Dir(trackSrc)
		trackDst = filepath.Dir(trackDst)
		files = append(files, trackSrcInfo)
	}

	filesSrc := []string{}
	filesDst := []string{}
	for _, file := range files {
		filePathSrc := filepath.Join(trackSrc, file.Name())
		filePathDst := filepath.Join(trackDst, file.Name())

		trackSrcInfo, err := t.srcWorkTree.Filesystem.Stat(filePathSrc)
		if err != nil {
			return nil, nil, err
		}

		// Walk folders recursively
		if trackSrcInfo.IsDir() {
			includes.include[filePathDst] = true
			filePathSrc, filePathDst, err := t.searchSrcWorkTreeFiles(filePathSrc, filePathDst, includes)
			if err != nil {
				return nil, nil, err
			}

			filesSrc = append(filesSrc, filePathSrc...)
			filesDst = append(filesDst, filePathDst...)
			continue
		}

		_, err = t.dstWorkTree.Filesystem.Stat(filePathDst)
		if err == nil {
			logTrack.Warn("Object already is tracked", "path", filePathDst)
		} else if errors.Is(err, fs.ErrNotExist) {
			filesSrc = append(filesSrc, filePathSrc)
			filesDst = append(filesDst, filePathDst)
			includes.include[filePathDst] = true
		} else {
			return nil, nil, err
		}
	}

	return filesSrc, filesDst, nil
}

func (t *Track) copyObject(srcObject, dstObject []string) error {
	vendorFileSystem := t.srcWorkTree.Filesystem
	trackObjectsFileSystem := t.dstWorkTree.Filesystem

	for key, filePathSrc := range srcObject {
		filePathDst := dstObject[key]

		fileRead, err := vendorFileSystem.Open(filePathSrc)
		if err != nil {
			return err
		}
		defer fileRead.Close()

		fileWrite, err := trackObjectsFileSystem.Create(filePathDst)
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
