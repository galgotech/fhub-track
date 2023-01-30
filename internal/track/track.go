package track

import (
	"path/filepath"

	git "github.com/libgit2/git2go/v34"

	"github.com/galgotech/fhub-track/internal/log"
	"github.com/galgotech/fhub-track/internal/setting"
	"github.com/galgotech/fhub-track/internal/track/object"
	"github.com/galgotech/fhub-track/internal/track/rename"
	"github.com/galgotech/fhub-track/internal/track/status"
	"github.com/galgotech/fhub-track/internal/track/update"
)

var logTrack = log.New("track")

type Track struct {
	src *git.Repository
	dst *git.Repository
}

func Object(setting *setting.Setting, srcObject, dstObject string) error {
	src, dst, err := initRepos(setting)
	if err != nil {
		return err
	}

	o := object.New(src, dst)
	err = o.Run(srcObject, dstObject)
	if err != nil {
		logTrack.Error("Track object fail", "object", srcObject, "error", err.Error())
		return err
	}

	return nil
}

func Rename(setting *setting.Setting, old string, new string) error {
	src, dst, err := initRepos(setting)
	if err != nil {
		return err
	}

	r := rename.New(src, dst)
	err = r.Run(old, new)
	if err != nil {
		logTrack.Error("Rename object fail", "old", old, "new", new, "error", err.Error())
		return err
	}

	return nil
}

func Update(setting *setting.Setting) error {
	src, dst, err := initRepos(setting)
	if err != nil {
		return err
	}

	u := update.New(src, dst)

	err = u.Run()
	if err != nil {
		logTrack.Error("Update fail", "error", err.Error())
		return err
	}

	return nil
}

func Status(setting *setting.Setting) error {
	src, dst, err := initRepos(setting)
	if err != nil {
		return err
	}

	s := status.New(src, dst)
	err = s.Run()
	if err != nil {
		logTrack.Error("Status fail", "error", err.Error())
		return err
	}

	return nil
}

func initRepos(setting *setting.Setting) (*git.Repository, *git.Repository, error) {
	// Source repository
	src, err := git.OpenRepository(filepath.Join(setting.RootPath, setting.SrcRepo))
	if err != nil {
		logTrack.Error("Fail start src repository", "err", err.Error(), "repositoryPath", setting.SrcRepo)
		return nil, nil, err
	}

	// Destionation repository
	dst, err := git.OpenRepository(filepath.Join(setting.RootPath, setting.DstRepo))
	if err != nil {
		logTrack.Error("Fail start dst repository", "err", err, "WorkTree", setting.DstRepo)
		return nil, nil, err
	}

	return src, dst, nil
}
