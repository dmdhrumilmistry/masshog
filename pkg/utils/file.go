package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
