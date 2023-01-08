package track

import (
	"errors"
	"strings"

	"github.com/galgotech/gotools/diff"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

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

func getObjectName(objectsName map[string]*objectName, name string) *objectName {
	if _, ok := objectsName[name]; !ok {
		objectsName[name] = &objectName{
			src: name,
			dst: name,
		}
	}

	return objectsName[name]
}
