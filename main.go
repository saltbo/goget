package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/manifoldco/promptui"
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
	Name  string
	Intro string
}

func main() {
	app := cli.NewApp()
	app.Name = "gomods"
	app.Usage = "Find the package you need easily"
	app.Copyright = "(c) 2019 saltbo.cn"
	app.Compiled = time.Now()
	app.Version = fmt.Sprintf("release: %s, repo: %s, commit: %s", release, repo, commit)
	app.Action = appAction
	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
	}
}

func appAction(c *cli.Context) error {
	args := c.Args()
	if len(args) == 0 {
		return fmt.Errorf("input the keyword that you want serach.")
	}

	items := pkgSearch(args.First())
	prompt := promptui.Select{
		Label: "Select the packages",
		Items: items,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}?",
			Active:   "\U0001F336 {{ .Name | cyan }} ({{ .Intro | red }})",
			Inactive: "  {{ .Name | cyan }} ({{ .Intro | red }})",
			Selected: "\U0001F336 {{ .Name | cyan }}",
		},
	}
	idx, _, err := prompt.Run()
	if err != nil {
		return fmt.Errorf("Prompt failed %v\n", err)
	}

	pkg := items[idx].Name
	if err := openDoc(pkg); err != nil {
		return fmt.Errorf("open doc failed: %s", err)
	}

	if err := goGet(pkg); err != nil {
		return fmt.Errorf("go get failed %v\n", err)
	}

	return nil
}

func pkgSearch(name string) []Package {
	// Request the HTML page.
	res, err := http.Get("https://pkg.go.dev/search?q=" + name)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	// Find the packages
	items := make([]Package, 0)
	doc.Find(".SearchSnippet").Each(func(i int, s *goquery.Selection) {
		pkg := s.Find("a").Text()
		intro := s.Find("p").Text()
		if intro == "" {
			intro = "-"
		}

		items = append(items, Package{Name: pkg, Intro: intro})
	})
	return items
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
	prompt := promptui.Prompt{
		Label:     fmt.Sprintf("go get %s", pkg),
		IsConfirm: true,
		Default:   "y",
		Templates: &promptui.PromptTemplates{
			Confirm: "\U0001F336 {{ . | cyan }}?",
			Success: "\U0001F336 {{ . | cyan }}",
		},
	}
	if _, err := prompt.Run(); err != nil {
		return nil
	}

	cmd := exec.Command("go", "get", "-v", pkg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
