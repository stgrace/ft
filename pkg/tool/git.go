package tool

import (
	"fmt"
	"strings"

	"github.com/helm/chart-testing/v3/pkg/exec"
)

type Git struct {
	exec exec.ProcessExecutor
}

func NewGit(exec exec.ProcessExecutor) Git {
	return Git{
		exec: exec,
	}
}

func (g Git) FileExistsOnBranch(file string, remote string, branch string) bool {
	fileSpec := fmt.Sprintf("%s/%s:%s", remote, branch, file)
	_, err := g.exec.RunProcessAndCaptureOutput("git", "cat-file", "-e", fileSpec)
	return err == nil
}

func (g Git) AddWorktree(path string, ref string) error {
	return g.exec.RunProcess("git", "worktree", "add", path, ref)
}

func (g Git) RemoveWorktree(path string) error {
	return g.exec.RunProcess("git", "worktree", "remove", path)
}

func (g Git) Show(file string, remote string, branch string) (string, error) {
	fileSpec := fmt.Sprintf("%s/%s:%s", remote, branch, file)
	return g.exec.RunProcessAndCaptureOutput("git", "show", fileSpec)
}

func (g Git) MergeBase(commit1 string, commit2 string) (string, error) {
	return g.exec.RunProcessAndCaptureOutput("git", "merge-base", commit1, commit2)
}

func (g Git) ListChangedFiles(commit string) ([]string, error) {
	changedChartFilesString, err :=
		g.exec.RunProcessAndCaptureOutput("git", "diff", "--find-renames", "--name-only", commit)
	if err != nil {
		return nil, fmt.Errorf("failed creating diff: %w", err)
	}
	if changedChartFilesString == "" {
		return nil, nil
	}
	return strings.Split(changedChartFilesString, "\n"), nil
}

func (g Git) GetURLForRemote(remote string) (string, error) {
	return g.exec.RunProcessAndCaptureOutput("git", "ls-remote", "--get-url", remote)
}

func (g Git) ValidateRepository() error {
	_, err := g.exec.RunProcessAndCaptureOutput("git", "rev-parse", "--is-inside-work-tree")
	return err
}
