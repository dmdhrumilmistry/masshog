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
	Concurrency  int    // trufflehog concurreny to be used for scanning
	Workers      int    // golang concurrency workers
	OnlyVerified bool   // only verified flag from trufflehog

	// For Processing
	DataIgnored    []map[string]interface{}
	DataVerified   []map[string]interface{}
	DataUnverified []map[string]interface{}
	exceptions     []error
	lock           sync.Mutex

	// Channels for worker pattern
	jobs    chan github.Repo
	results chan error
}

func NewTrufflehog(path string, concurrency, workers int, onlyVerified bool) *Trufflehog {
	return &Trufflehog{
		Path:         path,
		Concurrency:  concurrency,
		OnlyVerified: onlyVerified,
		Workers:      workers,
	}
}

func (th *Trufflehog) InitChannels(bufferSize int) {
	th.jobs = make(chan github.Repo, bufferSize)
	th.results = make(chan error, bufferSize)
}

func (th *Trufflehog) CloseChannels() {
	close(th.results)
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
			if !containsRaw(t.DataVerified, jline["Raw"].(string)) {
				t.DataVerified = append(t.DataVerified, jline)
			}
			t.lock.Unlock()

		} else if strings.Contains(line, `"Verified":false`) {
			t.lock.Lock()
			if !contains(t.DataUnverified, jline) {
				t.DataUnverified = append(t.DataUnverified, jline)
			}
			t.lock.Unlock()
		}

	}
	return nil
}

// AddJobs sends repositories to the jobs channel.
func (th *Trufflehog) AddJobs(repos []github.Repo) {
	for _, repo := range repos {
		th.jobs <- repo
	}
	close(th.jobs) // Close the jobs channel after adding all jobs.
}

func (th *Trufflehog) RunWorkers() {
	var wg sync.WaitGroup

	for i := 0; i < th.Workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for repo := range th.jobs {
				log.Info().Msgf("Worker %d scanning %s\n", id, repo.HttpsUrl)
				err := th.ScanRepo(repo)
				th.lock.Lock()
				if err != nil {
					th.exceptions = append(th.exceptions, err)
					log.Error().Err(err).Msgf("Worker %d: failed to scan repo: %s", id, repo.HttpsUrl)
				}
				th.lock.Unlock()
				th.results <- err
			}
		}(i)
	}

	wg.Wait()
	th.CloseChannels()
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
