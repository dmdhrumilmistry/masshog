package main

import (
	"bufio"
	"flag"
	"os"
	"strings"
	"sync"

	"github.com/dmdhrumilmistry/masshog/pkg/github"
	_ "github.com/dmdhrumilmistry/masshog/pkg/logging"
	"github.com/dmdhrumilmistry/masshog/pkg/trufflehog"
	"github.com/dmdhrumilmistry/masshog/pkg/utils"
	"github.com/rs/zerolog/log"
)

// ReadReposFromFile reads repository URLs from a file and returns a slice of Repo objects.
func ReadReposFromFile(filePath string) ([]github.Repo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var repos []github.Repo
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			parts := strings.Split(line, "/")
			size := len(parts)
			owner := parts[size-2]
			name := strings.Split(parts[size-1], ".git")[0]
			repos = append(repos, github.Repo{
				HttpsUrl: line,
				Owner:    owner,
				Name:     name,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return repos, nil
}

func main() {
	workers := flag.Int("w", 20, "number of concurrent workers")
	filePath := flag.String("f", "", "file path containing github repo https urls on each line")
	concurrency := flag.Int("c", 10, "trufflehog scan concurrency")
	onlyVerified := flag.Bool("ov", false, "only provides verified secrets in output")
	batchSize := flag.Int("bs", 100, "batch processing size")
	timeout := flag.Int("t", 60, "timeout for a trufflehog scan")

	username := flag.String("gu", "", "github username for scanning private repos")
	token := flag.String("gt", "", "github token for scanning private repos")

	outputFile := flag.String("o", "results.json", "file path for storing json result file")
	commitHashMapFile := flag.String("s", "state.json", "file path for loading and storing commit hash map state")

	flag.Parse()

	if *filePath == "" {
		log.Fatal().Msgf("file path is required. use -h flag for more info")
	}

	// check whether trufflehog is installed
	thPath := utils.IsTrufflehogInstalled()
	if thPath == "" {
		log.Fatal().Msgf("Trufflehog binary is required to be in path for this tool to run! Please install and retry")
	}

	// Read repos list from file
	repos, err := ReadReposFromFile(*filePath)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to read repos list from the file %s", *filePath)
	}
	log.Info().Msgf("%v", repos)

	// adjust batch size
	if *batchSize > len(repos) {
		*batchSize = len(repos)
	}

	log.Debug().Int("batch size", *batchSize).Msg("")

	// add jobs and init scan using workers
	th := trufflehog.NewTrufflehog(thPath, *workers, *batchSize, *concurrency, *timeout, *onlyVerified, *username, *token)

	// read commit hash map
	if err := utils.ReadSyncMapFromJsonFile(&th.CommitHashStateMap, *commitHashMapFile); err != nil {
		log.Error().Err(err).Msgf("Failed to load commit hash map from file: %s", *commitHashMapFile)
		th.CommitHashStateMap = sync.Map{}
	}

	th.AddJobs(repos)
	th.RunWorkers()

	log.Info().Msgf("%v", th)

	// dump commit hash map
	bson, err := utils.DumpSyncMapToJSON(&th.CommitHashStateMap)
	if err != nil {
		log.Error().Err(err).Msgf("failed to JSON marshal commit hash map")
	}

	if err := os.WriteFile(*commitHashMapFile, bson, 511); err != nil {
		log.Error().Err(err).Msgf("failed to write commit hash map to file: %s", *commitHashMapFile)
	}

	if err := utils.DumpJson(*outputFile, th); err != nil {
		log.Error().Err(err).Msgf("failed to store output file at path %s", *outputFile)
	}
}
