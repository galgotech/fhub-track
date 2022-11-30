package track

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
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
	vendor       *git.Repository
	trackObjects *git.Repository
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
		if line == "repo:" || line == "hash:" || line == "files:" {
			return line[:len(line)-1], true
		}
		return "", false
	}

	tracks := map[string][]string{}

	err = trackLog.ForEach(func(commit *object.Commit) error {
		lines := strings.Split(commit.Message, "\n")
		if lines[0] == "fhub-track" {
			var repos, files []string
			var hash string

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
				}
			}

			tracks[hash] = files
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return tracks, nil
}

func (t *Track) trackObject(trackObject string) error {
	trackSrc := ""
	trackDst := ""
	paths := strings.Split(trackObject, ":")
	if len(paths) == 1 {
		trackSrc = trackObject
		trackDst = trackObject
	} else if len(paths) == 2 {
		trackSrc = paths[0]
		trackDst = paths[1]
	} else {
		return errors.New("Invalid track path")
	}

	vendorWorkTree, err := t.vendor.Worktree()
	if err != nil {
		return err
	}

	vendorHash, err := t.vendor.Head()
	if err != nil {
		return err
	}

	vendorConfig, err := t.vendor.Config()
	if err != nil {
		return err
	}

	trackObjectWorkTree, err := t.trackObjects.Worktree()
	if err != nil {
		return err
	}

	status, err := trackObjectWorkTree.Status()
	if err != nil {
		return err
	}

	if !status.IsClean() {
		fmt.Println(status.String())
		return errors.New("Need status clean to track objects")
	}

	files, err := vendorWorkTree.Filesystem.ReadDir(trackSrc)
	if err != nil {
		return err
	}

	filesPath := []string{}
	for _, file := range files {
		filePathSrc := filepath.Join(trackSrc, file.Name())
		filePathDst := filepath.Join(trackDst, file.Name())
		filesPath = append(filesPath, filePathSrc)

		fileRead, err := vendorWorkTree.Filesystem.Open(filePathSrc)
		if err != nil {
			return err
		}

		fileWrite, err := trackObjectWorkTree.Filesystem.Create(filePathDst)
		if err != nil {
			return err
		}

		defer fileRead.Close()
		defer fileWrite.Close()

		bytes := make([]byte, 8)
		for {
			_, err := fileRead.Read(bytes)
			if err != nil {
				if err == io.EOF {
					break
				}
				logTrack.Error("Fail error", "error", err.Error())
				return err
			}

			_, err = fileWrite.Write(bytes)
			if err != nil {
				logTrack.Error("Fail error", "error", err)
				return err
			}
		}

		trackObjectWorkTree.Add(filePathDst)
	}

	status, err = trackObjectWorkTree.Status()
	if err != nil {
		return err
	}

	if !status.IsClean() {
		remotes := []string{}
		for key, remote := range vendorConfig.Remotes {
			remotes = append(remotes, fmt.Sprintf("%s:%s", key, strings.Join(remote.URLs, ",")))
		}

		msg := fmt.Sprintf("fhub-track\nrepo:\n  %s\nhash:\n  %s\nfiles:\n  %s", strings.Join(remotes, "\n  "), vendorHash.Hash().String(), strings.Join(filesPath, "\n  "))
		commitHash, err := trackObjectWorkTree.Commit(msg, &git.CommitOptions{All: true})
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

	track.vendor, err = initRepository(filepath.Join(setting.RootPath, cmd.WorkTreeSrc))
	if err != nil {
		logTrack.Error("Fail start repository", "err", err, "repositoryPath", cmd.WorkTreeSrc)
		return 1
	}

	track.trackObjects, err = initRepository(filepath.Join(setting.RootPath, cmd.WorkTreeDst))
	if err != nil {
		logTrack.Error("Fail start repository", "err", err, "WorkTree", cmd.WorkTreeDst)
		return 1
	}

	if cmd.Init {
		return 0
	}

	if cmd.Track != "" {
		err := track.trackObject(cmd.Track)
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
