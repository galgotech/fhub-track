package track

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/filesystem"

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
	trackObjectsHash     *plumbing.Reference
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

func (t *Track) trackUpdate() error {
	tracks, err := t.searchTrackObjects()
	if err != nil {
		return err
	}

	fmt.Println(tracks)

	return nil
}

func (t *Track) searchTrackObjects() (map[string][]string, error) {
	trackLog, err := t.trackObjects.Log(&git.LogOptions{})
	if err != nil {
		return nil, err
	}

	parseMessageKey := func(line string) (string, bool) {
		if line == "repo:" || line == "hash:" || line == "files:" || line == "src:" || line == "dst:" {
			return line[:len(line)-1], true
		}
		return "", false
	}

	tracks := map[string][]string{}

	err = trackLog.ForEach(func(commit *object.Commit) error {
		lines := strings.Split(commit.Message, "\n")
		if lines[0] == "fhub-track" {
			var repos, files []string
			var hash, trackSrc, trackDst string

			lastKey := ""
			for _, line := range lines[1:] {
				line = strings.TrimSpace(line)

				if key, ok := parseMessageKey(line); ok {
					lastKey = key
				} else if lastKey == "src" {
					trackSrc = line
				} else if lastKey == "dst" {
					trackDst = line
				} else if lastKey == "repo" {
					repos = append(repos, line)
				} else if lastKey == "hash" {
					hash = line
				} else if lastKey == "files" {
					files = append(files, line)
				}
			}

			fmt.Println(trackSrc, trackDst)

			tracks[hash] = files
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return tracks, nil
}

func (t *Track) trackMultipeObject(trackMultipleObject string) error {
	status, err := t.trackObjectsWorkTree.Status()
	if err != nil {
		return err
	}

	filesModified := []string{}
	for key, status := range status {
		if status.Staging == git.Added {
			return errors.New("Unable to track files because they were added to the stage area")
		} else if status.Staging == git.Modified {
			filesModified = append(filesModified, key)
		}
	}

	filesSrc := []string{}
	filesDst := []string{}
	trackObjects := strings.Split(trackMultipleObject, ",")
	for _, trackObject := range trackObjects {
		partialFilesSrc, partialFilesDst, err := t.searchVendorWorkTreeFiles(t.vendorWorkTree, trackObject)
		if err != nil {
			return err
		}

		filesSrc = append(filesSrc, partialFilesSrc...)
		filesDst = append(filesDst, partialFilesDst...)
	}

	err = t.trackObject(filesSrc, filesDst)
	if err != nil {
		return err
	}

	status, err = t.trackObjectsWorkTree.Status()
	if err != nil {
		return err
	}

	if !status.IsClean() {
		remotes := []string{}
		for key, remote := range t.vendorConfig.Remotes {
			remotes = append(remotes, fmt.Sprintf("%s:%s", key, strings.Join(remote.URLs, ",")))
		}

		msg := fmt.Sprintf(
			"fhub-track\nrepo:\n  %s\nhash:\n  %s\nfiles:\n  %s",
			strings.Join(remotes, "\n  "), t.vendorHash.Hash().String(), strings.Join(filesSrc, "\n  "),
		)
		commitHash, err := t.trackObjectsWorkTree.Commit(msg, &git.CommitOptions{All: false})
		if err != nil {
			return err
		}

		logTrack.Trace("Commit message", "message", msg, "hash", commitHash.String())

		trackLog, err := t.trackObjects.Log(&git.LogOptions{})
		if err != nil {
			return err
		}

		commit, err := trackLog.Next()
		if err != nil {
			return err
		}
		fmt.Println(commit)
	}

	return nil
}

func (t *Track) searchVendorWorkTreeFiles(workTree *git.Worktree, trackObject string) ([]string, []string, error) {
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
		return nil, nil, errors.New("Invalid track path")
	}

	trackSrcInfo, err := workTree.Filesystem.Stat(trackSrc)
	if err != nil {
		return nil, nil, err
	}

	var files []fs.FileInfo
	if trackSrcInfo.IsDir() {
		files, err = workTree.Filesystem.ReadDir(trackSrc)
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

		trackSrcInfo, err := workTree.Filesystem.Stat(filePathSrc)
		if err != nil {
			return nil, nil, err
		}

		// Track recursively
		if trackSrcInfo.IsDir() {
			_, _, err := t.searchVendorWorkTreeFiles(workTree, fmt.Sprintf("%s:%s", filePathSrc, filePathDst))
			if err != nil {
				return nil, nil, err
			}
			continue
		}

		filesSrc = append(filesSrc, filePathSrc)
		filesDst = append(filesDst, filePathDst)
	}

	return filesSrc, filesDst, nil
}

func (t *Track) trackObject(trackSrc, trackDst []string) error {
	for key, filePathSrc := range trackSrc {
		filePathDst := trackDst[key]

		fileRead, err := t.vendorWorkTree.Filesystem.Open(filePathSrc)
		if err != nil {
			return err
		}

		fileWrite, err := t.trackObjectsWorkTree.Filesystem.Create(filePathDst)
		if err != nil {
			return err
		}

		defer fileRead.Close()
		defer fileWrite.Close()

		for {
			bytes := make([]byte, 8)
			readLen, err := fileRead.Read(bytes)
			if err != nil {
				if err == io.EOF {
					break
				}
				logTrack.Error("Fail error", "error", err.Error())
				return err
			}

			if readLen == 0 {
				break
			}

			_, err = fileWrite.Write(bytes[:readLen])
			if err != nil {
				logTrack.Error("Fail error", "error", err.Error())
				return err
			}
		}

		t.trackObjectsWorkTree.Add(filePathDst)
	}

	return nil
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

func initRepository2(repositoryPath, workTree string) (*git.Repository, error) {
	var r *git.Repository
	var err error

	wt := osfs.New(repositoryPath)
	dot := osfs.New(repositoryPath)
	s := filesystem.NewStorage(dot, cache.NewObjectLRUDefault())

	if _, err = dot.Stat(""); err != nil {
		if os.IsNotExist(err) {
			logTrack.Info("Git init", "repositoryPath", repositoryPath)
			r, err = git.Init(s, wt)
		} else {
			logTrack.Error("Git osfs fail stat", "err", err.Error(), "repositoryPath", repositoryPath)
			return nil, err
		}
	} else {
		logTrack.Info("Git open", "repositoryPath", repositoryPath)
		r, err = git.Open(s, wt)
	}

	if err != nil {
		logTrack.Error("Git fail", "err", err.Error(), "repositoryPath", repositoryPath)
		return nil, err
	}

	return r, nil
}

func Run(cmd *cmd.Cmd, setting *setting.Setting) int {
	var err error
	track := &Track{}

	// Vendor
	track.vendor, err = initRepository(filepath.Join(setting.RootPath, cmd.WorkTreeSrc))
	if err != nil {
		logTrack.Error("Fail start repository", "err", err, "repositoryPath", cmd.WorkTreeSrc)
		return 1
	}

	track.vendorWorkTree, err = track.vendor.Worktree()
	if err != nil {
		return 1
	}

	track.vendorHash, err = track.vendor.Head()
	if err != nil {
		return 1
	}

	track.vendorConfig, err = track.vendor.Config()
	if err != nil {
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
		return 1
	}

	track.trackObjectsHash, err = track.trackObjects.Head()
	if err != nil {
		return 1
	}

	track.trackObjectsConfig, err = track.trackObjects.Config()
	if err != nil {
		return 1
	}

	if cmd.Init {
		return 0
	}

	if cmd.Track != "" {
		err := track.trackMultipeObject(cmd.Track)
		if err != nil {
			logTrack.Error("Track fail", "track", cmd.Track, "error", err.Error())
			return 1
		}
	}

	if cmd.Status {
		err := track.status()
		if err != nil {
			logTrack.Error("Status fail", "error", err.Error())
			return 1
		}
	}

	if cmd.TrackUpdate {
		err := track.trackUpdate()
		if err != nil {
			logTrack.Error("Update track fail", "error", err.Error())
			return 1
		}
	}

	return 0
}
