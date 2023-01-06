package track

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/galgotech/gotools/diff"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/galgotech/fhub-track/internal/cmd"
	"github.com/galgotech/fhub-track/internal/log"
	"github.com/galgotech/fhub-track/internal/setting"
)

var logTrack = log.New("track")

type logGitProgess struct {
	log log.Logger
}

func (l *logGitProgess) Write(s []byte) (int, error) {
	return len(s), nil
}

type Track struct {
	vendor         *git.Repository
	vendorWorkTree *git.Worktree
	vendorHash     *plumbing.Reference
	vendorConfig   *config.Config

	trackObjects         *git.Repository
	trackObjectsWorkTree *git.Worktree
	trackObjectsConfig   *config.Config
}

func (t *Track) status() error {
	trackObjectWorkTree, err := t.trackObjects.Worktree()
	if err != nil {
		return err
	}

	status, err := trackObjectWorkTree.Status()
	if err != nil {
		return err
	}

	fmt.Println(status)

	return nil
}

type hashObjectsTracked struct {
	vendorHash   plumbing.Hash
	trackHash    plumbing.Hash
	objects      []string
	objectRename string
}

type fileHash struct {
	vendorHash plumbing.Hash
	trackHash  plumbing.Hash
}

type objectName struct {
	src string
	dst string
}

type listObjectTracked = []*hashObjectsTracked

func (t *Track) trackUpdate() error {
	tracked, err := t.searchTrackHash()
	if err != nil {
		return err
	}

	objectsName := map[string]*objectName{}
	objects := map[*objectName][]fileHash{}
	for _, track := range tracked {
		for _, object := range track.objects {
			objectName := getObjectName(objectsName, object)
			if _, ok := objects[objectName]; !ok {
				objects[objectName] = make([]fileHash, 0)
			}
			objects[objectName] = append(objects[objectName], fileHash{
				vendorHash: track.vendorHash,
				trackHash:  track.trackHash,
			})
		}

		if track.objectRename != "" {
			rename := strings.Split(track.objectRename, ":")
			objectName := getObjectName(objectsName, rename[0])
			objectName.dst = rename[1]
		}
	}

	for objectName, hash := range objects {
		vendorContent, err := getCommitFileContent(t.vendor, hash[0].vendorHash, objectName.src)
		if err != nil {
			if errors.Is(err, object.ErrFileNotFound) {
				// TODO: go-git random issue. Random not found some file
				logTrack.Warn("File not found in commit", "repository", "vendor", "error", err.Error(), "vendorHash", hash[0].vendorHash.String(), "objectSrc", objectName.src)
				continue
			} else {
				logTrack.Error("Read file contents", "repository", "vendor", "error", err.Error(), "vendorHash", hash[0].vendorHash.String(), "objectSrc", objectName.src)
				return err
			}
		}

		trackContent, err := getCommitFileContent(t.trackObjects, hash[0].trackHash, objectName.dst)
		if err != nil {
			logTrack.Error("Read file contents", "repository", "track", "error", err.Error(), "trackObjectsHash", hash[0].trackHash.String(), "objectDst", objectName.dst)
			return err
		}

		contentDiff := diff.Strings(vendorContent, trackContent)
		if len(contentDiff) > 0 {
			// fmt.Println(contentDiff)
			break
		}
	}

	return nil
}

func getObjectName(objectsName map[string]*objectName, name string) *objectName {
	if _, ok := objectsName[name]; !ok {
		objectsName[name] = &objectName{
			src: name,
			dst: name,
		}
	}

	return objectsName[name]
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

func (t *Track) searchTrackHash() (listObjectTracked, error) {
	trackLog, err := t.trackObjects.Log(&git.LogOptions{})
	if err != nil {
		return nil, err
	}

	parseMessageKey := func(line string) (string, bool) {
		if line == "repo:" || line == "hash:" || line == "files:" || line == "rename:" {
			return line[:len(line)-1], true
		}
		return "", false
	}

	tracks := listObjectTracked{}

	err = trackLog.ForEach(func(commit *object.Commit) error {
		lines := strings.Split(commit.Message, "\n")
		if lines[0] == "fhub-track" {
			var repos, files []string
			var hash, rename string

			lastKey := ""
			for _, line := range lines[1:] {
				line = strings.TrimSpace(line)

				if key, ok := parseMessageKey(line); ok {
					lastKey = key
				} else if lastKey == "repo" {
					repos = append(repos, line)
				} else if lastKey == "hash" {
					hash = line
				} else if lastKey == "files" {
					files = append(files, line)
				} else if lastKey == "rename" {
					rename = line
				}
			}

			tracks = append(tracks, &hashObjectsTracked{
				vendorHash:   plumbing.NewHash(hash),
				trackHash:    commit.Hash,
				objects:      files,
				objectRename: rename,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return tracks, nil
}

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

func (t *Track) trackRenameObject(trackRenameObject string) error {
	trackSrc, trackDst, err := splitTrackObject(trackRenameObject)
	if err != nil {
		return err
	}

	status, err := t.trackObjectsWorkTree.Status()
	if err != nil {
		return err
	}

	for key, status := range status {
		if status.Staging == git.Added {
			return errors.New("unable to track files because they were added to the stage area")
		} else if status.Worktree == git.Modified && key == trackSrc {
			return errors.New("unable to rename track files because they were modified in worktree")
		}
	}

	err = t.trackObjectsWorkTree.Filesystem.Rename(trackSrc, trackDst)
	if err != nil {
		return err
	}

	_, err = t.trackObjectsWorkTree.Remove(trackSrc)
	if err != nil {
		return err
	}

	_, err = t.trackObjectsWorkTree.Add(trackDst)
	if err != nil {
		return err
	}

	msg := fmt.Sprintf("rename:\n  %s", trackRenameObject)
	err = t.commit(msg)
	if err != nil {
		return err
	}

	return nil
}

func (t *Track) commit(msg string) error {
	remotes := []string{}
	for key, remote := range t.vendorConfig.Remotes {
		remotes = append(remotes, fmt.Sprintf("%s:%s", key, strings.Join(remote.URLs, ",")))
	}

	msg = fmt.Sprintf(
		"fhub-track\nrepo:\n  %s\nhash:\n  %s\n%s",
		strings.Join(remotes, "\n  "), t.vendorHash.Hash().String(), msg,
	)

	commitHash, err := t.trackObjectsWorkTree.Commit(msg, &git.CommitOptions{All: false})
	if err != nil {
		return err
	}

	logTrack.Debug("Commit message", "message", msg, "hash", commitHash.String())

	trackLog, err := t.trackObjects.Log(&git.LogOptions{})
	if err != nil {
		return err
	}

	commit, err := trackLog.Next()
	if err != nil {
		return err
	}
	fmt.Println(commit)

	return nil
}

func splitTrackObject(trackObject string) (string, string, error) {
	trackSrc := ""
	trackDst := ""
	paths := strings.Split(trackObject, ":")
	if len(paths) == 1 {
		trackSrc = paths[0]
		trackDst = paths[0]
	} else if len(paths) == 2 {
		trackSrc = paths[0]
		trackDst = paths[1]
	} else {
		return "", "", errors.New("invalid track path")
	}

	return trackSrc, trackDst, nil
}

func cloneRepository(repositoryURL, repositoryPath string) (*git.Repository, error) {
	r, err := git.PlainOpen(repositoryPath)
	if err == git.ErrRepositoryNotExists {
		logTrack.Info("Git plain clone", "repositoryURL", repositoryURL, "repositoryPath", repositoryPath)
		r, err = git.PlainClone(repositoryPath, false, &git.CloneOptions{
			URL:      repositoryURL,
			Progress: os.Stdout,
		})

		if err != nil {
			logTrack.Error("Git clone fail", "err", err.Error(), "repositoryPath", repositoryPath, "repositoryURL", repositoryURL)
			return nil, err
		}
	} else {
		logTrack.Info("Git plain open", "repositoryPath", repositoryPath)
	}

	return r, nil
}

func initRepository(workTree string) (*git.Repository, error) {
	r, err := git.PlainOpen(workTree)
	if err != nil {
		if err == git.ErrRepositoryNotExists {
			r, err = git.PlainInit(workTree, false)
		}
	}

	return r, err
}

func Run(cmd *cmd.Cmd, setting *setting.Setting) int {
	var err error
	track := &Track{}

	// Vendor
	track.vendor, err = initRepository(filepath.Join(setting.RootPath, cmd.WorkTreeSrc))
	if err != nil {
		logTrack.Error("Fail start repository", "err", err.Error(), "repositoryPath", cmd.WorkTreeSrc)
		return 1
	}

	track.vendorWorkTree, err = track.vendor.Worktree()
	if err != nil {
		logTrack.Error("Fail get vendor repository worktree", "err", err.Error())
		return 1
	}

	track.vendorHash, err = track.vendor.Head()
	if err != nil {
		logTrack.Error("Fail get vendor repository hash", "err", err.Error())
		return 1
	}

	track.vendorConfig, err = track.vendor.Config()
	if err != nil {
		logTrack.Error("Fail get vendor repository config", "err", err.Error())
		return 1
	}

	// Track objects
	track.trackObjects, err = initRepository(filepath.Join(setting.RootPath, cmd.WorkTreeDst))
	if err != nil {
		logTrack.Error("Fail start repository", "err", err, "WorkTree", cmd.WorkTreeDst)
		return 1
	}

	track.trackObjectsWorkTree, err = track.trackObjects.Worktree()
	if err != nil {
		logTrack.Error("Fail get track repository worktree", "err", err.Error())
		return 1
	}

	track.trackObjectsConfig, err = track.trackObjects.Config()
	if err != nil {
		logTrack.Error("Fail get track repository track", "err", err.Error())
		return 1
	}

	if cmd.Init {
		return 0
	}

	if cmd.Track != "" {
		err := track.trackMultipeObject(cmd.Track, cmd.TrackIgnoreModified)
		if err != nil {
			logTrack.Error("Track fail", "track", cmd.Track, "error", err.Error())
			return 1
		}
	} else if cmd.TrackRename != "" {
		err := track.trackRenameObject(cmd.TrackRename)
		if err != nil {
			logTrack.Error("Track rename fail", "track", cmd.TrackRename, "error", err.Error())
			return 1
		}
	} else if cmd.Status {
		err := track.status()
		if err != nil {
			logTrack.Error("Status fail", "error", err.Error())
			return 1
		}
	} else if cmd.TrackUpdate {
		err := track.trackUpdate()
		if err != nil {
			logTrack.Error("Update track fail", "error", err.Error())
			return 1
		}
	}

	return 0
}
