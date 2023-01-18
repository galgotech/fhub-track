package track

import (
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"

	"github.com/galgotech/fhub-track/internal/log"
	"github.com/galgotech/fhub-track/internal/setting"
)

var logTrack = log.New("track")

type Track struct {
	srcRepository *git.Repository
	srcConfig     *config.Config
	srcWorkTree   *git.Worktree

	dstRepository *git.Repository
	dstWorkTree   *git.Worktree
}

func Object(setting *setting.Setting, srcObject, dstObject string) error {
	track, err := initTrack(setting)
	if err != nil {
		return err
	}

	err = track.trackObject(srcObject, dstObject)
	if err != nil {
		logTrack.Error("Track object fail", "object", srcObject, "error", err.Error())
		return err
	}

	return nil
}

func Rename(setting *setting.Setting, old string, new string) error {
	track, err := initTrack(setting)
	if err != nil {
		return err
	}

	err = track.trackRenameObject(old, new)
	if err != nil {
		logTrack.Error("Rename object fail", "old", old, "new", new, "error", err.Error())
		return err
	}

	return nil
}

func Update(setting *setting.Setting) error {
	track, err := initTrack(setting)
	if err != nil {
		return err
	}

	err = track.trackUpdate()
	if err != nil {
		logTrack.Error("Track update fail", "error", err.Error())
		return err
	}

	return nil
}

func Status(setting *setting.Setting) error {
	track, err := initTrack(setting)
	if err != nil {
		return err
	}

	err = track.status()
	if err != nil {
		logTrack.Error("Status fail", "error", err.Error())
		return err
	}

	return nil
}

func initTrack(setting *setting.Setting) (*Track, error) {
	var err error
	track := &Track{}

	// Source repository
	track.srcRepository, err = initRepository(filepath.Join(setting.RootPath, setting.SrcRepo))
	if err != nil {
		logTrack.Error("Fail start src repository", "err", err.Error(), "repositoryPath", setting.SrcRepo)
		return nil, err
	}

	track.srcConfig, err = track.srcRepository.Config()
	if err != nil {
		logTrack.Error("Fail get src repository config", "err", err.Error())
		return nil, err
	}

	track.srcWorkTree, err = track.srcRepository.Worktree()
	if err != nil {
		logTrack.Error("Fail get src repository worktree", "err", err.Error())
		return nil, err
	}

	// Destionation repository
	track.dstRepository, err = initRepository(filepath.Join(setting.RootPath, setting.DstRepo))
	if err != nil {
		logTrack.Error("Fail start dst repository", "err", err, "WorkTree", setting.DstRepo)
		return nil, err
	}

	track.dstWorkTree, err = track.dstRepository.Worktree()
	if err != nil {
		logTrack.Error("Fail get dst repository worktree", "err", err.Error())
		return nil, err
	}

	return track, nil
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
