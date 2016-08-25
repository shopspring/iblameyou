//go:generate esc -private -pkg internal -o templates.go templates

package internal

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/maruel/panicparse/stack"
)

type Palette struct {
	FunctionStdLib         string `yaml:"function_std_lib,omitempty"`
	FunctionStdLibExported string `yaml:"function_std_lib_exported,omitempty"`
	FunctionMain           string `yaml:"function_main,omitempty"`
	FunctionOther          string `yaml:"function_other,omitempty"`
	FunctionOtherExported  string `yaml:"function_other_exported,omitempty"`

	Routine      string `yaml:"routine,omitempty"`
	RoutineFirst string `yaml:"routine_first,omitempty"`

	Package    string `yaml:"package,omitempty"`
	SourceFile string `yaml:"source_file,omitempty"`
	Arguments  string `yaml:"arguments,omitempty"`

	CommitID   string `yaml:"commit_id,omitempty"`
	CommitDate string `yaml:"commit_date,omitempty"`
}

func DefaultPalette() Palette {
	return Palette{
		FunctionStdLib:         "fg-green",
		FunctionStdLibExported: "fg-green,fg-bold",
		FunctionMain:           "fg-yellow,fg-bold",
		FunctionOther:          "fg-red",
		FunctionOtherExported:  "fg-red,fg-bold",

		Routine:      "fg-magenta",
		RoutineFirst: "fg-magenta,fg-bold",

		Package:    "fg-white,fg-bold",
		SourceFile: "fg-white",
		Arguments:  "fg-white",

		CommitID:   "fg-white,fg-bold",
		CommitDate: "fg-white",
	}
}

type Format struct {
	CommitURL     string `yaml:"commit_url,omitempty"`
	FileURL       string `yaml:"file_url,omitempty"`
	BlameURL      string `yaml:"blame_url,omitempty"`
	CustomMessage string `yaml:"custom_message,omitempty"`

	FullPath bool `yaml:"full_path,omitempty"`

	Colors Palette `yaml:"colors,omitempty"`

	templates struct {
		CommitURL *template.Template
		FileURL   *template.Template
		BlameURL  *template.Template
		Message   *template.Template
	}
}

func (f *Format) Init() error {
	var err error
	f.templates.CommitURL, err = template.New("CommitURL").Parse(f.CommitURL)
	if err != nil {
		return err
	}
	f.templates.FileURL, err = template.New("FileURL").Parse(f.FileURL)
	if err != nil {
		return err
	}
	f.templates.BlameURL, err = template.New("BlameURL").Parse(f.BlameURL)
	if err != nil {
		return err
	}
	if f.CustomMessage == "" {
		f.templates.Message, err = message.Clone()
	} else {
		f.templates.Message, err =
			template.New("CustomMessage").
				Funcs(messageFuncs).
				Parse(f.CustomMessage)
	}
	return err
}

func (f *Format) format(t *template.Template, v interface{}) string {
	data := struct {
		Format *Format
		V      interface{}
	}{f, v}
	b := new(bytes.Buffer)
	err := t.Execute(b, data)
	if err != nil {
		panic(err)
	}
	return b.String()
}

func (f *Format) Commits(c []*Commit) []string {
	commits := make([]string, len(c))
	for i := range c {
		commits[i] =
			fmt.Sprintf(`[%s] %s`, c[i].Date.Format("2006-02-01 15:04:05"), c[i].ID)
	}
	return commits
}

type SourcePath struct {
	Head     string
	File     string
	Line     int
	CommitID string
}

// functionColor returns the color to be used for the function name based on
// the type of package the function is in.
func (f *Format) functionColor(line *stack.Call) string {
	p := &f.Colors
	if line.IsStdlib() {
		if line.Func.IsExported() {
			return p.FunctionStdLibExported
		}
		return p.FunctionStdLib
	} else if line.IsPkgMain() {
		return p.FunctionMain
	} else if line.Func.IsExported() {
		return p.FunctionOtherExported
	}
	return p.FunctionOther
}

// routineColor returns the color for the header of the goroutines bucket.
func (f *Format) routineColor(bucket *stack.Bucket, multipleBuckets bool) string {
	p := &f.Colors
	if bucket.First() && multipleBuckets {
		return p.RoutineFirst
	}
	return p.Routine
}

// BucketHeader prints the header of a goroutine signature.
func (f *Format) BucketHeader(bucket *stack.Bucket, multipleBuckets bool) string {
	extra := ""
	if bucket.SleepMax != 0 {
		if bucket.SleepMin != bucket.SleepMax {
			extra += fmt.Sprintf(" (%d~%d minutes)", bucket.SleepMin, bucket.SleepMax)
		} else {
			extra += fmt.Sprintf(" (%d minutes)", bucket.SleepMax)
		}
	}
	if bucket.Locked {
		extra += " (locked)"
	}
	created := bucket.CreatedBy.Func.PkgDotName()
	if created != "" {
		created += " @ "
		created += bucket.CreatedBy.SourceLine()
		extra += " (Created by " + created + ")"
	}
	return fmt.Sprintf(
		"[%d: %s%s](%s)",
		len(bucket.Routines), bucket.State, extra,
		f.routineColor(bucket, multipleBuckets))
}

func colorf(color, format string, a ...interface{}) string {
	s := fmt.Sprintf(format, a...)
	if color == "" {
		return s
	}
	return fmt.Sprintf("[%s](%s)", s, color)
}

// callLine prints one stack line.
func (f *Format) callLine(line *stack.Call, commit *Commit, srcLen int) string {
	p := &f.Colors
	id := "????"
	date := "????-??-??"
	if commit != nil {
		id = commit.ID[:4]
		date = commit.Date.Format("2006-01-02")
	}

	var srcLine string
	if f.FullPath {
		srcLine = line.FullSourceLine()
	} else {
		srcLine = line.SourceLine()
	}

	return fmt.Sprintf(
		"    {%s @ %s} %s  %s%s",
		colorf(p.CommitDate, "%s", date),
		colorf(p.CommitID, "%s", id),
		colorf(p.SourceFile, "%-*s", srcLen, srcLine),
		colorf(f.functionColor(line), "%s", line.Func.Name()),
		colorf(p.Arguments, "(%s)", line.Args))
}

// StackLines prints one complete stack trace, without the header.
func (f *Format) StackLines(head string, signature *stack.Signature,
	commits *Commits, srcLen int) ([]string, []SourcePath) {
	lines := make([]string, len(signature.Stack.Calls))
	files := make([]SourcePath, len(signature.Stack.Calls))
	for i, c := range signature.Stack.Calls {
		commit := commits.BySource[c.FullSourceLine()]
		lines[i] = f.callLine(&c, commit, srcLen)
		files[i] = SourcePath{Head: head, File: c.SourcePath, Line: c.Line}
		if commit != nil {
			files[i].CommitID = commit.ID
		}
	}
	if signature.Stack.Elided {
		lines = append(lines, "    (...)")
		files = append(files, SourcePath{})
	}
	return lines, files
}

func (f *Format) Stacktrace(d Dump) ([]string, []SourcePath) {
	lines := strings.Split(d.Skipped, "\n")
	files := make([]SourcePath, len(lines))

	srcLen, _ := stack.CalcLengths(d.Buckets, f.FullPath)
	for _, bucket := range d.Buckets {
		if len(lines) > 0 {
			lines = append(lines, "")
			files = append(files, SourcePath{})
		}
		lines = append(lines, f.BucketHeader(&bucket, len(d.Buckets) > 1))
		files = append(files, SourcePath{})
		stackLines, stackFiles :=
			f.StackLines(d.Head, &bucket.Signature, &d.Commits, srcLen)
		lines = append(lines, stackLines...)
		files = append(files, stackFiles...)
	}
	return lines, files
}

func (f *Format) StacktraceForMessage(d Dump) string {
	out := bytes.NewBufferString(d.Skipped)
	p := stack.Palette{}
	fullPath := true
	srcLen, pkgLen := stack.CalcLengths(d.Buckets, fullPath)
	for _, bucket := range d.Buckets {
		out.WriteString(p.BucketHeader(&bucket, fullPath, len(d.Buckets) > 1))
		out.WriteString(p.StackLines(&bucket.Signature, srcLen, pkgLen, fullPath))
	}
	return out.String()
}

func (f *Format) Commit(c *Commit) string {
	return f.format(commit, c)
}

type Candidate struct {
	Dump   *Dump
	Commit *Commit
}

func (f *Format) Message(c Candidate) string {
	return f.format(f.templates.Message, c)
}

var (
	commit = template.Must(template.New("commit").Parse(
		_escFSMustString(false, "/templates/commit.template")))
	message = template.Must(template.New("message").
		Funcs(messageFuncs).
		Parse(_escFSMustString(false, "/templates/message.template")))
	messageFuncs = template.FuncMap{
		"replace": func(s, old, new string) string {
			return strings.Replace(s, old, new, -1)
		},
	}
)
