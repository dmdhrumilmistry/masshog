package trufflehog

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"

	"github.com/dmdhrumilmistry/masshog/pkg/github"
	"github.com/rs/zerolog/log"
)

var emptyString = ""

type RepoScanResult struct {
	Repo        *github.Repo
	CommandArgs []string
	Stdout      string
	StdErr      string
	ExitCode    int
}

type Trufflehog struct {
	Path         string // binary path
	Concurrency  int    // concurreny to be used for scanning
	OnlyVerified bool   // only verified flag from trufflehog
}

func NewTrufflehog(path string, concurrency int, onlyVerified bool) *Trufflehog {
	return &Trufflehog{
		Path:         path,
		Concurrency:  concurrency,
		OnlyVerified: onlyVerified,
	}
}

func (t *Trufflehog) ScanRepo(repo github.Repo) (RepoScanResult, error) {
	commandArgs := []string{"git", repo.HttpsUrl, "--json"}

	if t.OnlyVerified {
		commandArgs = append(commandArgs, "--only-verified")
	}

	if repo.CommitHash != "" {
		commandArgs = append(commandArgs, fmt.Sprintf("--since-commit=%s", repo.CommitHash))
	}

	cmd := exec.Command(t.Path, commandArgs...)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err := cmd.Run()

	// Get exit status
	exitStatus := 0
	scanResult := RepoScanResult{
		Repo:        &repo,
		CommandArgs: commandArgs,
		Stdout:      "",
		StdErr:      "",
		ExitCode:    0,
	}
	if err != nil {
		// If an error occurred, check if it's an exit error and get exit status
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				exitStatus = status.ExitStatus()
			}
		} else {
			return scanResult, err
		}
	} else {
		// If no error, get the exit status from ProcessState
		if status, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
			exitStatus = status.ExitStatus()
		}
	}

	// Return stdout, stderr, exit status, and error
	scanResult.ExitCode = exitStatus
	scanResult.Stdout = stdout.String()
	scanResult.StdErr = stderr.String()

	log.Info().Msgf("%v", scanResult)
	return scanResult, nil
}
