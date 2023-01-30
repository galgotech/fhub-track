package update

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	git "github.com/libgit2/git2go/v34"
)

func recountPatch(ancestorBuffer []byte, patch *git.Patch) (string, error) {
	patchString, err := patch.String()
	if err != nil {
		return "", err
	}

	index := strings.Index(patchString, "@")
	newPatch := patchString[:index]

	numHunks, err := patch.NumHunks()
	if err != nil {
		return "", err
	}
	for i := 0; i < numHunks; i++ {
		diffHunk, numLines, err := patch.Hunk(i)
		if err != nil {
			return "", err
		}

		hunk := ""
		hunkSearch := ""
		for j := 0; j < numLines; j++ {
			diffLine, err := patch.HunkLine(i, j)
			if err != nil {
				return "", err
			}

			hunkContent := diffLine.Content
			switch diffLine.Origin {
			case git.DiffLineContext:
				hunk = fmt.Sprintf("%s%c", hunk, ' ')
			case git.DiffLineAddition:
				hunk = fmt.Sprintf("%s%c", hunk, '+')
			case git.DiffLineDeletion:
				hunk = fmt.Sprintf("%s%c", hunk, '-')
			}
			hunk = fmt.Sprintf("%s%s", hunk, hunkContent)
			if diffLine.Origin == git.DiffLineContext || diffLine.Origin == git.DiffLineDeletion {
				hunkSearch += hunkContent
			}
		}

		index := strings.Index(string(ancestorBuffer), hunkSearch)
		if index == -1 {
			return "", errors.New("patch conflict")
		}

		line := 1
		for j := 0; j < index; j++ {
			if ancestorBuffer[j] == '\n' || (ancestorBuffer[j] == '\r' && ancestorBuffer[j+1] == '\n') {
				line++
			}
		}

		newDiffHunkHeader := rewriteDiffHunkHeader(diffHunk, line)
		newPatch = fmt.Sprintf("%s%s%s", newPatch, newDiffHunkHeader, hunk)
	}

	return newPatch, nil
}

func rewriteDiffHunkHeader(hunk git.DiffHunk, line int) string {
	// format: @@ -112,12 +112,15 @@ func (hs *HTTPServer) registerRoutes() {
	header := hunk.Header
	if header[0] != '@' || header[1] != '@' || header[2] != ' ' {
		return ""
	}

	i := 4
	oldStart, err := readNumber(header, &i, ',')
	if err != nil {
		return ""
	}
	oldLine, err := readNumber(header, &i, ' ')
	if err != nil {
		return ""
	}
	newStart, err := readNumber(header, &i, ',')
	if err != nil {
		return ""
	}
	newLine, err := readNumber(header, &i, ' ')
	if err != nil {
		return ""
	}
	if header[i] != '@' || header[i+1] != '@' || header[i+2] != ' ' {
		return ""
	}
	i += 3

	return fmt.Sprintf("@@ -%d,%d +%d,%d @@ %s", line, oldLine, line+(newStart-oldStart), newLine, header[i:])
}

func readNumber(header string, i *int, stop byte) (int, error) {
	number := ""
	for header[*i] != stop && *i < len(header) {
		number += string(header[*i])
		*i++
	}
	*i++
	return strconv.Atoi(number)
}
