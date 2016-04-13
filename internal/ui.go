package internal

import (
	"bytes"
	"text/template"

	"github.com/atotto/clipboard"
	"github.com/gizak/termui"
	"github.com/mwek/iblameyou/widgets"
	"github.com/toqueteos/webbrowser"
)

type UI struct {
	format  *Format
	widgets struct {
		commit     *termui.Par
		stackTrace *widgets.ScrollableList
		messages   *widgets.MessageBox
	}

	messages struct {
		status *widgets.Message
		usage  *widgets.Message
	}

	dump       *Dump
	stackTrace []SourcePath
}

func (ui *UI) Init(f *Format) error {
	err := f.Init()
	if err != nil {
		return err
	}
	ui.format = f

	err = termui.Init()
	if err != nil {
		return err
	}

	// Messages
	ui.messages.status = &widgets.Message{Ticks: 5}
	ui.messages.usage = &widgets.Message{
		Content: "[m]essage | [f]ile | [c]ommit | [b]lame | j/k scroll",
		Ticks:   -1}

	// Widgets
	ui.widgets.commit = termui.NewPar("")
	ui.widgets.commit.BorderLabel = "Commit"

	ui.widgets.stackTrace = widgets.NewScrollableList()
	ui.widgets.stackTrace.BorderLabel = "Stacktrace"
	ui.widgets.stackTrace.Items = []string{"Loading..."}

	ui.widgets.messages = widgets.NewMessageBox()
	ui.widgets.messages.AddMessage(ui.messages.usage, widgets.Right)

	ui.SetHeight(termui.TermHeight())

	// Handlers
	termui.Handle("/sys/kbd/q", func(termui.Event) {
		termui.StopLoop()
	})

	termui.Handle("/sys/kbd/k", func(termui.Event) {
		ui.widgets.stackTrace.SelectPrevious()
		ui.updateCommit()
		ui.refresh()
	})
	termui.Handle("/sys/kbd/j", func(termui.Event) {
		ui.widgets.stackTrace.SelectNext()
		ui.updateCommit()
		ui.refresh()
	})

	termui.Handle("/sys/kbd/m", func(termui.Event) {
		status := ""
		defer ui.showMessage(&status)

		file := ui.stackTrace[ui.widgets.stackTrace.CurrentItem]
		if file.CommitID == "" {
			status = "Error: no associated commit!"
			return
		}

		c := Candidate{
			Commit: ui.dump.Commits.ByID[file.CommitID],
			Dump:   ui.dump,
		}

		err = clipboard.WriteAll(f.Message(c))
		if err != nil {
			status = "Error: " + err.Error()
			return
		}
		status = "Message copied to clipboard!"
	})
	if f.templates.CommitURL != nil {
		termui.Handle("/sys/kbd/c", func(termui.Event) {
			file := ui.stackTrace[ui.widgets.stackTrace.CurrentItem]
			if file.CommitID != "" {
				ui.open(f.templates.CommitURL, file)
			}
		})
	}
	if f.templates.FileURL != nil {
		termui.Handle("/sys/kbd/f", func(termui.Event) {
			file := ui.stackTrace[ui.widgets.stackTrace.CurrentItem]
			if file.File != "" {
				ui.open(f.templates.FileURL, file)
			}
		})
	}
	if f.templates.BlameURL != nil {
		termui.Handle("/sys/kbd/b", func(termui.Event) {
			file := ui.stackTrace[ui.widgets.stackTrace.CurrentItem]
			if file.File != "" {
				ui.open(f.templates.BlameURL, file)
			}
		})
	}

	termui.Handle("/sys/wnd/resize", func(e termui.Event) {
		wnd := e.Data.(termui.EvtWnd)
		ui.SetHeight(wnd.Height)
		termui.Body.Align()
		ui.refresh()
	})
	termui.Handle("/timer/1s", func(termui.Event) {
		if ui.widgets.messages.Tick() {
			ui.refresh()
		}
	})

	// Layout
	termui.Body.AddRows(
		termui.NewRow(
			termui.NewCol(9, 0, ui.widgets.stackTrace),
			termui.NewCol(3, 0, ui.widgets.commit),
		// termui.NewCol(4, 0, ui.widgets.commits),
		),
		termui.NewRow(termui.NewCol(12, 0, ui.widgets.messages)),
	)

	termui.Body.Align()
	termui.Render(termui.Body)
	termui.Render(termui.Body)

	return nil
}

func (ui *UI) SetHeight(h int) {
	ui.widgets.commit.Height = h - 1
	ui.widgets.stackTrace.Height = h - 1
}

func (ui *UI) refresh() {
	termui.Render(termui.Body)
}

func (ui *UI) RenderDump(dump Dump) {
	ui.dump = &dump

	stack, files := ui.format.Stacktrace(dump)
	ui.stackTrace = files
	first := 0
	for first < len(ui.stackTrace) && ui.stackTrace[first].File == "" {
		first++
	}
	if first == len(ui.stackTrace) {
		first = 0
	}
	ui.widgets.stackTrace.SetItems(stack)
	ui.widgets.stackTrace.Select(first)
	ui.updateCommit()
	ui.refresh()
}

func (ui *UI) updateCommit() {
	id := ui.stackTrace[ui.widgets.stackTrace.CurrentItem].CommitID
	if id == "" {
		ui.widgets.commit.Text = ""
	} else {
		ui.widgets.commit.Text = ui.format.Commit(ui.dump.Commits.ByID[id])
	}
}

func (ui *UI) showMessage(status *string) {
	if status == nil {
		return
	}

	ui.messages.status.Content = *status
	ui.widgets.messages.AddMessage(ui.messages.status, widgets.Left)
	ui.refresh()
}

func (ui *UI) Loop() {
	termui.Loop()
}

func (ui *UI) Close() {
	termui.Close()
}

func (ui *UI) open(t *template.Template, file SourcePath) {
	url := bytes.Buffer{}
	err := t.Execute(&url, file)
	if err != nil {
		msg := err.Error()
		ui.showMessage(&msg)
		return
	}
	webbrowser.Open(url.String())
}
