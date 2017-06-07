package agent

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	metrics "github.com/armon/go-metrics"
	"github.com/elsevier-core-engineering/replicator/command"
	"github.com/elsevier-core-engineering/replicator/command/base"
	"github.com/elsevier-core-engineering/replicator/logging"
	"github.com/elsevier-core-engineering/replicator/replicator"
	"github.com/elsevier-core-engineering/replicator/replicator/structs"
	"github.com/elsevier-core-engineering/replicator/version"
)

// Command is the agent command structure used to track passed args as well as
// the CLI meta.
type Command struct {
	command.Meta
	args []string
}

// Run triggers a run of the replicator agent by setting up and parsing the
// configuration and then initiating a new runner.
func (c *Command) Run(args []string) int {

	c.args = args
	conf := c.parseFlags()
	if conf == nil {
		return 1
	}

	err := c.initialzeAgent(conf)
	if err != nil {
		logging.Error("command/agent: unable to initialize agent: %v", err)
		return 1
	}

	// Create the initial runner with the merged configuration parameters.
	runner, err := replicator.NewRunner(conf)
	if err != nil {
		return 1
	}

	logging.Info("command/agent: running version %v", version.Get())
	logging.Info("command/agent: starting replicator agent...")
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
				logging.Info("command/agent: caught signal %v", s)
				runner.Stop()
				return 1

			case syscall.SIGHUP:
				logging.Info("command/agent: caught signal %v", s)
				runner.Stop()

				// Reload the configuration in order to make proper use of SIGHUP.
				config := c.parseFlags()
				if err != nil {
					return 1
				}

				err := c.initialzeAgent(config)
				if err != nil {
					logging.Error("command/agent: unable to initialize agent: %v", err)
					return 1
				}

				// Setup a new runner with the new configuration.
				runner, err = replicator.NewRunner(config)
				if err != nil {
					return 1
				}

				go runner.Start()
			}
		}
	}
}

func (c *Command) parseFlags() *structs.Config {

	var configPath string
	var dev bool

	// An empty new config is setup here to allow us to fill this with any passed
	// cli flags for later merging.
	cliConfig := &structs.Config{
		ClusterScaling: &structs.ClusterScaling{},
		JobScaling:     &structs.JobScaling{},
		Telemetry:      &structs.Telemetry{},
		Notification:   &structs.Notification{},
	}

	flags := c.Meta.FlagSet("agent", command.FlagSetClient)
	flags.Usage = func() { c.UI.Error(c.Help()) }

	flags.StringVar(&configPath, "config", "", "")
	flags.BoolVar(&dev, "dev", false, "")

	// Top level configuration flags
	flags.StringVar(&cliConfig.Region, "aws-region", "", "")
	flags.StringVar(&cliConfig.Consul, "consul", "", "")
	flags.StringVar(&cliConfig.ConsulKeyLocation, "consul-key-location", "", "")
	flags.StringVar(&cliConfig.ConsulToken, "consul-token", "", "")
	flags.StringVar(&cliConfig.LogLevel, "log-level", "", "")
	flags.StringVar(&cliConfig.Nomad, "nomad", "", "")
	flags.IntVar(&cliConfig.ScalingInterval, "scaling-interval", 0, "")

	// Cluster scaling configuration flags
	flags.BoolVar(&cliConfig.ClusterScaling.Enabled, "cluster-scaling-enabled", false, "")
	flags.IntVar(&cliConfig.ClusterScaling.MaxSize, "cluster-max-size", 0, "")
	flags.IntVar(&cliConfig.ClusterScaling.MinSize, "cluster-mix-size", 0, "")
	flags.Float64Var(&cliConfig.ClusterScaling.CoolDown, "cluster-scaling-cool-down", 0, "")
	flags.IntVar(&cliConfig.ClusterScaling.NodeFaultTolerance, "cluster-node-fault-tolerance", 0, "")
	flags.StringVar(&cliConfig.ClusterScaling.AutoscalingGroup, "cluster-autoscaling-group", "", "")
	flags.IntVar(&cliConfig.ClusterScaling.RetryThreshold, "cluster-retry-threshold", 0, "")

	// Job scaling configuration flags
	flags.BoolVar(&cliConfig.JobScaling.Enabled, "job-scaling-enabled", false, "")

	// Telemetry configuration flags
	flags.StringVar(&cliConfig.Telemetry.StatsdAddress, "statsd-address", "", "")

	// Notification configuration flags
	flags.StringVar(&cliConfig.Notification.ClusterScalingUID, "cluster-scaling-uid", "", "")
	flags.StringVar(&cliConfig.Notification.ClusterIdentifier, "cluster-identifier", "", "")
	flags.StringVar(&cliConfig.Notification.PagerDutyServiceKey, "pagerduty-service-key", "", "")

	if err := flags.Parse(c.args); err != nil {
		return nil
	}

	// Depending on the flags provided (if any) we load a default configuration
	// which will be the basis for all merging.
	var config *structs.Config

	if dev {
		config = base.DevConfig()
	} else {
		config = base.DefaultConfig()
	}

	if configPath != "" {
		current, err := base.LoadConfig(configPath)
		if err != nil {
			c.UI.Error(fmt.Sprintf("Error loading configuration from %s: %s", configPath, err))
			return nil
		}

		config = config.Merge(current)
	}

	config = config.Merge(cliConfig)
	return config

}

// setupTelemetry is used to setup Replicators telemetry.
func (c *Command) setupTelemetry(config *structs.Telemetry) error {

	// Setup telemetry to aggregate on 10 second intervals for 1 minute.
	inm := metrics.NewInmemSink(10*time.Second, time.Minute)
	metrics.DefaultInmemSignal(inm)

	var telemetry *structs.Telemetry
	if config == nil {
		telemetry = &structs.Telemetry{}
	} else {
		telemetry = config
	}

	metricsConf := metrics.DefaultConfig("replicator")

	var fanout metrics.FanoutSink

	// Configure the statsd sink
	if telemetry.StatsdAddress != "" {
		sink, err := metrics.NewStatsdSink(telemetry.StatsdAddress)
		if err != nil {
			return err
		}
		fanout = append(fanout, sink)
	}

	// Initialize the global sink
	if len(fanout) > 0 {
		fanout = append(fanout, inm)
		metrics.NewGlobal(metricsConf, fanout)
	} else {
		metricsConf.EnableHostname = false
		metrics.NewGlobal(metricsConf, inm)
	}
	return nil
}

// initialzeAgent setups up a number of configuration clients which depend on
// the merged configuration.
func (c *Command) initialzeAgent(config *structs.Config) (err error) {

	// Setup telemetry
	if err = c.setupTelemetry(config.Telemetry); err != nil {
		return
	}

	// Setup logging
	logging.SetLevel(config.LogLevel)

	// Setup the Consul and Nomad clients
	if err = base.InitializeClients(config); err != nil {
		return
	}

	return nil
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

    -aws-region=<region>
      The AWS region in which the cluster is running. If no region is
      specified, Replicator attempts to dynamically determine the region.

    -config=<path>
      The path to either a single config file or a directory of config
      files to use for configuring the Replicator agent. Replicator
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
      for persistent configuration and job scaling policies. By default,
      this is replicator/config.

    -consul-token=<token>
      The Consul ACL token to use when communicating with an ACL
      protected Consul cluster.

    -dev
      Start the Replicator agent in development mode. This runs the
      Replicator agent with a configuration which is ideal for development
      or local testing.

    -log-level=<level>
      Specify the verbosity level of Replicator's logs. The default is
      INFO.

    -nomad=<address:port>
      The address and port Replicator will use when making connections
      to the Nomad API. By default, this http://localhost:4646, which
      is the default bind and port for a local Nomad server.

    -scaling-interval=<num>
      The time period in seconds between Replicator check runs. The
      default is 10.

  Cluster Scaling Options:

    -cluster-autoscaling-group=<name>
      The name of the AWS autoscaling group that contains the worker
      nodes. This should be a separate ASG from the one containing
      the server nodes.

    -cluster-max-size=<num>
      Indicates the maximum number of worker nodes allowed in the cluster.
      The default is 10.

    -cluster-min-size=<num>
      Indicates the minimum number of worker nodes allowed in the cluster.
      The default is 5.

    -cluster-node-fault-tolerance=<num>
      The number of worker nodes the cluster can tolerate losing while
      still maintaining sufficient operation capacity. This is used by
      the scaling algorithm when calculating allowed capacity consumption.
      The default is 1.

    -cluster-retry-threshold=<num>
      Replicator fully verifies cluster scale-out by confirming the node
      joins the cluster. If it does not join after a certain period the
      actioned is marked as failed. This retry is the number of times
      Replicator will attempt to scale the cluster with new instances.

    -cluster-scaling-cool-down=<num>
      The number of seconds Replicator will wait between triggering
      cluster scaling actions. The default is 600.

    -cluster-scaling-enabled
      Indicates whether the daemon should perform scaling actions. If
      disabled, the actions that would have been taken will be reported
      in the logs but skipped.

  Job Scaling Options:

    -job-scaling-enabled
      Indicates whether the daemon should perform scaling actions. If
      disabled, the actions that would have been taken will be reported
      in the logs but skipped.

  Telemetry Options:

    -statsd-address=<address:port>
      Specifies the address of a statsd server to forward metrics
      to and should include the port.

  Notifications Options:

    -cluster-identifier=<name>
      A human readable cluster name to allow operators to quickly identify
      which cluster is alerting.

    -cluster-scaling-uid=<uid>
      The cluster UID is an identifier which represents a run book entry
      which allows operators and support to quickly work through
      resolution steps.

    -pagerduty-service-key=<key>
      The PagerDuty integration key which has been setup to allow
      replicator to send events.

`
	return strings.TrimSpace(helpText)
}

// Synopsis is provides a brief summary of the agent command.
func (c *Command) Synopsis() string {
	return "Runs a Replicator agent"
}
