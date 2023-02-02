package update

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	git "github.com/libgit2/git2go/v34"
)

type hunkMatch struct {
	baseLine []int
	origin   git.DiffLineType
	line     int
	content  string
}

// Naive implementation need optimization
func recountPatch(base string, patch *git.Patch) (string, error) {
	newPatch, err := rewritePatchHeader(patch)
	if err != nil {
		return "", err
	}

	numHunks, err := patch.NumHunks()
	if err != nil {
		return "", err
	}
	for i := 0; i < numHunks; i++ {
		diffHunk, numHunkLines, err := patch.Hunk(i)
		if err != nil {
			return "", err
		}

		baseLines := strings.SplitAfter(base, "\n")
		M := len(baseLines)

		// Search string match
		hunkMatch := make([]hunkMatch, numHunkLines)
		for j := 0; j < numHunkLines; j++ {
			diffLine, err := patch.HunkLine(i, j)
			if err != nil {
				return "", err
			}
			hunkMatch[j].origin = diffLine.Origin
			hunkMatch[j].line = j
			hunkMatch[j].content = diffLine.Content

			if diffLine.Origin == git.DiffLineContext || diffLine.Origin == git.DiffLineDeletion {
				for m := 0; m < M; m++ {
					if baseLines[m] == diffLine.Content {
						hunkMatch[j].baseLine = append(hunkMatch[j].baseLine, m)
					}
				}
			}
		}

		// Check has conflicts
		conflicts := []int{}
		for j, p := range hunkMatch {
			if len(p.baseLine) != 1 {
				conflicts = append(conflicts, j)
			}
		}

		// Resolve conflicts
		for len(conflicts) > 0 {
			var a, b int
			if conflicts[0] == len(hunkMatch)-1 {
				a = conflicts[0]
				b = conflicts[0] - 1
			} else {
				a = conflicts[0]
				i := 1
				for len(hunkMatch[conflicts[0]+i].baseLine) != 1 {
					i++
				}
				b = conflicts[0] + i
			}

			if len(hunkMatch[a].baseLine) > 0 {
				k := 0
				for k < len(hunkMatch[a].baseLine) && hunkMatch[b].baseLine[0] > hunkMatch[a].baseLine[k] {
					k++
				}
				if conflicts[0] != len(hunkMatch)-1 {
					k--
				}
				hunkMatch[a].baseLine = []int{hunkMatch[a].baseLine[k]}
			} else {
				hunkMatch[a].baseLine = []int{-1}
			}

			conflicts = conflicts[1:]
		}

		k := hunkMatch[0].baseLine[0]
		for j := 1; j < numHunkLines; j++ {
			if hunkMatch[j].baseLine[0] != -1 {
				k++
			}
		}

		hunk := ""
		contextLine := 0
		addLine := 0
		oldLine := 0
		line := hunkMatch[0].baseLine[0]
		for j := 0; j < numHunkLines; j++ {
			c := ' '
			switch hunkMatch[j].origin {
			case git.DiffLineContext:
				c = ' '
				contextLine++
			case git.DiffLineAddition:
				c = '+'
				addLine++
			case git.DiffLineDeletion:
				c = '-'
				oldLine++
			}

			if hunkMatch[j].baseLine[0] == -1 {
				if hunkMatch[j].origin == git.DiffLineAddition {
					hunk = fmt.Sprintf("%s%c%s", hunk, c, hunkMatch[j].content)
				} else if hunkMatch[j].origin == git.DiffLineDeletion {
					oldLine++
					hunk = fmt.Sprintf("%s>%c%s", hunk, c, hunkMatch[j].content)
				} else {
					contextLine--
					hunk = fmt.Sprintf("%s>%c%s", hunk, c, hunkMatch[j].content)
				}
			} else {
				if hunkMatch[j].baseLine[0]-line > 1 {
					line++
					for ; line < hunkMatch[j].baseLine[0]; line++ {
						hunk = fmt.Sprintf("%s<+%s", hunk, baseLines[line])
					}
				}
				line = hunkMatch[j].baseLine[0]
				hunk = fmt.Sprintf("%s%c%s", hunk, c, hunkMatch[j].content)
			}
		}

		oldLine = contextLine + oldLine
		newLine := contextLine + addLine

		if hunkMatch[len(hunkMatch)-1].baseLine[0] != k {
			fmt.Println("-- conflict")
			fmt.Println(hunk)
			os.Exit(0)
			// return "", ErrPatchConflict
		}

		newDiffHunkHeader := rewriteDiffHunkHeader(diffHunk, hunkMatch[0].baseLine[0]+1, oldLine, newLine)
		newPatch = fmt.Sprintf("%s%s%s", newPatch, newDiffHunkHeader, hunk)

	}

	fmt.Println(newPatch)

	return newPatch, nil
}

func rewritePatchHeader(patch *git.Patch) (string, error) {
	patchString, err := patch.String()
	if err != nil {
		return "", err
	}

	index := strings.Index(patchString, "@")
	newPatch := patchString[:index]

	return newPatch, nil
}

func rewriteDiffHunkHeader(hunk git.DiffHunk, line, oldLine, newLine int) string {
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
	_, err = readNumber(header, &i, ' ')
	if err != nil {
		return ""
	}
	newStart, err := readNumber(header, &i, ',')
	if err != nil {
		return ""
	}
	_, err = readNumber(header, &i, ' ')
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
