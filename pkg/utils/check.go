package utils

import (
	"os/exec"

	"github.com/rs/zerolog/log"
)

func IsTrufflehogInstalled() string {
	path, err := exec.LookPath("trufflehog")
	if err != nil {
		log.Error().Err(err).Msgf("error raised while trying to fetch trufflehog binary in path")
		return ""
	}
	log.Debug().Msgf("trufflehog binary found at path: %s", path)
	return path
}
