package command

import (
	"fmt"
	"strings"

	"github.com/elsevier-core-engineering/replicator/command/base"
	core "github.com/elsevier-core-engineering/replicator/replicator"
	"github.com/elsevier-core-engineering/replicator/replicator/structs"
)

// FailsafeCommand is a command implementation that allows operators to
// place the daemon in or take the daemon out of failsafe mode.
type FailsafeCommand struct {
	Meta
	args []string
}

// Help provides the help information for the failsafe command.
func (c *FailsafeCommand) Help() string {
	helpText := `
Usage: replicator failsafe [options]

  Allows an operator to administratively control the failsafe behavior
  of Replicator. When Replicator enters failsafe mode, all running
  copies of Replicator will prohibit any scaling operations.

  Failsafe mode is intended to stabilize a cluster that has experienced
  consecutive critical failures while attempting to perform scaling
  operations.

  To exit failsafe mode, an operator must explicitly remove the failsafe
  lock after identifying the root cause of the failures.

  General Options:

    -config=<path>
      The path to either a single config file or a directory of config
      files to use when configuring the Replicator agent. Replicator
      processes configuration files in lexicographic order.

    -consul=<address:port>
      This is the address of the Consul agent. By default, this is
      localhost:8500, which is the default bind and port for a local
      Consul agent. It is not recommended that you communicate directly
      with a Consul server, and instead communicate with the local
      Consul agent. There are many reasons for this, most importantly
      the Consul agent is able to multiplex connections to the Consul
      server and reduce the number of open HTTP connections. Additionally,
      it provides a "well-known" IP address for which clients can connect.

    -consul-key-location=<key>
      The Consul Key/Value Store location that Replicator will use
      for persistent configuration, state tracking and job scaling policies.
			By default, this is replicator/config.

    -consul-token=<token>
      The Consul ACL token to use when communicating with an ACL
      protected Consul cluster.

  Failsafe Mode Options:

    -disable
      Disable the global failsafe lock. All copies of Replicator wil
      return to normal operations.

    -enable
      Enable the global failsafe lock. All copies of Replicator will
      be prohibited from taking any scaling actions.

    -force
      Suppress confirmation prompts when enabling or disabling the
      global failsafe lock.
`
	return strings.TrimSpace(helpText)
}

// Synopsis is provides a brief summary of the failsafe command.
func (c *FailsafeCommand) Synopsis() string {
	return "Provide an administrative interface to control failsafe mode."
}

// Run triggers the failsafe command to update the distributed state tracking
// data and manipulate the failsafe lock.
func (c *FailsafeCommand) Run(args []string) int {
	// Initialize an new empty state tracking object.
	state := &structs.State{}

	// The operator must specify at least one operation.
	if len(args) == 0 {
		c.UI.Error(c.Help())
		return 1
	}

	// Parse flags and generate a resulting configuration.
	c.args = args
	conf := c.parseFlags()
	if conf == nil {
		return 1
	}

	// Setup the Consul client.
	if err := base.InitializeClients(conf.Config); err != nil {
		c.UI.Error(fmt.Sprintf("An error occurred while attempting to initialize "+
			"the Consul client: %v", err))
		return 1
	}

	// Grab the initialized consul client from the returned configuration object.
	consul := conf.Config.ConsulClient

	// Check that we were sent either enable or disable, but not both.
	if (conf.Enable && conf.Disable) || (!conf.Enable && !conf.Disable) {
		c.UI.Error(c.Help())
		return 1
	}

	// Attempt to load state tracking data from Consul.
	state = consul.LoadState(conf.Config, state)

	// If failsafe mode is already in the desired state, report and take no
	// action.
	if state.FailsafeMode && conf.Enable || !state.FailsafeMode && conf.Disable {
		c.UI.Warn(fmt.Sprintf("Failsafe mode is already in desired state \"%vd\""+
			", no action required.", conf.Verb))
		return 0
	}

	// If the user has not disabled confirmation prompts, ask for confirmation.
	if !conf.Force {
		keyLocation := conf.Config.ConsulKeyLocation + "/" + "state"

		question := fmt.Sprintf("Are you sure you want to %s the global failsafe "+
			"lock stored at %q?\n", conf.Verb, keyLocation)

		// If we're enabling failsafe mode, give the user a clear warning about
		// the implications.
		if conf.Enable {
			question = fmt.Sprintf("%vNo scaling operations will be permitted "+
				"from any running copies of Replicator.\n", question)
		}

		// Ask for confirmation and parse the response.
		answer, err := c.UI.Ask(fmt.Sprintf("%vConfirm [y/N]: ", question))
		if err != nil {
			c.UI.Error(fmt.Sprintf("Failed to parse answer: %v", err))
			return 1
		}

		// Validate the confirmation response.
		if answer == "" || strings.ToLower(answer)[0] == 'n' {
			c.UI.Output(fmt.Sprintf("Cancelling, will not %v failsafe mode.",
				conf.Verb))
			return 0
		} else if strings.ToLower(answer)[0] == 'y' && len(answer) > 1 {
			c.UI.Output("For confirmation, an exact 'y' is required.")
			return 0
		} else if answer != "y" {
			c.UI.Output("No confirmation detected. For confirmation, an exact 'y' " +
				"is required.")
			return 1
		}
	}

	// Indicate that failsafe mode was administratively updated.
	state.FailsafeModeAdmin = true

	// Set desired failsafe mode.
	if err := core.SetFailsafeMode(state, conf.Config, conf.Enable); err != nil {
		c.UI.Error(fmt.Sprintf("An error occurred while attempting to %v "+
			"failsafe mode: %v", conf.Verb, err))
		return 1
	}

	c.UI.Info(fmt.Sprintf("Successfully %vd failsafe mode.", conf.Verb))

	return 0
}

func (c *FailsafeCommand) parseFlags() *structs.FailsafeMode {
	var configPath string

	// Initialize an empty configuration object that will be populated with
	// any passed CLI flags for later merging.
	cliConfig := &structs.FailsafeMode{
		Config: &structs.Config{},
	}

	// Initialize command flags.
	flags := c.Meta.FlagSet("failsafe", FlagSetClient)
	flags.Usage = func() { c.UI.Error(c.Help()) }

	// General configuration flags.
	flags.StringVar(&configPath, "config", "", "")
	flags.StringVar(&cliConfig.Config.Consul, "consul", "", "")
	flags.StringVar(&cliConfig.Config.ConsulToken, "consul-token", "", "")
	flags.StringVar(&cliConfig.Config.ConsulKeyLocation,
		"consul-key-location", "", "")

	// Failsafe mode configuration flags.
	flags.BoolVar(&cliConfig.Enable, "enable", false, "Enable failsafe mode")
	flags.BoolVar(&cliConfig.Disable, "disable", false, "Disable failsafe mode")
	flags.BoolVar(&cliConfig.Force, "force", false,
		"Supress confirmation prompts.")

	// Parse the passed CLI flags.
	if err := flags.Parse(c.args); err != nil {
		return nil
	}

	// Determine the appropriate verbage for confirmation prompts.
	cliConfig.Verb = "enable"
	if cliConfig.Disable {
		cliConfig.Verb = "disable"
	}

	// Create default configuration object on which to base the merge.
	config := base.DefaultConfig()

	// If a configuration path has been specified, load configuration from the
	// specified location.
	if configPath != "" {
		current, err := base.LoadConfig(configPath)
		if err != nil {
			c.UI.Error(fmt.Sprintf("Error loading configuration from %s: %s",
				configPath, err))
			return nil
		}

		// Merge loaded configuration with the default configuration.
		config = config.Merge(current)
	}

	// Merge passed CLI flags with the configuration derived from the defaults
	// and optionally, the loaded configuration.
	cliConfig.Config = config.Merge(cliConfig.Config)

	return cliConfig
}
