package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
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

	case "TESTNET_5V":
		dirs = nil
		for i := 1; i <= 5; i++ {
			dirs = append(dirs, filepath.Join(baseDir, "V"+strconv.Itoa(i)))
		}
	case "TESTNET_21V":
		dirs = nil
		for i := 1; i <= 21; i++ {
			dirs = append(dirs, filepath.Join(baseDir, "V"+strconv.Itoa(i)))
		}
	}

	return dirs
}

func pipeWithPrefix(r io.Reader, prefix string, dst io.Writer, wg *sync.WaitGroup) {

	wg.Add(1)
	go func() {
		defer wg.Done()
		sc := bufio.NewScanner(r)
		buf := make([]byte, 0, 1024*64)
		sc.Buffer(buf, 1024*1024)
		for sc.Scan() {
			fmt.Fprintf(dst, "[%s]: %s\n", prefix, sc.Text())
		}
	}()
}

type ioWaitGroup struct {
	wg     sync.WaitGroup
	Prefix string
}

func (i *ioWaitGroup) Attach(stdout, stderr io.Reader) {
	pipeWithPrefix(stdout, i.Prefix, os.Stdout, &i.wg)
	pipeWithPrefix(stderr, i.Prefix, os.Stderr, &i.wg)
}
func (i *ioWaitGroup) Wait() { i.wg.Wait() }

func runCoreProcess(pathToChainDir, corePath string) (*exec.Cmd, error) {

	// Invoke the Go node binary directly
	cmd := exec.Command(corePath)

	// Inherit environment and provide per-node overrides
	cmd.Env = append(os.Environ(), "CHAINDATA_PATH="+pathToChainDir)

	stdout, err := cmd.StdoutPipe()

	if err != nil {
		return nil, err
	}

	stderr, err := cmd.StderrPipe()

	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	var wg ioWaitGroup

	wg.Prefix = pathToChainDir
	wg.Attach(stdout, stderr)

	go func() {
		_ = cmd.Wait()
		wg.Wait()
		fmt.Fprintf(os.Stdout, "[%s]: process exited\n", pathToChainDir)
	}()

	return cmd, nil
}
