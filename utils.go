package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// Function to check if the dir is exists
func exeDir() (string, error) {

	p, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(p), nil
}

// Function to read json config to run appropriate size network
func readConfig() (Config, string, error) {

	dir, err := exeDir()
	if err != nil {
		return Config{}, "", err
	}
	cfgPath := filepath.Join(dir, "configs.json")
	b, err := os.ReadFile(cfgPath)
	if err != nil {
		return Config{}, "", fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return Config{}, "", fmt.Errorf("parse config: %w", err)
	}
	return cfg, dir, nil
}

// Function to create directories with chaindata for each node
func createDirsForNodes(cfg Config, baseDir string) []string {

	// Default to V1 and V2
	dirs := []string{filepath.Join(baseDir, "V1"), filepath.Join(baseDir, "V2")}

	switch cfg.NetMode {

	case "TESTNET_V5":
		dirs = nil
		for i := 1; i <= 5; i++ {
			dirs = append(dirs, filepath.Join(baseDir, "V"+strconv.Itoa(i)))
		}
	case "TESTNET_V21":
		dirs = nil
		for i := 1; i <= 21; i++ {
			dirs = append(dirs, filepath.Join(baseDir, "V"+strconv.Itoa(i)))
		}
	}

	return dirs
}
