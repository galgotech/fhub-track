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

type ignoreFiles struct {
	files map[string]bool
}

// Pattern defines a single gitignore pattern.
func (i *ignoreFiles) Match(path []string, isDir bool) gitignore.MatchResult {
	pathJoin := filepath.Join(path...)
	if _, ok := i.files[pathJoin]; ok {
		return gitignore.Include
	}
	return gitignore.Exclude
}

func (t *Track) trackMultipeObject(trackMultipleObject string, ignoreModified bool) error {
	status, err := t.trackObjectsWorkTree.Status()
	if err != nil {
		return err
	}

	includes := &ignoreFiles{
		files: map[string]bool{".": true},
	}
	t.trackObjectsWorkTree.Excludes = append(t.trackObjectsWorkTree.Excludes, includes)

	filesModified := []string{}
	for key, status := range status {
		if status.Staging == git.Added {
			return errors.New("Unable to track files because they were added to the stage area")
		} else if status.Worktree == git.Modified {
			filesModified = append(filesModified, key)
		}
	}

	filesSrc := []string{}
	filesDst := []string{}
	trackObjects := strings.Split(trackMultipleObject, ",")
	for _, trackObject := range trackObjects {
		trackSrc, trackDst, err := splitTrackObject(trackObject)
		if err != nil {
			return err
		}

		trackDstBase := trackDst
		for trackDstBase != "." {
			includes.files[trackDstBase] = true
			trackDstBase = filepath.Dir(trackDstBase)
		}

		partialFilesSrc, partialFilesDst, err := t.searchVendorWorkTreeFiles(trackSrc, trackDst, includes)
		if err != nil {
			return err
		}

		if len(partialFilesSrc) != len(partialFilesDst) {
			return errors.New("Differente number of files src and dst")
		}

		filesSrc = append(filesSrc, partialFilesSrc...)
		filesDst = append(filesDst, partialFilesDst...)
	}

	for key, fileDst := range filesDst {
		for _, fileModified := range filesModified {
			if ignoreModified {
				filesSrc = append(filesSrc[:key], filesSrc[key+1:]...)
				filesDst = append(filesDst[:key], filesDst[key+1:]...)
			} else if fileModified == fileDst {
				return fmt.Errorf("File to track is modified %s", fileDst)
			}
		}
	}

	err = t.copyObject(filesSrc, filesDst)
	if err != nil {
		return err
	}

	err = t.trackObjectsWorkTree.AddWithOptions(&git.AddOptions{All: true})
	if err != nil {
		return err
	}

	status, err = t.trackObjectsWorkTree.Status()
	if err != nil {
		return err
	}

	if !status.IsClean() {
		msg := fmt.Sprintf("files:\n  %s", strings.Join(filesSrc, "\n  "))
		err := t.commit(msg)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *Track) searchVendorWorkTreeFiles(trackSrc, trackDst string, includes *ignoreFiles) ([]string, []string, error) {
	trackSrcInfo, err := t.vendorWorkTree.Filesystem.Stat(trackSrc)
	if err != nil {
		return nil, nil, err
	}

	var files []fs.FileInfo
	if trackSrcInfo.IsDir() {
		files, err = t.vendorWorkTree.Filesystem.ReadDir(trackSrc)
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

		trackSrcInfo, err := t.vendorWorkTree.Filesystem.Stat(filePathSrc)
		if err != nil {
			return nil, nil, err
		}

		// Track recursively
		if trackSrcInfo.IsDir() {
			includes.files[filePathDst] = true
			filePathSrc, filePathDst, err := t.searchVendorWorkTreeFiles(filePathSrc, filePathDst, includes)
			if err != nil {
				return nil, nil, err
			}

			filesSrc = append(filesSrc, filePathSrc...)
			filesDst = append(filesDst, filePathDst...)
			continue
		}

		_, err = t.trackObjectsWorkTree.Filesystem.Stat(filePathDst)
		if err == nil {
			logTrack.Warn("Object already is tracked", "path", filePathDst)
		} else if errors.Is(err, fs.ErrNotExist) {
			filesSrc = append(filesSrc, filePathSrc)
			filesDst = append(filesDst, filePathDst)
			includes.files[filePathDst] = true
		} else {
			return nil, nil, err
		}
	}

	return filesSrc, filesDst, nil
}

func (t *Track) copyObject(trackSrc, trackDst []string) error {
	vendorFileSystem := t.vendorWorkTree.Filesystem
	trackObjectsFileSystem := t.trackObjectsWorkTree.Filesystem

	for key, filePathSrc := range trackSrc {
		filePathDst := trackDst[key]

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
