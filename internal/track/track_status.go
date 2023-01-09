package track

import "fmt"

func (t *Track) status() error {
	status, err := t.dstWorkTree.Status()
	if err != nil {
		return err
	}

	fmt.Println(status)

	return nil
}
