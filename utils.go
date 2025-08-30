package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

func getNetSpawnerDirPath() (string, error) {

	p, err := os.Executable()

	if err != nil {
		return "", err
	}

	return filepath.Dir(p), nil

}

func fileExists(p string) bool {

	fi, err := os.Stat(p)

	return err == nil && !fi.IsDir()

}

func dirExists(p string) bool {

	fi, err := os.Stat(p)

	return err == nil && fi.IsDir()

}

func copyFile(src, dst string) error {

	in, err := os.Open(src)

	if err != nil {
		return err
	}

	defer in.Close()

	if err := ensureDir(filepath.Dir(dst)); err != nil {
		return err
	}

	out, err := os.Create(dst)

	if err != nil {
		return err
	}

	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Sync()
}

// Function to iterate over chaindata directories and update the field FIRST_EPOCH_START_TIMESTAMP
// This field is a start timestamp of the first epoch and shows when blockchain was started

/*
For example, if you want to run a new network with 5 validators
this function will iterate over all 5 chaindata directories and modify the field in genesis.json
to the same new timestamp
*/
func updateGenesisTimestamp(genesisPath string, tsMs int64) error {
	b, err := os.ReadFile(genesisPath)
	if err != nil {
		return err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	m["FIRST_EPOCH_START_TIMESTAMP"] = tsMs
	nb, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(genesisPath, nb, 0o644)
}

func parseNodesCount(testnetDir string) (int, error) {

	// Expect format like "TESTNET_5V"
	parts := strings.Split(testnetDir, "_")

	if len(parts) < 2 {
		return 0, errors.New("cannot parse node count")
	}

	// Take the last part, e.g. "5V"
	last := parts[len(parts)-1]
	if !strings.HasSuffix(last, "V") {
		return 0, errors.New("invalid format, must end with 'V'")
	}

	numStr := strings.TrimSuffix(last, "V")
	n, err := strconv.Atoi(numStr)
	if err != nil || n <= 0 {
		return 0, errors.New("invalid node count in testnetDir")
	}
	return n, nil

}

func ensureDir(p string) error {
	return os.MkdirAll(p, 0o755)
}

// Function to read json config to run appropriate size network
func readConfig() (Config, string, error) {

	dir, err := getNetSpawnerDirPath()

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
func CreateDirsForNodes(cfg Config, baseDir string) []string {

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

func RunCoreProcess(pathToChainDir, corePath string) (*exec.Cmd, error) {

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
