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
	defaultTimeout = 5 * time.Second
	helpText       = `Interactively switch to a local branch.

USAGE
  gh sw [branch]
  gh sw [flags]

FLAGS
  -a, --all       Select from all branches (local + remote)
  -r, --remote    Select from remote branches (+ current branch)
  --help          Show help for command

EXAMPLES
  $ gh sw              # Interactive branch selection
  $ gh sw feature/auth # Switch to specific branch
  $ gh sw -            # Switch to previous branch
  $ gh sw -a           # Select from all branches
  $ gh sw -r           # Select from remote branches
`
)

var grayStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "--help", "-h":
			fmt.Print(helpText)
			return
		case "--all", "-a":
			interactiveSwitchAll(ctx)
			return
		case "--remote", "-r":
			interactiveSwitchRemote(ctx)
			return
		}
		if err := switchBranch(args[0]); err != nil {
			exitWithStatus(err)
		}
		return
	}

	interactiveSwitchLocal(ctx)
}

func getCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func getLocalBranches(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "for-each-ref", "--format=%(refname:short)", "refs/heads")
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Stderr.Write(exitErr.Stderr)
		}
		return nil, err
	}

	var branches []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		branches = append(branches, line)
	}

	slices.Sort(branches)

	return branches, nil
}

func getRemoteBranches(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "for-each-ref", "--format=%(refname:short)", "refs/remotes")
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Stderr.Write(exitErr.Stderr)
		}
		return nil, err
	}

	var branches []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		// Skip entries without '/' (e.g., "origin" from symbolic refs)
		if !strings.Contains(line, "/") {
			continue
		}
		// Skip HEAD references like "origin/HEAD"
		if strings.HasSuffix(line, "/HEAD") {
			continue
		}
		branches = append(branches, line)
	}

	slices.Sort(branches)

	return branches, nil
}

func interactiveSwitchLocal(ctx context.Context) {
	branches, err := fetchLocalBranches(ctx)

	if err != nil {
		exitWithStatus(err)
	}

	if len(branches) == 0 {
		fmt.Fprintln(os.Stderr, grayStyle.Render("No local branches found."))
		return
	}

	current, _ := getCurrentBranch()

	var options []huh.Option[string]
	// Add current branch first with * prefix and gray style
	if current != "" {
		options = append(options, huh.NewOption(grayStyle.Render("* "+current), current))
	}
	// Add other branches
	for _, branch := range branches {
		if branch != current {
			options = append(options, huh.NewOption(branch, branch))
		}
	}

	var selected string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a branch to switch to:").
				Options(options...).
				Value(&selected),
		),
	)

	err = form.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, grayStyle.Render("Operation cancelled."))
		return
	}

	if err := switchBranch(selected); err != nil {
		exitWithStatus(err)
	}
}

func interactiveSwitchRemote(ctx context.Context) {
	branches, err := fetchRemoteBranches(ctx)

	if err != nil {
		exitWithStatus(err)
	}

	if len(branches) == 0 {
		fmt.Fprintln(os.Stderr, grayStyle.Render("No remote branches found."))
		return
	}

	current, _ := getCurrentBranch()

	var options []huh.Option[string]
	// Add current local branch first with * prefix and gray style
	if current != "" {
		options = append(options, huh.NewOption(grayStyle.Render("* "+current), current))
	}
	// Add remote branches
	for _, branch := range branches {
		options = append(options, huh.NewOption(branch, branch))
	}

	var selected string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a remote branch to switch to:").
				Options(options...).
				Value(&selected),
		),
	)

	err = form.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, grayStyle.Render("Operation cancelled."))
		return
	}

	// Strip remote prefix: origin/main -> main, origin/feature/auth -> feature/auth
	if idx := strings.Index(selected, "/"); idx != -1 {
		selected = selected[idx+1:]
	}

	if err := switchBranch(selected); err != nil {
		exitWithStatus(err)
	}
}

func interactiveSwitchAll(ctx context.Context) {
	localBranches, remoteBranches, err := fetchAllBranches(ctx)

	if err != nil {
		exitWithStatus(err)
	}

	if len(localBranches) == 0 && len(remoteBranches) == 0 {
		fmt.Fprintln(os.Stderr, grayStyle.Render("No branches found."))
		return
	}

	current, _ := getCurrentBranch()

	var options []huh.Option[string]
	// Add current branch first with * prefix and gray style
	if current != "" {
		options = append(options, huh.NewOption(grayStyle.Render("* "+current), current))
	}
	// Add local branches
	for _, branch := range localBranches {
		if branch != current {
			options = append(options, huh.NewOption(branch, branch))
		}
	}
	// Add remote branches
	for _, branch := range remoteBranches {
		options = append(options, huh.NewOption(branch, branch))
	}

	var selected string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a branch to switch to:").
				Options(options...).
				Value(&selected),
		),
	)

	err = form.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, grayStyle.Render("Operation cancelled."))
		return
	}

	// Strip remote prefix if remote branch selected: origin/main -> main
	if strings.Contains(selected, "/") {
		if idx := strings.Index(selected, "/"); idx != -1 {
			selected = selected[idx+1:]
		}
	}

	if err := switchBranch(selected); err != nil {
		exitWithStatus(err)
	}
}

func fetchLocalBranches(ctx context.Context) ([]string, error) {
	var branches []string
	var fetchErr error

	_ = spinner.New().
		Title("Fetching local branches...").
		Action(func() {
			branches, fetchErr = getLocalBranches(ctx)
		}).
		Run()

	return branches, fetchErr
}

func fetchRemoteBranches(ctx context.Context) ([]string, error) {
	var branches []string
	var fetchErr error

	_ = spinner.New().
		Title("Fetching remote branches...").
		Action(func() {
			branches, fetchErr = getRemoteBranches(ctx)
		}).
		Run()

	return branches, fetchErr
}

func fetchAllBranches(ctx context.Context) ([]string, []string, error) {
	var localBranches, remoteBranches []string
	var fetchErr error

	_ = spinner.New().
		Title("Fetching branches...").
		Action(func() {
			localBranches, fetchErr = getLocalBranches(ctx)
			if fetchErr != nil {
				return
			}
			remoteBranches, fetchErr = getRemoteBranches(ctx)
		}).
		Run()

	return localBranches, remoteBranches, fetchErr
}

func switchBranch(branch string) error {
	cmd := exec.Command("git", "switch", branch)
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
