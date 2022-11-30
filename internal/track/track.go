package track

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/cache"
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
	pathObjects  string
}

func (t *Track) trackObject(trackObject string) error {
	vendorWorkTree, err := t.vendor.Worktree()
	if err != nil {
		return err
	}

	vendorHash, err := t.vendor.Head()
	if err != nil {
		return err
	}

	trackObjectWorkTree, err := t.trackObjects.Worktree()
	if err != nil {
		return err
	}

	files, _ := vendorWorkTree.Filesystem.ReadDir(trackObject)
	for _, file := range files {
		filePath := filepath.Join(trackObject, file.Name())
		fileRead, _ := vendorWorkTree.Filesystem.Open(filePath)
		fileWrite, _ := trackObjectWorkTree.Filesystem.Create(filePath)
		defer fileRead.Close()
		defer fileWrite.Close()

		bytes := make([]byte, 8)
		for {
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
			_, err = fileWrite.Write(bytes)
			if err != nil {
				logTrack.Error("Fail error", "error", err)
				return err
			}
		}

		trackObjectWorkTree.Add(filePath)
	}

	status, _ := trackObjectWorkTree.Status()
	if !status.IsClean() {
		fmt.Println(status.String())
		msg := fmt.Sprintf("track-hash: %s", vendorHash.Hash().String())
		_, err := trackObjectWorkTree.Commit(msg, &git.CommitOptions{All: true})
		if err != nil {
			return err
		}
	}

	trackLog, _ := t.trackObjects.Log(&git.LogOptions{})
	c, _ := trackLog.Next()
	fmt.Println(c)
	// trackLog.ForEach(func(c *object.Commit) error {
	// 	fmt.Println(c)
	// 	return nil
	// })

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

func initRepository(repositoryPath string) (*git.Repository, error) {
	var r *git.Repository
	var err error

	wt := osfs.New("tmp/project")
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
	track := &Track{
		pathObjects: cmd.Folder,
	}
	logTrack.Debug("cmd", "cmd", cmd)

	repositoryName, err := repositoryName(cmd.Repository)
	if err != nil {
		logTrack.Error("Fail extract repository name", "err", err, "repository", cmd.Repository)
		return 1
	}

	logTrack.Debug("Repository name extracted", "repositoryName", repositoryName)

	repositoryPath := filepath.Join(setting.TrackFolder, "vendor", repositoryName)
	track.vendor, err = cloneRepository(cmd.Repository, repositoryPath)
	if err != nil {
		logTrack.Error("Fail start repository", "err", err, "repositoryPath", repositoryPath)
		return 1
	}

	repositoryPath = filepath.Join(setting.TrackFolder, "track-objects")
	track.trackObjects, err = initRepository(repositoryPath)
	if err != nil {
		logTrack.Error("Fail start repository", "err", err, "repositoryPath", repositoryPath)
		return 1
	}

	track.trackObjects.Head()

	refVendor, err := track.vendor.Head()
	if err != nil {
		logTrack.Error("Fail get head from repository", "err", err)
		return 1
	}

	logTrack.Debug("Head hash", "refHash", refVendor.Hash().String(), "refName", refVendor.Name())

	track.trackObject("public/app/core/components/NavBar")

	return 0
}

func repositoryName(repositoryURL string) (string, error) {
	u, err := url.Parse(repositoryURL)
	if err != nil {
		return "", err
	}
	paths := strings.Split(u.Path, "/")

	return paths[len(paths)-1], nil
}
