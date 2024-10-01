package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

func DumpJson(filePath string, data interface{}) error {
	// Create the directories if they do not exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directories: %v", err)
	}

	// Convert struct to JSON without indentation
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal struct to JSON: %v", err)
	}

	// Write JSON data to file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	_, err = file.Write(jsonData)
	if err != nil {
		return fmt.Errorf("failed to write data to file: %v", err)
	}

	return nil
}

// Dump sync.Map to JSON
func DumpSyncMapToJSON(m *sync.Map) ([]byte, error) {
	results := make(map[string]string)

	// Iterate over the sync.Map and populate the results map.
	m.Range(func(key, value interface{}) bool {
		results[key.(string)] = value.(string)
		return true
	})

	// Marshal the results map to JSON.
	jsonData, err := json.Marshal(results)
	if err != nil {
		return []byte{}, err
	}
	return jsonData, nil
}

// Read results from JSON file into sync.Map.
func ReadSyncMapFromJsonFile(m *sync.Map, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	var results map[string]string
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&results); err != nil {
		return err
	}

	// Load results into sync.Map
	for k, v := range results {
		m.Store(k, v)
	}

	return nil
}
