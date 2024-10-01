package trufflehog

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/dmdhrumilmistry/masshog/pkg/github"
	"github.com/rs/zerolog/log"
)

type Trufflehog struct {
	Path         string `json:"path"`          // binary path
	Concurrency  int    `json:"concurrency"`   // trufflehog concurreny to be used for scanning
	Workers      int    `json:"workers"`       // golang concurrency workers
	OnlyVerified bool   `json:"only_verified"` // only verified flag from trufflehog
	Timeout      int    `json:"timeout"`       // kills trufflehog command execution after x seconds

	// Github authentication
	GithubToken    string `json:"-"` // github token for scanning private repos
	GithubUsername string `json:"-"` // github username for auth while scanning private repos

	// for storing commit hash states
	CommitHashStateMap sync.Map `json:"-"`

	// For Processing
	DataIgnored    []map[string]interface{} `json:"ignored_secrets"`
	DataVerified   []map[string]interface{} `json:"verified_secrets"`
	DataUnverified []map[string]interface{} `json:"unverified_secrets"`
	exceptions     []error                  `json:"-"`
	lock           sync.Mutex               `json:"-"`

	// Channels for worker pattern
	jobs    chan github.Repo `json:"-"`
	results chan error       `json:"-"`
}

func NewTrufflehog(path string, workers, batchSize, concurrency, timeout int, onlyVerified bool, githubUsername, githubToken string) *Trufflehog {
	th := &Trufflehog{
		Path:         path,
		Concurrency:  concurrency,
		OnlyVerified: onlyVerified,
		Workers:      workers,
		Timeout:      timeout,

		GithubUsername: githubUsername,
		GithubToken:    githubToken,
	}

	th.InitChannels(batchSize)
	return th
}

func (th *Trufflehog) InitChannels(bufferSize int) {
	th.jobs = make(chan github.Repo, bufferSize)
	th.results = make(chan error, bufferSize)
}

func (th *Trufflehog) CloseChannels() {
	close(th.results)
}

func (t *Trufflehog) ScanRepo(repo github.Repo) error {
	// Set a timeout duration for the command
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(t.Timeout)*time.Second)
	defer cancel() // Ensure that resources are cleaned up after the timeout

	cloneUrl := repo.HttpsUrl
	if t.GithubToken != "" && t.GithubUsername != "" {
		cloneUrl = repo.GetCloneUrl(t.GithubUsername, t.GithubToken)
	}

	commandArgs := []string{"git", cloneUrl, "--json", "--no-update"}

	// get latest commit hash
	if err := repo.GetCommitHash(t.GithubToken); err != nil {
		log.Error().Err(err).Msgf("failed to fetch latest commit hash for repo: %s", repo.HttpsUrl)
	}

	oldCommitHash, ok := t.CommitHashStateMap.Load(repo.HttpsUrl)
	if ok {
		if oldCommitHash == repo.CommitHash {
			log.Info().Msgf("Skipping scanning repo since there are no new commits after %s for repo %s", repo.CommitHash, repo.HttpsUrl)
			return nil
		} else {
			commandArgs = append(commandArgs, fmt.Sprintf("--since-commit=%s", oldCommitHash))
		}
	}

	if t.OnlyVerified {
		commandArgs = append(commandArgs, "--only-verified")
	}

	log.Info().Msgf("Scanning repo %s", repo.HttpsUrl)
	cmd := exec.CommandContext(ctx, t.Path, commandArgs...)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err := cmd.Run()

	// Get exit status
	exitStatus := 0

	// Handle different outcomes
	if ctx.Err() == context.DeadlineExceeded {
		log.Error().Err(ctx.Err()).Msgf("timeout reached while scanning repo %s", repo.HttpsUrl)
		return ctx.Err()
	} else if err != nil {
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

	// save latest commit hash to map after successful run
	t.CommitHashStateMap.Store(repo.HttpsUrl, repo.CommitHash)

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
