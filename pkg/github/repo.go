package github

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"
)

// Commit represents a commit structure from the GitHub API response
type Commit struct {
	SHA string `json:"sha"`
}

type Repo struct {
	Owner string
	Name  string

	HttpsUrl string

	CommitHash string
}

func (r *Repo) GetCloneUrl(username, token string) string {
	return fmt.Sprintf("https://%s:%s@github.com/%s/%s.git", username, token, r.Owner, r.Name)
}

func (r *Repo) GetCommitHash(token string) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits", r.Owner, r.Name)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Error().Err(err).Msgf("failed to generate request for getting %s repo commit hash", r.HttpsUrl)
		return err
	}

	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "masshog")
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error().Err(err).Msgf("failed to get response for %s repo commit hash", r.HttpsUrl)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotModified {
		var commits []Commit
		if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil {
			log.Error().Err(err).Msgf("Failed to decode response from %s", url)
			return err
		}
		if len(commits) > 0 {
			r.CommitHash = commits[0].SHA
			log.Info().Msgf("Fetched latest commit hash %s using commits url: %s", r.CommitHash, url)
		}
	}

	return nil
}
