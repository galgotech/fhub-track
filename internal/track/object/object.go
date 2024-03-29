package object

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/galgotech/fhub-track/internal/log"
	"github.com/galgotech/fhub-track/internal/track/utils"
	git "github.com/libgit2/git2go/v34"
)

type Object struct {
	src, dst *git.Repository
}

func New(src, dst *git.Repository) *Object {
	return &Object{src, dst}
}

var logTrack = log.New("track-object")

func (t *Object) Run(srcObject, dstObject string) error {
	logTrack.Info("start track object", "srcObject", srcObject, "dstObject", dstObject)

	c, err := utils.StatusEntryCount(t.dst)
	if err != nil {
		return err
	}
	if c > 0 {
		return errors.New("the destination repository has files change")
	}

	allSrcObjects, err := t.searchObjectsInWorkTree(srcObject)
	if err != nil {
		return err
	}

	allDstObjects := renameObjectsToDst(allSrcObjects, srcObject, dstObject)

	err = t.copyObject(allSrcObjects, allDstObjects)
	if err != nil {
		return err
	}

	index, err := t.dst.Index()
	if err != nil {
		return err
	}

	for _, object := range allDstObjects {
		err := index.AddByPath(object)
		if err != nil {
			return err
		}
	}

	err = index.Write()
	if err != nil {
		return err
	}

	treeOid, err := index.WriteTree()
	if err != nil {
		return err
	}

	tree, err := t.dst.LookupTree(treeOid)
	if err != nil {
		return err
	}

	c, err = utils.StatusEntryCount(t.dst)
	if err != nil {
		return err
	}
	if c > 0 {
		zipObjects, err := zipObjects(allSrcObjects, allDstObjects)
		if err != nil {
			return err
		}

		var parents []*git.Commit
		head, err := t.dst.Head()
		if err == nil {
			commit, err := t.dst.LookupCommit(head.Target())
			if err != nil {
				return err
			}
			parents = append(parents, commit)
		}

		msg := fmt.Sprintf("files:\n  %s", strings.Join(zipObjects, "\n  "))
		_, err = utils.Commit(t.dst, msg, tree, parents...)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *Object) searchObjectsInWorkTree(object string) ([]string, error) {
	allObjects := []string{}

	objectPath := filepath.Join(t.src.Workdir(), object)
	file, err := os.Open(objectPath) // For read access.
	if err != nil {
		return nil, err
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}

	if !fileInfo.IsDir() {
		return []string{object}, nil
	}

	subObjectsInfo, err := os.ReadDir(objectPath)
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

func (t *Object) copyObject(allSrcObjects, allDstObjects []string) error {
	if len(allSrcObjects) != len(allDstObjects) {
		return errors.New("allSrcObjects and allDstObjects have different length")
	}

	srcWorkdir := t.src.Workdir()
	dstWorkdir := t.dst.Workdir()

	for i := 0; i < len(allSrcObjects); i++ {
		srcObject := filepath.Join(srcWorkdir, allSrcObjects[i])
		dstObject := filepath.Join(dstWorkdir, allDstObjects[i])

		logTrack.Debug("copy object", "src", srcObject, "dst", dstObject)

		fileRead, err := os.OpenFile(srcObject, os.O_RDONLY, os.ModeAppend)
		if err != nil {
			return err
		}
		defer fileRead.Close()

		err = os.MkdirAll(filepath.Dir(dstObject), 0750)
		if err != nil {
			return err
		}

		fileWrite, err := os.OpenFile(dstObject, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
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

func renameObjectsToDst(objects []string, srcObject, dstObject string) []string {
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
