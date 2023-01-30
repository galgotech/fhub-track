package status

import (
	"fmt"

	"github.com/galgotech/fhub-track/internal/track/utils"
	git "github.com/libgit2/git2go/v34"
)

func New(src, dst *git.Repository) *Status {
	return &Status{src, dst}
}

type Status struct {
	src, dst *git.Repository
}

func (t *Status) Run() error {
	status, err := utils.Status(t.dst)
	if err != nil {
		return err
	}

	fmt.Println(status)

	return nil
}
