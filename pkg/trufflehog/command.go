package trufflehog

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"syscall"

	"github.com/dmdhrumilmistry/masshog/pkg/github"
	"github.com/rs/zerolog/log"
)

type Trufflehog struct {
	Path         string // binary path
	Concurrency  int    // concurreny to be used for scanning
	OnlyVerified bool   // only verified flag from trufflehog

	// For Processing
	dataIgnored    []map[string]interface{}
	dataVerified   []map[string]interface{}
	dataUnverified []map[string]interface{}
	exceptions     []error
	lock           sync.Mutex
}

func NewTrufflehog(path string, concurrency int, onlyVerified bool) *Trufflehog {
	return &Trufflehog{
		Path:         path,
		Concurrency:  concurrency,
		OnlyVerified: onlyVerified,
	}
}

func (t *Trufflehog) ScanRepo(repo github.Repo) error {
	// TODO: mask https url with token
	maskedUrl := repo.HttpsUrl
	log.Info().Msgf("Scanning repo %s", maskedUrl)
	commandArgs := []string{"git", repo.HttpsUrl, "--json", "--no-update"}

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

	if err != nil {
		// If an error occurred, check if it's an exit error and get exit status
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				exitStatus = status.ExitStatus()
			}
		} else {
			return err
		}
	} else {
		// If no error, get the exit status from ProcessState
		if status, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
			exitStatus = status.ExitStatus()
		}
	}
	if exitStatus != 0 {
		return fmt.Errorf("trufflehog command returned exit code %d instead of 0", exitStatus)
	}

	// Process Result
	for lineCount, line := range strings.Split((stderr.String() + stdout.String()), "\n") {
		if !strings.Contains(line, "SourceMetadata") {
			continue
		}

		var jline map[string]interface{}

		if err := json.Unmarshal([]byte(line), &jline); err != nil {
			t.lock.Lock()
			t.exceptions = append(t.exceptions, err)
			t.lock.Unlock()
			log.Error().Err(err).Msgf("Error occurred while parsing line %d: %s", lineCount, line)
			continue
		}

		if strings.Contains(line, `"Verified":true`) {
			t.lock.Lock()
			if !containsRaw(t.dataVerified, jline["Raw"].(string)) {
				t.dataVerified = append(t.dataVerified, jline)
			}
			t.lock.Unlock()

		} else if strings.Contains(line, `"Verified":false`) {
			t.lock.Lock()
			if !contains(t.dataUnverified, jline) {
				t.dataUnverified = append(t.dataUnverified, jline)
			}
			t.lock.Unlock()
		}

	}
	return nil
}

func contains(slice []map[string]interface{}, item map[string]interface{}) bool {
	for _, elem := range slice {
		if fmt.Sprintf("%v", elem) == fmt.Sprintf("%v", item) {
			return true
		}
	}
	return false
}

func containsRaw(slice []map[string]interface{}, raw string) bool {
	for _, elem := range slice {
		if elem["Raw"] == raw {
			return true
		}
	}
	return false
}
