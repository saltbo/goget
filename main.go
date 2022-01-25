package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/go-resty/resty/v2"
	"github.com/manifoldco/promptui"
	"github.com/rivo/tview"
	"github.com/urfave/cli"
)

var (
	// RELEASE returns the release version
	release = "unknown"
	// REPO returns the git repository URL
	repo = "unknown"
	// COMMIT returns the short sha from git
	commit = "unknown"
)

type Package struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Intro string `json:"synopsis"`
}

func main() {
	app := cli.NewApp()
	app.Name = "goget"
	app.Usage = "Find the package you need easily"
	app.Copyright = "(c) 2019 saltbo.cn"
	app.Compiled = time.Now()
	app.Version = fmt.Sprintf("release: %s, repo: %s, commit: %s", release, repo, commit)
	app.Action = appAction
	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
	}
}

var app = tview.NewApplication()
var console = tview.NewTextView()

func setBoxAttr(box *tview.Box, title string) {
	box.SetTitle(fmt.Sprintf(" %s ", title))
	box.SetBackgroundColor(tcell.ColorDefault)
	box.SetBorder(true)
	box.SetBorderColor(tcell.ColorDefault)
	box.SetBackgroundColor(tcell.ColorDefault)
	box.SetBorderPadding(0, 0, 1, 1)
}

func appAction(c *cli.Context) error {
	console.SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).SetChangedFunc(func() {
		app.Draw()
	})
	setBoxAttr(console.Box, "Output")
	console.SetText("Ready.")

	searchInput := tview.NewInputField()
	setBoxAttr(searchInput.Box, "Search")
	searchInput.SetFieldBackgroundColor(tcell.ColorDefault)

	resultTable := tview.NewTable()
	setBoxAttr(resultTable.Box, "Results")
	resultTable.SetSelectable(true, false)

	var currentKeyword = c.Args().First()
	var appFocus tview.Primitive = searchInput
	if currentKeyword != "" {
		searchInput.SetText(currentKeyword)
		search(currentKeyword, resultTable)
		appFocus = resultTable
	}
	searchInput.SetDoneFunc(func(key tcell.Key) {
		if searchInput.GetText() == "" || searchInput.GetText() == currentKeyword {
			return
		}

		currentKeyword = searchInput.GetText()
		search(currentKeyword, resultTable)
	})

	leftLayout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(searchInput, 3, 1, false).
		AddItem(resultTable, 0, 3, false)

	rootLayout := tview.NewFlex().
		AddItem(leftLayout, 0, 1, false).
		AddItem(tview.NewBox().SetBackgroundColor(tcell.ColorDefault), 2, 1, false).
		AddItem(console, 0, 1, false)

	var currentTab = 0
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			tabs := []tview.Primitive{
				searchInput, resultTable,
			}
			idx := currentTab % len(tabs)
			app.SetFocus(tabs[idx])
			currentTab++
		}

		return event
	})

	return app.SetRoot(rootLayout, true).SetFocus(appFocus).Run()
}

func search(kw string, resultTable *tview.Table) {
	pkgs := pkgSearch(kw)

	resultTable.Clear()
	for idx, item := range pkgs {
		resultTable.SetCellSimple(idx, 0, item.Name)
		resultTable.SetCellSimple(idx, 1, item.Path)
	}
	resultTable.SetSelectedFunc(func(row, column int) {
		// enter to trigger go get
		modulePath := resultTable.GetCell(row, 1).Text
		console.Clear()
		console.Write([]byte(fmt.Sprintf("go get -v %s\n", modulePath)))
		go goGet(modulePath)
	})
	resultTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		//if event.Key() == tcell.KeyTab {
		//	app.SetFocus(searchInput)
		//}
		return event
	})

	resultTable.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorBlue))
	resultTable.Select(0, 0)
	resultTable.ScrollToBeginning()
	app.SetFocus(resultTable)
}

func pkgSearch(name string) []Package {
	respBody := map[string][]Package{}
	resp, err := resty.New().R().SetResult(&respBody).Get("https://api.godoc.org/search?q=" + name)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode() != 200 {
		log.Fatalf("status code error: %d %s", resp.StatusCode(), resp.Status())
	}

	return respBody["results"]
}

func openDoc(pkg string) error {
	prompt := promptui.Select{
		Label: fmt.Sprintf("Open the doc for %s", pkg),
		Items: []string{"No", "Yes"},
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }} ?",
			Active:   "\U0001F336 {{ . | cyan }}",
			Inactive: "  {{ . | cyan }}",
			Selected: "\U0001F336 Open the doc: {{ . | cyan }}",
		},
	}

	idx, _, err := prompt.Run()
	if err != nil {
		return err
	} else if idx == 0 {
		return nil
	}

	url := fmt.Sprintf("https://pkg.go.dev/%s", pkg)
	return exec.Command(`open`, url).Start()
}

func goGet(pkg string) error {
	cmd := exec.Command("go", "get", "-v", pkg)
	cmd.Stdout = console
	cmd.Stderr = console
	return cmd.Run()
}
