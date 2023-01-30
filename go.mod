module github.com/galgotech/fhub-track

go 1.19

require (
	github.com/go-git/go-git/v5 v5.4.2
	github.com/libgit2/git2go/v34 v34.0.0
	github.com/rs/zerolog v1.28.0
	github.com/urfave/cli/v2 v2.23.7
)

replace github.com/libgit2/git2go/v34 => ./git2go

require (
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	golang.org/x/crypto v0.0.0-20210421170649-83a5a9bb288b // indirect
	golang.org/x/sys v0.4.0 // indirect
	golang.org/x/term v0.4.0 // indirect
)
