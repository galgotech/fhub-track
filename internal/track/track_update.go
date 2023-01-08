package track

import (
	"errors"

	"github.com/galgotech/gotools/diff"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type fileHash struct {
	vendorHash plumbing.Hash
	trackHash  plumbing.Hash
}

type objectName struct {
	src string
	dst string
}

func (t *Track) trackUpdate() error {
	tracked, err := t.searchTrackedObjects()
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
				vendorHash: track.srcCommit,
				trackHash:  track.dstCommit,
			})
		}

	}

	for objectName, hash := range objects {
		vendorContent, err := getCommitFileContent(t.srcRepository, hash[0].vendorHash, objectName.src)
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

		trackContent, err := getCommitFileContent(t.dstRepository, hash[0].trackHash, objectName.dst)
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

func getObjectName(objectsName map[string]*objectName, name string) *objectName {
	if _, ok := objectsName[name]; !ok {
		objectsName[name] = &objectName{
			src: name,
			dst: name,
		}
	}

	return objectsName[name]
}
