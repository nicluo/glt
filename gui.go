package main

import (
	gc "github.com/rthornton128/goncurses"
	"github.com/speedata/gogit"

	"fmt"
	"log"
	"strings"
	"time"
)

func selectCommit(stdscr *gc.Window, commits []*gogit.Commit) *gogit.Commit {
	stdscr.Clear()
	_, mx := stdscr.MaxYX()
	title := "Welcome to GLT!"
	stdscr.MovePrint(1, mx/2-len(title)/2, title)
	stdscr.MovePrint(16, 1, "'esc' to exit")
	stdscr.Keypad(true)

	win, err := gc.NewWindow(12, mx, 3, 0)
	if err != nil {
		log.Fatal(err)
	}
	win.Keypad(true)
	win.ColorOn(2)
	win.Box(0, 0)
	win.ColorOff(2)
	dwin := win.Derived(10, mx-2, 1, 1)

	// calculate remainder length for commit message
	messageLength := mx - 41

	items := make([]*gc.MenuItem, len(commits))
	for i, commit := range commits {
		label := " " + commit.Oid.String()[:16]

		// Get first line and trim description characters
		trimMessage := strings.Split(commit.CommitMessage, "\n")[0]
		if len(trimMessage) > messageLength {
			trimMessage = trimMessage[:messageLength-2] + ".."
		}
		desc := commit.Committer.When.String()[5:19] + " - " + trimMessage

		items[i], _ = gc.NewItem(label, desc)
		defer items[i].Free()
	}

	menu, err := gc.NewMenu(items)
	if err != nil {
		log.Fatal(err)
	}

	menu.SetPad('-')
	menu.SetSpacing(3, 1, 1)
	menu.SubWindow(dwin)
	menu.Post()
	defer menu.UnPost()
	defer menu.Free()

	stdscr.Refresh()
	win.Refresh()

	for {
		gc.Update()
		ch := win.GetChar()
		if ch == 27 {
			return nil
		}

		switch gc.KeyString(ch) {
		case "enter":
			index := menu.Current(nil).Index()
			return commits[index]
		case "down":
			menu.Driver(gc.REQ_DOWN)
		case "up":
			menu.Driver(gc.REQ_UP)
		}
	}
}

func editCommit(stdscr *gc.Window, commit *gogit.Commit) *gogit.Commit {
	stdscr.Clear()
	_, mx := stdscr.MaxYX()
	title := fmt.Sprintf("Edit Commit %s", commit.Oid.String())
	stdscr.MovePrint(1, mx/2-len(title)/2, title)
	stdscr.MovePrint(16, 1, "'enter' to save, 'esc' to exit")
	stdscr.Keypad(true)

	win, err := gc.NewWindow(12, mx, 3, 0)
	if err != nil {
		log.Fatal(err)
	}
	dwin := win.Derived(10, mx-2, 1, 1)
	win.Keypad(true)
	win.ColorOn(1)
	win.Box(0, 0)
	win.ColorOff(1)

	fields := make([]*gc.Field, 6)
	for i := 0; i < 6; i++ {
		fields[i], _ = gc.NewField(1, 30, int32(i), 19, 0, 0)
		defer fields[i].Free()
		fields[i].SetForeground(gc.ColorPair(3))
		fields[i].SetBackground(gc.ColorPair(3) | gc.A_UNDERLINE | gc.A_BOLD)
		fields[i].SetOptionsOff(gc.FO_AUTOSKIP)
	}

	fields[0].SetBuffer(commit.Author.Name)
	fields[1].SetBuffer(commit.Author.Email)
	fields[2].SetBuffer(commit.Author.When.String())
	fields[3].SetBuffer(commit.Committer.Name)
	fields[4].SetBuffer(commit.Committer.Email)
	fields[5].SetBuffer(commit.Committer.When.String())

	form, _ := gc.NewForm(fields)
	form.SetWindow(win)
	form.SetSub(dwin)
	form.Post()
	defer form.UnPost()
	defer form.Free()

	dwin.MovePrint(0, 1, "Author Name    :")
	dwin.MovePrint(1, 1, "Author Email   :")
	dwin.MovePrint(2, 1, "Author Date    :")
	dwin.MovePrint(3, 1, "Committer Name :")
	dwin.MovePrint(4, 1, "Committer Email:")
	dwin.MovePrint(5, 1, "Committer Date :")

	messageLength := mx - 4
	trimMessage := fmt.Sprintf("Message: %s", strings.Split(commit.CommitMessage, "\n")[0])
	if len(trimMessage) > messageLength {
		trimMessage = trimMessage[:messageLength-2] + ".."
	}
	dwin.MovePrint(7, 1, trimMessage)

	stdscr.Refresh()
	win.Refresh()

	form.Driver(gc.REQ_FIRST_FIELD)

	ch := win.GetChar()
	for ch != 27 {
		switch ch {
		case gc.KEY_ENTER, gc.KEY_RETURN:
			form.Driver(gc.REQ_VALIDATION)

			const sample = "2006-01-02 15:04:05 -0700 MST"
			authorTime, _ := time.Parse(sample, strings.TrimSpace(fields[2].Buffer()))
			committerTime, _ := time.Parse(sample, strings.TrimSpace(fields[5].Buffer()))

			commit.Author.Name = strings.TrimSpace(fields[0].Buffer())
			commit.Author.Email = strings.TrimSpace(fields[1].Buffer())
			commit.Author.When = authorTime
			commit.Committer.Name = strings.TrimSpace(fields[3].Buffer())
			commit.Committer.Email = strings.TrimSpace(fields[4].Buffer())
			commit.Committer.When = committerTime

			return commit
		case gc.KEY_LEFT:
			form.Driver(gc.REQ_PREV_CHAR)
		case gc.KEY_RIGHT:
			form.Driver(gc.REQ_NEXT_CHAR)
		case gc.KEY_DOWN, gc.KEY_TAB:
			form.Driver(gc.REQ_NEXT_FIELD)
		case gc.KEY_UP:
			form.Driver(gc.REQ_PREV_FIELD)
		case gc.KEY_BACKSPACE, 127:
			form.Driver(gc.REQ_DEL_PREV)
		case gc.KEY_DC:
			form.Driver(gc.REQ_DEL_CHAR)
		default:
			form.Driver(ch)
		}
		win.Refresh()
		ch = stdscr.GetChar()
	}

	return nil
}

func showResult(stdscr *gc.Window, result string) {
	y, x := 5, 20
	h, w := 10, 40

	var title string
	if result != "" {
		title = fmt.Sprintf("Changed: %s.", result)
	} else {
		title = "No Changes. Exiting."
	}
	exit := "Press any key to quit."
	window, _ := gc.NewWindow(h, w, y, x)
	window.Box(0, 0)
	window.MovePrint(1, (w/2)-(len(title)/2), title)
	window.MovePrint(2, (w/2)-(len(exit)/2), exit)
	gc.NewPanel(window)

	gc.UpdatePanels()
	gc.Update()

	stdscr.GetChar()
}
