package agent

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	metrics "github.com/armon/go-metrics"
	"github.com/elsevier-core-engineering/replicator/command"
	"github.com/elsevier-core-engineering/replicator/logging"
	"github.com/elsevier-core-engineering/replicator/replicator"
	"github.com/elsevier-core-engineering/replicator/replicator/structs"
	"github.com/elsevier-core-engineering/replicator/version"
)

type Command struct {
	command.Meta
}

// Help provides the help information for the agent command.
func (c *Command) Help() string {
	helpText := `
  Usage: replicator agent [options]

    Starts the Replicator agent and runs until an interrupt is received.
    The Replicator agent's configuration primarily comes from the config
    files used. If no config file is passed, a default config will be
    used.

  General Options:

    -config=<path>
      The path to either a single config file or a directory of config
      files to use for configuring the Replicator agent. Replicator
      processes configuration files in lexicographic order.
`
	return strings.TrimSpace(helpText)
}

// Synopsis is provides a brief summary of the agent command.
func (c *Command) Synopsis() string {
	return "Runs a Replicator agent"
}

// Run triggers a run of the replicator agent by setting up and parsing the
// configuration and then initiating a new runner.
func (c *Command) Run(args []string) int {
	var config string

	flags := c.Meta.FlagSet("agent", command.FlagSetClient)
	flags.Usage = func() { c.UI.Output(c.Help()) }
	flags.StringVar(&config, "config", "", "")

	if err := flags.Parse(args); err != nil {
		return 1
	}

	conf, err := c.setupConfig(args)
	if err != nil {
		c.UI.Error(fmt.Sprintf("%v", err))
		return 1
	}

	// Set the logging level for the logger.
	logging.SetLevel(conf.LogLevel)

	// Initialize telemetry if this was configured by the user.
	if conf.Telemetry.StatsdAddress != "" {
		sink, statsErr := metrics.NewStatsdSink(conf.Telemetry.StatsdAddress)
		if statsErr != nil {
			c.UI.Error(fmt.Sprintf("unable to setup telemetry correctly: %v", statsErr))
			return 1
		}
		metrics.NewGlobal(metrics.DefaultConfig("replicator"), sink)
	}

	// Create the initial runner with the merged configuration parameters.
	runner, err := replicator.NewRunner(conf)
	if err != nil {
		return 1
	}

	logging.Debug("running version %v", version.Get())
	go runner.Start()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	for {
		select {
		case s := <-signalCh:
			switch s {
			case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				runner.Stop()
				return 0

			case syscall.SIGHUP:
				runner.Stop()

				// Reload the configuration in order to make proper use of SIGHUP.
				c, err := c.setupConfig(args)
				if err != nil {
					return 1
				}

				// Setup a new runner with the new configuration.
				runner, err = replicator.NewRunner(c)
				if err != nil {
					return 1
				}

				go runner.Start()
			}
		}
	}
}

// setupConfig asseses the CLI arguments, or lack of, and then iterates through
// the load order sementics for the configuration to return a configuration
// object.
func (c *Command) setupConfig(args []string) (*structs.Config, error) {

	// If the length of the CLI args is greater than one then there is an error.
	if len(args) > 1 {
		return nil, fmt.Errorf("too many command line args")
	}

	// If no cli flags are passed then we just return a default configuration
	// struct for use.
	if len(args) == 0 {
		return DefaultConfig(), nil
	}

	// If one CLI argument is passed this is split using the equals delimiter and
	// the right hand side used as the configuration file/path to parse.
	split := strings.Split(args[0], "=")

	switch p := split[0]; p {
	case "-config":
		c, err := FromPath(split[1])
		if err != nil {
			return nil, err
		}
		return c, nil
	default:
		return nil, fmt.Errorf("unable to correctly determine config location %v", split[1])
	}
}
