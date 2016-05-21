package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	gc "github.com/rthornton128/goncurses"
	"github.com/speedata/gogit"
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	app := cli.NewApp()
	app.Name = "glt"
	app.Usage = "Git Local Transform"
	app.Version = "1.0.0"
	app.Action = func(c *cli.Context) {
		repo, err := OpenCurrentRepository()
		if err != nil {
			log.Fatalf("error opening repository: %v", err)
		}

		commits, err := repo.GetLog(10)
		if err != nil {
			log.Fatalf("error getting commit log: %v", err)
		}

		// Initialize file logging just before curses
		f, err := os.OpenFile("glt.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("error opening log file: %v", err)
		}
		defer f.Close()
		log.SetOutput(f)

		stdscr, err := gc.Init()
		if err != nil {
			log.Fatal("init:", err)
		}
		defer gc.End()
		gc.Raw(true)
		gc.CBreak(true)
		gc.Echo(false)
		gc.StartColor()
		gc.Cursor(1)

		// gui()
		commitIndex := selectCommit(stdscr, commits)
		if commitIndex == -1 {
			return
		}
		commit := commits[commitIndex]
		log.Println("Entering Edit")
		LogCommit(commit)
		commit = editCommit(stdscr, commit)
		log.Println("After Edit")
		LogCommit(commit)

		if commit != nil {
			change, err := repo.SaveCommitIfModified(commit)
			if err != nil {
				log.Fatalf("Error saving commit: %s", err)
			}
			if change != "" {
				log.Printf("Successfully saved: %s", change)
			}
			showResult(stdscr, change)
		}
	}
	app.Run(os.Args)
}

func selectCommit(stdscr *gc.Window, commits []*gogit.Commit) int {
	menu_items := make([]string, len(commits))
	for i, commit := range commits {
		menu_items[i] = commit.Oid.String()
	}

	stdscr.Clear()
	stdscr.Keypad(true)

	items := make([]*gc.MenuItem, len(menu_items))
	for i, val := range menu_items {
		items[i], _ = gc.NewItem(val, "")
		defer items[i].Free()
	}

	menu, err := gc.NewMenu(items)
	if err != nil {
		log.Fatal(err)
	}
	defer menu.Free()

	menu.Post()

	stdscr.MovePrint(20, 0, "'esc' to exit")
	stdscr.Refresh()

	for {
		gc.Update()
		ch := stdscr.GetChar()
		if ch == 27 {
			return -1
		}

		switch gc.KeyString(ch) {
		case "enter":
			return menu.Current(nil).Index()
		case "down":
			menu.Driver(gc.REQ_DOWN)
		case "up":
			menu.Driver(gc.REQ_UP)
		}
	}
}

func showResult(stdscr *gc.Window, result string) {
	y, x := 2, 4
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

func editCommit(stdscr *gc.Window, commit *gogit.Commit) *gogit.Commit {
	stdscr.Clear()
	stdscr.Keypad(true)

	gc.InitPair(1, gc.C_WHITE, gc.C_BLUE)
	gc.InitPair(2, gc.C_YELLOW, gc.C_BLUE)
	gc.InitPair(3, gc.C_WHITE, gc.C_GREEN)

	fields := make([]*gc.Field, 6)
	for i := 0; i < 6; i++ {
		fields[i], _ = gc.NewField(1, 30, int32(i), 17, 0, 0)
		defer fields[i].Free()
		fields[i].SetForeground(gc.ColorPair(1))
		fields[i].SetBackground(gc.ColorPair(2) | gc.A_UNDERLINE | gc.A_BOLD)
		fields[i].SetOptionsOff(gc.FO_AUTOSKIP)
	}

	form, _ := gc.NewForm(fields)
	form.Post()
	defer form.UnPost()
	defer form.Free()
	stdscr.Refresh()

	fields[0].SetBuffer(commit.Author.Name)
	fields[1].SetBuffer(commit.Author.Email)
	fields[2].SetBuffer(commit.Author.When.String())
	fields[3].SetBuffer(commit.Committer.Name)
	fields[4].SetBuffer(commit.Committer.Email)
	fields[5].SetBuffer(commit.Committer.When.String())

	stdscr.AttrOn(gc.ColorPair(2) | gc.A_BOLD)
	stdscr.MovePrint(0, 0, "Author Name    :")
	stdscr.MovePrint(1, 0, "Author Email   :")
	stdscr.MovePrint(2, 0, "Author Date    :")
	stdscr.MovePrint(3, 0, "Committer Name :")
	stdscr.MovePrint(4, 0, "Committer Email:")
	stdscr.MovePrint(5, 0, "Committer Date :")
	stdscr.AttrOff(gc.ColorPair(2) | gc.A_BOLD)
	stdscr.Refresh()

	form.Driver(gc.REQ_FIRST_FIELD)

	ch := stdscr.GetChar()
	for ch != 27 {
		switch ch {
		case gc.KEY_ENTER, gc.KEY_RETURN:
			log.Println(fields[2].Buffer())
			const sample = "2006-01-02 15:04:05 -0700 MST"
			authorTime, _ := time.Parse(sample, strings.TrimSpace(fields[2].Buffer()))
			committerTime, _ := time.Parse(sample, strings.TrimSpace(fields[5].Buffer()))

			log.Println(authorTime)
			log.Println(committerTime)

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
		ch = stdscr.GetChar()
	}

	return nil
}
