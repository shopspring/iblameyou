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
	Revision string
	Buckets  stack.Buckets
	Commits  Commits
	Skipped  string

	source *Source
}

type Source struct {
	Repository string `yaml:"repository,omitempty"`
	Revision   string `yaml:"revision,omitempty"`
}

func (s *Source) ParseDump(message io.Reader) (Dump, error) {
	skip := new(bytes.Buffer)
	routines, err := stack.ParseDump(message, skip)
	if err != nil {
		return Dump{}, err
	}

	if s.Revision == "" {
		s.Revision = "HEAD"
	}
	cmd := exec.Command("git", "rev-parse", s.Revision)
	cmd.Dir = s.Repository
	cmd.Stderr = ioutil.Discard
	revision, err := cmd.Output()
	if err != nil {
		revision = []byte("HEAD")
	}

	dump := Dump{
		Revision: strings.TrimSpace(string(revision)),
		Buckets:  stack.SortBuckets(stack.Bucketize(routines, stack.AnyPointer)),
		Commits:  DefaultCommits(),
		Skipped:  skip.String(),
		source:   s,
	}

	for _, b := range dump.Buckets {
		for _, c := range b.Stack.Calls {
			cm, err := Blame(s.Repository, c.SourcePath, c.Line, dump.Revision)
			if err == nil {
				dump.Commits.Add(c.FullSourceLine(), cm)
			}
		}
	}
	dump.Commits.SortByDate()
	return dump, nil
}
