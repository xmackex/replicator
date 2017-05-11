package command

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

const (
	// DefaultInitName is the default name we use when
	// initializing the example file
	DefaultInitName = "example.json"
)

type InitCommand struct {
	Meta
}

// Help provides the help information for the init command.
func (c *InitCommand) Help() string {
	helpText := `
Usage: replicator init

  Creates an example scaling document that can be used as a
  starting point to customize further. The example is designed
  to work with the 'nomad init' job example.
`
	return strings.TrimSpace(helpText)
}

// Synopsis is provides a brief summary of the init command.
func (c *InitCommand) Synopsis() string {
	return "Create an example Replicator job scaling document"
}

// Run triggers the init command to write the example.json file out to the
// current directory.
func (c *InitCommand) Run(args []string) int {

	// The command should be used with 0 extra flags.
	if len(args) != 0 {
		c.UI.Error(c.Help())
		return 1
	}

	// Check if the file already exists.
	_, err := os.Stat(DefaultInitName)
	if err != nil && !os.IsNotExist(err) {
		c.UI.Error(fmt.Sprintf("Failed to stat '%s': %v", DefaultInitName, err))
		return 1
	}
	if !os.IsNotExist(err) {
		c.UI.Error(fmt.Sprintf("Scaling document '%s' already exists", DefaultInitName))
		return 1
	}

	// Write the example file to the relative local directory where Replicator
	// was invoked from.
	err = ioutil.WriteFile(DefaultInitName, []byte(defaultScalingDocument), 0660)
	if err != nil {
		c.UI.Error(fmt.Sprintf("Failed to write '%s': %v", DefaultInitName, err))
		return 1
	}

	c.UI.Output(fmt.Sprintf("Example scaling document file written to %s", DefaultInitName))
	return 0
}

var defaultScalingDocument = strings.TrimSpace(`
{"enabled":true,"groups":[{"name":"cache","scaling":{"min":1,"max":3,"scaleout":{"cpu":80,"mem":80},"scalein":{"cpu":30,"mem":30}}}]}
`)
