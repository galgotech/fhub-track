package track

import "fmt"

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
