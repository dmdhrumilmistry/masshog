package main

import (
	"flag"
	"strings"

	"github.com/dmdhrumilmistry/masshog/pkg/github"
	_ "github.com/dmdhrumilmistry/masshog/pkg/logging"
	"github.com/dmdhrumilmistry/masshog/pkg/trufflehog"
	"github.com/dmdhrumilmistry/masshog/pkg/utils"
	"github.com/rs/zerolog/log"
)

func main() {
	workers := flag.Int("w", 20, "number of concurrent workers")
	concurrency := flag.Int("c", 10, "trufflehog scan concurrency")
	onlyVerified := flag.Bool("ov", true, "only provides verified secrets in output")
	flag.Parse()

	thPath := utils.IsTrufflehogInstalled()
	if thPath == "" {
		log.Fatal().Msgf("Trufflehog binary is required to be in path for this tool to run! Please install and retry")
	}

	th := trufflehog.NewTrufflehog(thPath, *concurrency, *onlyVerified)
	repo := github.Repo{
		HttpsUrl: "https://github.com/OWASP/OFFAT.git",
	}
	result, err := th.ScanRepo(repo)

	if err != nil {
		log.Fatal().Err(err).Msgf("failed to scan repo: %v", repo)
	}

	log.Info().Msgf("%v", strings.Split(result.Stdout, "\n"))

	log.Info().Msgf("%v", th)
	log.Info().Int("workers", *workers).Msg("")
}
