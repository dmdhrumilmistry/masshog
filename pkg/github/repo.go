package github

import "fmt"

type Repo struct {
	Owner string
	Name  string

	HttpsUrl string

	CommitHash string
}

func (r *Repo) GetCloneUrl(username, token string) string {
	return fmt.Sprintf("https://%s:%s@github.com/%s/%s.git", username, token, r.Owner, r.Name)
}
