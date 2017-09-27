package command

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

const (
	defaultJobScalingName     = "job_scaling.hcl"
	defaultClusterScalingName = "cluster_scaling.hcl"
)

// InitCommand is the command implamentation for init.
type InitCommand struct {
	Meta
}

// Help provides the help information for the init command.
func (c *InitCommand) Help() string {
	helpText := `
Usage: replicator init [options]

  Creates example job and cluster scaling configurations.

  General Options:

    -job-scaling
      Write a file which contains example job scaling configuration. This
      can be used directly within the Nomad job specification file to enable
      scaling for the desired job group.

    -cluster-scaling
      Write a file which contains example cluster scaling configuration.
      This can be adapted to your configuration management to enable
      cluster scaling.
`
	return strings.TrimSpace(helpText)
}

// Synopsis is provides a brief summary of the init command.
func (c *InitCommand) Synopsis() string {
	return "Create example Replicator job and cluster scaling configurations"
}

// Run triggers the init command to write the example.json file out to the
// current directory.
func (c *InitCommand) Run(args []string) int {

	var jobScaling, clusterScaling bool

	// Initialize command flags.
	flags := c.Meta.FlagSet("init", FlagSetClient)
	flags.Usage = func() { c.UI.Error(c.Help()) }

	// General configuration flags.
	flags.BoolVar(&jobScaling, "job-scaling", false, "")
	flags.BoolVar(&clusterScaling, "cluster-scaling", false, "")

	// Parse the passed CLI flags.
	if err := flags.Parse(args); err != nil {
		return 1
	}

	if jobScaling || len(args) == 0 {
		err := c.writeFile(defaultJobScalingName, defaultJobScalingDocument)
		if err != nil {
			c.UI.Error(fmt.Sprintf("Unable to write file %s: %v", defaultJobScalingName, err))
			return 1
		}
	}

	if clusterScaling || len(args) == 0 {
		err := c.writeFile(defaultClusterScalingName, defaultClusterScalingDocument)
		if err != nil {
			c.UI.Error(fmt.Sprintf("Unable to write file %s: %v", defaultClusterScalingName, err))
			return 1
		}
	}
	return 0
}

func (c *InitCommand) writeFile(file, content string) (err error) {

	// Check if the file already exists.
	if _, err = os.Stat(file); err != nil && !os.IsNotExist(err) {
		return err
	}

	if !os.IsNotExist(err) {
		return fmt.Errorf("scaling document already exists")
	}

	// Write the example file to the relative local directory where Replicator
	// was invoked from.
	if err = ioutil.WriteFile(file, []byte(content), 0660); err != nil {
		return err
	}

	c.UI.Info(fmt.Sprintf("Example scaling configuration written to %s", file))
	return
}

var defaultJobScalingDocument = strings.TrimSpace(`
meta {
  "replicator_cooldown"         = 50
  "replicator_enabled"          = true
  "replicator_max"              = 10
  "replicator_min"              = 1
  "replicator_notification_uid" = "REP1"
  "replicator_scalein_mem"      = 30
  "replicator_scalein_cpu"      = 30
  "replicator_scaleout_mem"     = 80
  "replicator_scaleout_cpu"     = 80
}
`)

var defaultClusterScalingDocument = strings.TrimSpace(`
meta {
  "replicator_cool_down"            = 400
  "replicator_enabled"              = true
  "replicator_max"                  = 10
  "replicator_min"                  = 5
  "replicator_node_fault_tolerance" = 1
  "replicator_notification_uid"     = "REP2"
  "replicator_region"               = "us-east-1"
  "replicator_retry_threshold"      = 3
  "replicator_scaling_threshold"    = 3
  "replicator_worker_pool"          = "container-node-public-prod"
}
`)
