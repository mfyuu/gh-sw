package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
)

func main() {
	args := os.Args[1:]
	if len(args) > 0 {
		switchBranch(args[0])
	} else {
		interactiveSwitch()
	}
}

func getBranches() (branches []string, current string, err error) {
	// 現在のブランチを取得
	currentCmd := exec.Command("git", "branch", "--show-current")
	currentOut, err := currentCmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Stderr.Write(exitErr.Stderr)
		}
		return nil, "", err
	}
	current = strings.TrimSpace(string(currentOut))

	// ブランチ一覧を取得
	listCmd := exec.Command("git", "branch", "--format=%(refname:short)")
	listOut, err := listCmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Stderr.Write(exitErr.Stderr)
		}
		return nil, "", err
	}

	lines := strings.Split(strings.TrimSpace(string(listOut)), "\n")
	for _, line := range lines {
		if line != "" {
			branches = append(branches, line)
		}
	}

	// アルファベット順にソート
	slices.Sort(branches)

	return branches, current, nil
}

func interactiveSwitch() {
	var branches []string
	var current string
	var err error

	_ = spinner.New().
		Title("Fetching local branches...").
		Action(func() {
			branches, current, err = getBranches()
		}).
		Run()

	if err != nil {
		os.Exit(1)
	}

	if len(branches) == 0 {
		return
	}

	// 選択肢を構築（現在のブランチ以外）
	currentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")) // Gold
	var options []huh.Option[string]

	for _, branch := range branches {
		if branch != current {
			options = append(options, huh.NewOption(branch, branch))
		}
	}

	// 他に切り替え可能なブランチがない場合
	if len(options) == 0 {
		grayStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		fmt.Fprintln(os.Stderr, grayStyle.Render(fmt.Sprintf("Only one local branch exists: '%s'.", current)))
		return
	}

	var selected string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(currentStyle.Render("  * " + current)).
				Options(options...).
				Value(&selected),
		),
	)

	err = form.Run()
	if err != nil {
		grayStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		fmt.Fprintln(os.Stderr, grayStyle.Render(fmt.Sprintf("Operation cancelled, staying on '%s'.", current)))
		return
	}

	switchBranch(selected)
}

func switchBranch(branch string) {
	cmd := exec.Command("git", "switch", branch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}
