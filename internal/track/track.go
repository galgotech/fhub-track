package track

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"

	"github.com/galgotech/fhub-track/internal/cmd"
	"github.com/galgotech/fhub-track/internal/log"
	"github.com/galgotech/fhub-track/internal/setting"
)

var logTrack = log.New("track")

type Track struct {
	vendor         *git.Repository
	vendorWorkTree *git.Worktree
	vendorHash     *plumbing.Reference
	vendorConfig   *config.Config

	trackObjects         *git.Repository
	trackObjectsWorkTree *git.Worktree
	trackObjectsConfig   *config.Config
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
