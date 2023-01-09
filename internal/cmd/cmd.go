package cmd

import (
	"os"

	"github.com/galgotech/fhub-track/internal/log"
	"github.com/galgotech/fhub-track/internal/setting"
	"github.com/galgotech/fhub-track/internal/track"
	"github.com/urfave/cli/v2"
)

var logCmd = log.New("cmd")

type Cmd struct {
	RepoSrc string
	RepoDst string

	Status              bool
	Track               string
	TrackRename         string
	TrackIgnoreModified bool
	TrackUpdate         bool
}

func New(setting *setting.Setting) error {

	app := &cli.App{
		Name:  "fhub-track",
		Usage: "Fork a git repository with only the necessary folder or files.",
		Authors: []*cli.Author{
			{
				Name:  "GalgoTech",
				Email: "andre@galgo.tech",
			},
		},
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:     "src",
				Aliases:  []string{"s"},
				Usage:    "Source repository",
				Required: true,
				Action: func(c *cli.Context, path cli.Path) error {
					setting.SrcRepo = path
					return nil
				},
			},
			&cli.PathFlag{
				Name:     "dst",
				Aliases:  []string{"d"},
				Usage:    "Destination repository",
				Required: true,
				Action: func(c *cli.Context, path cli.Path) error {
					setting.DstRepo = path
					return nil
				},
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "object",
				Usage: "Track objects",
				Action: func(c *cli.Context) error {
					var arg1, arg2 string
					if c.NArg() == 1 {
						arg1 = c.Args().Get(0)
						arg2 = c.Args().Get(0)
					} else if c.NArg() > 1 {
						arg1 = c.Args().Get(0)
						arg2 = c.Args().Get(1)
					}
					return track.Object(setting, arg1, arg2)
				},
			},
			{
				Name:  "rename",
				Usage: "rename objects (folder ou file) <old_path> <new_path>",
				Action: func(c *cli.Context) error {
					if c.NArg() == 2 {
						old := c.Args().Get(0)
						new := c.Args().Get(1)
						return track.Rename(setting, old, new)
					}
					return nil
				},
			},
			{
				Name:  "status",
				Usage: "Objects status",
				Action: func(c *cli.Context) error {
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		return err
	}

	return nil

	// cmd := &Cmd{}

	// flag.StringVar(&cmd.RepoSrc, "repo-src", "", "Work tree src")
	// flag.StringVar(&cmd.RepoDst, "repo-dst", "", "Work tree dst")

	// flag.StringVar(&cmd.Track, "track", "", "Track objects")
	// flag.StringVar(&cmd.TrackRename, "track-rename", "", "Track rename object")
	// flag.BoolVar(&cmd.TrackIgnoreModified, "ignore-modified", false, "Track ignore modified files")
	// flag.BoolVar(&cmd.Status, "status", false, "Status objects")
	// flag.BoolVar(&cmd.TrackUpdate, "track-update", false, "Update objects")

	// help := flag.Bool("help", false, "Help")

	// flag.Parse()

	// logCmd.Debug("Arguments", "cmd", cmd)

	// if cmd.RepoSrc == "" {
	// 	*help = true
	// }

	// if *help {
	// 	flag.PrintDefaults()
	// 	return nil, 0
	// }

	// return cmd, 0
}
