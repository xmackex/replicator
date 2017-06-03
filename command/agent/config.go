package agent

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/elsevier-core-engineering/replicator/replicator/structs"
)

// Define default local addresses for Consul and Nomad
const (
	LocalConsulAddress = "localhost:8500"
	LocalNomadAddress  = "http://localhost:4646"
)

// DefaultConfig returns a default configuration struct with sane defaults.
func DefaultConfig() *structs.Config {

	return &structs.Config{
		Consul:            LocalConsulAddress,
		ConsulKeyLocation: "replicator/config",
		Nomad:             LocalNomadAddress,
		LogLevel:          "INFO",
		ScalingInterval:   10,

		ClusterScaling: &structs.ClusterScaling{
			MaxSize:            10,
			MinSize:            5,
			CoolDown:           600,
			NodeFaultTolerance: 1,
			RetryThreshold:     2,
		},

		JobScaling: &structs.JobScaling{},

		Telemetry:    &structs.Telemetry{},
		Notification: &structs.Notification{},
	}
}

// DevConfig returns a configuration struct with sane defaults for development
// and testing purposes.
func DevConfig() *structs.Config {

	return &structs.Config{
		Consul:            LocalConsulAddress,
		ConsulKeyLocation: "replicator/config",
		Nomad:             LocalNomadAddress,
		LogLevel:          "DEBUG",
		ScalingInterval:   10,

		ClusterScaling: &structs.ClusterScaling{
			Enabled:            false,
			MaxSize:            1,
			MinSize:            1,
			CoolDown:           0,
			NodeFaultTolerance: 0,
			RetryThreshold:     1,
		},

		JobScaling: &structs.JobScaling{},

		Telemetry:    &structs.Telemetry{},
		Notification: &structs.Notification{},
	}
}

// LoadConfig loads the configuration at the given path whether the specified
// path is an individual file or a directory of numerous configuration files.
func LoadConfig(path string) (*structs.Config, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if fi.IsDir() {
		return LoadConfigDir(path)
	}

	cleaned := filepath.Clean(path)
	config, err := ParseConfigFile(cleaned)
	if err != nil {
		return nil, fmt.Errorf("Error loading %s: %s", cleaned, err)
	}

	return config, nil
}

// LoadConfigDir loads all the configurations in the given directory
// in lexicographic order.
func LoadConfigDir(dir string) (*structs.Config, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf(
			"configuration path must be a directory: %s", dir)
	}

	var files []string
	err = nil
	for err != io.EOF {
		var fis []os.FileInfo
		fis, err = f.Readdir(128)
		if err != nil && err != io.EOF {
			return nil, err
		}

		for _, fi := range fis {

			// We do not wish tot navigate directories.
			if fi.IsDir() {
				continue
			}

			// Replicator can only parse HCL, and therefore json files, and so we
			// ignore all other file extensions.
			name := fi.Name()
			skip := true
			if strings.HasSuffix(name, ".hcl") {
				skip = false
			} else if strings.HasSuffix(name, ".json") {
				skip = false
			}
			if skip {
				continue
			}

			path := filepath.Join(dir, name)
			files = append(files, path)
		}
	}

	// If there are no files, there is no need to continue and therefore we exit
	// quickly.
	if len(files) == 0 {
		return &structs.Config{}, nil
	}

	sort.Strings(files)

	var result *structs.Config

	for _, f := range files {
		config, err := ParseConfigFile(f)
		if err != nil {
			return nil, fmt.Errorf("Error loading %s: %s", f, err)
		}

		if result == nil {
			result = config
		} else {
			result = result.Merge(config)
		}
	}

	return result, nil
}
