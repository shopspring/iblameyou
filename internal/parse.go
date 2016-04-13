package internal

import (
	"bytes"
	"io"
	"io/ioutil"
	"os/exec"
	"strings"

	"github.com/maruel/panicparse/stack"
)

type Dump struct {
	Head    string
	Buckets stack.Buckets
	Commits Commits
	Skipped string

	source *Source
}

type Source struct {
	Repository string
}

func (s *Source) ParseDump(message io.Reader) (Dump, error) {
	skip := new(bytes.Buffer)
	routines, err := stack.ParseDump(message, skip)
	if err != nil {
		return Dump{}, err
	}

	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = s.Repository
	cmd.Stderr = ioutil.Discard
	head, err := cmd.Output()
	if err != nil {
		head = nil
	}

	dump := Dump{
		Head:    strings.TrimSpace(string(head)),
		Buckets: stack.SortBuckets(stack.Bucketize(routines, stack.AnyPointer)),
		Commits: DefaultCommits(),
		Skipped: skip.String(),
		source:  s,
	}

	for _, b := range dump.Buckets {
		for _, c := range b.Stack.Calls {
			cm, err := Blame(s.Repository, c.SourcePath, c.Line)
			if err == nil {
				dump.Commits.Add(c.FullSourceLine(), cm)
			}
		}
	}
	dump.Commits.SortByDate()
	return dump, nil
}
