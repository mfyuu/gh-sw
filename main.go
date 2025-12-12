package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
)

const (
	fetchTitle     = "Fetching local branches..."
	defaultTimeout = 5 * time.Second
	helpText       = `Interactively switch to a local branch.

USAGE
  gh sw [branch]
  gh sw [flags]

FLAGS
  --help    Show help for command

EXAMPLES
  $ gh sw              # Interactive branch selection
  $ gh sw feature/auth # Switch to specific branch
  $ gh sw -            # Switch to previous branch
`
)

var grayStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	args := os.Args[1:]
	if len(args) > 0 {
		if args[0] == "--help" || args[0] == "-h" {
			fmt.Print(helpText)
			return
		}
		if err := switchBranch(ctx, args[0]); err != nil {
			exitWithStatus(err)
		}
		return
	}

	interactiveSwitch(ctx)
}

func getBranches(ctx context.Context) (branches []string, current string, err error) {
	cmd := exec.CommandContext(ctx, "git", "for-each-ref", "--format=%(refname:short)\t%(HEAD)", "refs/heads")
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Stderr.Write(exitErr.Stderr)
		}
		return nil, "", err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Split(line, "\t")
		branch := fields[0]
		branches = append(branches, branch)
		if len(fields) > 1 && fields[1] == "*" {
			current = branch
		}
	}

	if current == "" {
		return nil, "", fmt.Errorf("failed to detect current branch")
	}

	slices.Sort(branches)

	return branches, current, nil
}

func interactiveSwitch(ctx context.Context) {
	branches, current, err := fetchBranches(ctx)

	if err != nil {
		exitWithStatus(err)
	}

	if len(branches) == 0 {
		return
	}

	// Build options excluding the current branch
	var options []huh.Option[string]

	for _, branch := range branches {
		if branch != current {
			options = append(options, huh.NewOption(branch, branch))
		}
	}

	// No other branches to switch to
	if len(options) == 0 {
		fmt.Fprintln(os.Stderr, grayStyle.Render(fmt.Sprintf("Only one local branch exists: '%s'.", current)))
		return
	}

	var selected string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a branch to switch to:").
				Description(grayStyle.Render("  * " + current)).
				Options(options...).
				Value(&selected),
		),
	)

	err = form.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, grayStyle.Render(fmt.Sprintf("Operation cancelled, staying on '%s'.", current)))
		return
	}

	if err := switchBranch(ctx, selected); err != nil {
		exitWithStatus(err)
	}
}

func fetchBranches(ctx context.Context) ([]string, string, error) {
	var branches []string
	var current string
	var fetchErr error

	_ = spinner.New().
		Title(fetchTitle).
		Action(func() {
			branches, current, fetchErr = getBranches(ctx)
		}).
		Run()

	return branches, current, fetchErr
}

func switchBranch(ctx context.Context, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "switch", branch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func exitWithStatus(err error) {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		os.Exit(exitErr.ExitCode())
	}

	// Print message only for non-ExitError
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
