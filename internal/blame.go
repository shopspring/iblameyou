package internal

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Commit struct {
	ID          string
	Author      string
	Email       string
	Message     string
	FullMessage string

	// Date is the date when this commit was originally made. (It may
	// differ from the commit date, which is changed during rebases, etc.)
	Date time.Time
}

type Commits struct {
	All      []*Commit
	BySource map[string]*Commit
	ByID     map[string]*Commit
}

func DefaultCommits() Commits {
	return Commits{
		BySource: map[string]*Commit{},
		ByID:     map[string]*Commit{},
	}
}

func (cms *Commits) Add(source string, cm Commit) {
	ptr := &cm
	if found, ok := cms.ByID[cm.ID]; ok {
		ptr = found
	} else {
		cms.All = append(cms.All, ptr)
		cms.ByID[ptr.ID] = ptr
	}
	cms.BySource[source] = ptr
}

func (cms *Commits) SortByDate() {
	sort.Stable(byDate(cms.All))
}

type byDate []*Commit

func (c byDate) Len() int           { return len(c) }
func (c byDate) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c byDate) Less(i, j int) bool { return c[i].Date.After(c[j].Date) }

// Blame returns the commit information about the person that committed the
// single line in given file in given repository.
// Heavily influenced by https://github.com/sourcegraph/go-blame/
func Blame(repo, file string, line int, revision string) (Commit, error) {
	lineOpt := fmt.Sprintf("-L%[1]d,%[1]d", line)
	cmd := exec.Command("git", "blame", "-w", "--porcelain", lineOpt, revision,
		"--", file)
	cmd.Dir = repo
	cmd.Stderr = ioutil.Discard
	out, err := cmd.Output()
	if err != nil {
		return Commit{}, err
	}
	if len(out) < 1 {
		// go 1.8.5 changed the behavior of `git blame` on empty files.
		// previously, it returned a boundary commit. now, it returns nothing.
		// TODO(sqs) TODO(beyang): make `git blame` return the boundary commit
		// on an empty file somehow, or come up with some other workaround.
		st, err := os.Stat(filepath.Join(repo, file))
		if err == nil && st.Size() == 0 {
			return Commit{}, nil
		}
		return Commit{}, fmt.Errorf("Expected git output of length at least 1")
	}

	lines := strings.Split(string(out[:len(out)-1]), "\n")
	// Consume hunk
	hunk := strings.Split(lines[0], " ")
	if len(hunk) != 4 {
		return Commit{},
			fmt.Errorf("Expected at least 4 parts to hunk, but got: '%s'", hunk)
	}
	commitID := hunk[0]

	// Consume commit
	author := strings.Join(strings.Split(lines[1], " ")[1:], " ")
	email := strings.Join(strings.Split(lines[2], " ")[1:], " ")
	if len(email) >= 2 && email[0] == '<' && email[len(email)-1] == '>' {
		email = email[1 : len(email)-1]
	}
	date, err := strconv.ParseInt(strings.Join(strings.Split(lines[3], " ")[1:], " "), 10, 64)
	if err != nil {
		return Commit{}, fmt.Errorf("Failed to parse author-time %q", lines[3])
	}
	summary := strings.Join(strings.Split(lines[9], " ")[1:], " ")
	commit := Commit{
		ID:      commitID,
		Message: summary,
		Author:  author,
		Email:   email,
		Date:    time.Unix(date, 0),
	}

	// Get the full message
	cmd = exec.Command("git", "show", "-s", "--format=%B", commit.ID)
	cmd.Dir = repo
	cmd.Stderr = ioutil.Discard
	out, err = cmd.Output()
	if err == nil {
		commit.FullMessage = strings.TrimSpace(string(out))
	}

	return commit, nil
}
