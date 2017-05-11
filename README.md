# Replicator

[![Build Status](https://travis-ci.org/elsevier-core-engineering/replicator.svg?branch=master)](https://travis-ci.org/elsevier-core-engineering/replicator) [![Join the chat at https://gitter.im/els-core-engineering/replicator/Lobby](https://badges.gitter.im/els-core-engineering/replicator/Lobby.svg)](https://gitter.im/els-core-engineering/replicator?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge) [![GoDoc](https://godoc.org/github.com/elsevier-core-engineering/replicator?status.svg)](https://godoc.org/github.com/elsevier-core-engineering/replicator)

Replicator is a daemon that provides automatic scaling of [Nomad](https://github.com/hashicorp/nomad) jobs and worker nodes.

- Replicator stores job scaling policies in the Consul Key/Value Store. A job scaling policy allows scaling constraints to be defined per task-group.
- Replicator supports automatic scaling of cluster worker nodes in an AWS autoscaling group.

At present, cluster autoscaling is only supported on AWS but future support for GCE and Azure are planned.

## Installation

- Download the appropriate pre-compiled release for your platform from the [GitHub release page](https://github.com/elsevier-core-engineering/replicator/releases).
- Sample unit files and associated instructions are available in the [`dist`](https://github.com/elsevier-core-engineering/replicator/tree/master/dist) directory.

A [puppet module](https://github.com/elsevier-core-engineering/puppet-replicator) capable of automatically installing and configuring Replicator is also available.

## Commands

Replicator supports a number of commands (CLI) which allow for the easy control and manipulation of the replicator binary.

### Command: agent

The agent command is the main entry point into Replicator. A subset of the available replicator agent configuration can optionally be passed in via CLI arguments and the configuration parameters passed via CLI flags will always take precedent over parameters specified in configuration files.

- **-config=<path>** The path to either a single config file or a directory of config files to use for configuring the Replicator agent. Replicator processes configuration files in lexicographic order.

- **-consul=<address:path>** This is the address of the Consul agent. By default, this is localhost:8500, which is the default bind and port for a local Consul agent. It is not recommended that you communicate directly with a Consul server, and instead communicate with the local Consul agent. There are many reasons for this, most importantly the Consul agent is able to multiplex connections to the Consul server and reduce the number of open HTTP connections. Additionally, it provides a "well-known" IP address for which clients can connect.

- **-nomad=<address:path>** The address and port Replicator will use when making connections to the Nomad API. By default, this http://localhost:4646, which is the default bind and port for a local Nomad server.

- **-log-level=<level>** Specify the verbosity level of Replicator's logs. The default is INFO.

- **-scaling-interval=<num>** The time period in seconds between Replicator check runs. The default is 10.

- **-aws-region=<region>** The AWS region in which the cluster is running. If no region is specified, Replicator attempts to dynamically determine the region.

- **-cluster-scaling-enabled** Indicates whether the daemon should perform cluster scaling actions. If disabled, the actions that would have been taken will be reported in the logs but skipped.

- **-cluster-max-size=<num>** Indicates the maximum number of worker nodes allowed in the cluster. The default is 10.

- **-cluster-min-size=<num>** Indicates the minimum number of worker nodes allowed in the cluster. The default is 5.

- **-cluster-scaling-cool-down=<num>** The number of seconds Replicator will wait between triggering cluster scaling actions. The default is 600.

- **-cluster-node-fault-tolerance=<num>** The number of worker nodes the cluster can tolerate losing while still maintaining sufficient operation capacity. This is used by the scaling algorithm when calculating allowed capacity consumption. The default is 1.

- **-cluster-autoscaling-group=<name>** The name of the AWS autoscaling group that contains the worker nodes. This should be a separate ASG from the one containing the server nodes.

- **-job-scaling-enabled** Indicates whether the daemon should perform job scaling actions. If disabled, the actions that would have been taken will be reported in the logs but skipped.

- **-consul-token=<token>** The Consul ACL token to use when communicating with an ACL protected Consul cluster.

- **-consul-key-location=<key>** The Consul Key/Value Store location where Replicator will look for job scaling policies. By default, this is replicator/config/jobs.

- **-statsd-address=<address:port>** Specifies the address of a StatsD server to forward metrics to and should include the port.

### Command: init

The init command creates an example job scaling document in the current directory. This document can then be manipulated to meet your requirements, or be used to test replicator against the [Nomad init](https://www.nomadproject.io/docs/commands/init.html) example job.

### Command: version

The version command displays build information about the running binary, including the release version.

## Configuration File Syntax

Replicator uses the [HashiCorp Configuration Language](https://github.com/hashicorp/hcl) for configuration files. By proxy, this means the configuration is also JSON compatible.

You can specify a configuration file or a directory that contains configuration files using the `-config` flag. Replicator processes configuration files in lexicographic order.

```
# This is the address of the Consul agent. By default, this is
# localhost:8500, which is the default bind and port for a local Consul
# agent. It is not recommended that you communicate directly with a Consul
# server, and instead communicate with the local Consul agent. There are many
# reasons for this, most importantly the Consul agent is able to multiplex
# connections to the Consul server and reduce the number of open HTTP
# connections. Additionally, it provides a "well-known" IP address for which
# clients can connect.
consul     = "localhost:8500"

# The address and port Replicator will use when making connections to the Nomad API. By default, this http://localhost:4646, which is the default bind and port for a local Nomad server.
nomad      = "http://localhost:4646"

# The AWS region in which the cluster is running. If no region is specified, Replicator attempts to dynamically determine the region.
aws_region = "us-east-1"

# The log level the daemon should use.
log_level  = "info"

# The time period in seconds between replicator check runs.
scaling_interval = 10

# This denotes the start of the configuration section for cluster autoscaling.
cluster_scaling {
  # Indicates whether the daemon should perform scaling actions. If disabled, the actions that would have been taken will be reported in the logs but skipped.
  enabled              = true

  # Indicates the maximum number of worker nodes allowed in the cluster.
  max_size             = 4

  # Indicates the minimum number of worker nodes allowed in the cluster.
  min_size             = 2

  # The number of seconds Replicator will wait between scaling actions.
  cool_down            = 120

  # The number of worker nodes the cluster can tolerate losing while still maintaining sufficient operation capacity. This is used by the scaling algorithm when calculating allowed capacity consumption.
  node_fault_tolerance = 1

  # The name of the AWS autoscaling group that contains the worker nodes. This should be a separate ASG from the one containing the server nodes.
  autoscaling_group    = "container-agent-dev"
}

# This denotes the start of the configuration section for job autoscaling.
job_scaling {
  # The Consul Key/Value Store location where Replicator will look for job scaling policies. By default, this is `replicator/config/jobs`.
  consul_key_location = "replicator/config/jobs"

  # The Consul ACL token to use when communicating with an ACL protected Consul cluster.
  consul_token        = "278F9E37-8322-4EDC-AFDC-748D116B3DCE"

  # Indicates whether the daemon should perform scaling actions. If disabled, the actions that would have been taken will be reported in the logs but skipped.
  enabled             = true
}
```

## Job Scaling Policy Syntax

Replicator uses JSON documents stored in the Consul Key/Value Store for job scaling policies:

`replicator/config/jobs/samplejob`
```
{
 "enabled": true,
 "groups": [
   {
     "name": "group1",
     "scaling": {
       "min": 3,
       "max": 10,
       "scaleout": {
         "cpu": 80,
         "mem": 80
       },
       "scalein": {
         "cpu": 30,
         "mem": 30
       }
     }
   },
   {
     "name": "group2",
     "scaling": {
       "min": 2,
       "max": 6,
       "scaleout": {
         "cpu": 80,
         "mem": 80
       },
       "scalein": {
         "cpu": 30,
         "mem": 30
       }
     }
   }
 ]
}
```

## Permissions

The server node running the Replicator daemon will need access to certain AWS resources and actions. An example IAM policy is provided below:

```
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "Authorize AutoScaling Actions",
      "Action": [
        "autoscaling:DescribeAutoScalingGroups",
        "autoscaling:DescribeAutoScalingInstances",
        "autoscaling:DescribeScalingActivities",
        "autoscaling:DetachInstances",
        "autoscaling:UpdateAutoScalingGroup"
      ],
      "Effect": "Allow",
      "Resource": "*"
    },
    {
      "Sid": "Authorize EC2 Actions",
      "Action": [
        "ec2:DescribeInstances",
        "ec2:DescribeRegions",
        "ec2:TerminateInstances"
      ],
      "Effect": "Allow",
      "Resource": "*"
    }
  ]
}
```

When writing an IAM policy to control access to Auto Scaling actions, you must use `*` as the resource. There are no supported Amazon Resource Names (ARNs) for Auto Scaling resources.

## Frequently Asked Questions

### When does Replicator adjust the size of the worker pool?

Replicator will dynamically scale-in the worker pool when:
- Resource utilization falls below the capacity required to run all current jobs while sustaining the configured node fault-tolerance. When calculating required capacity, Replicator includes scaling overhead required to increase the count of all running jobs by one.
- Before removing a worker node, Replicator simulates capacity thresholds if we were to remove a node. If the new required capacity is within 10% of the current utilization, Replicator will decline to remove a node to prevent thrashing.

Replicator will dynamically scale-out the worker pool when:
- Resource utilization exceeds or closely approaches the capacity required to run all current jobs while sustaining the configured node fault-tolerance. When calculating required capacity, Replicator includes scaling overhead required to increase the count of all running jobs by one.

### When does Replicator perform scaling actions against running jobs?

Replicator will dynamically scale a job when:
- A valid scaling policy for the job is present at the appropriate location within the Consul Key/Value Store and is enabled.
- A job specification can consist of multiple groups, each group can contain multiple tasks. Resource allocations and count are specified at the group level.
- Replicator evaluates scaling thresholds against the resource requirements defined within a group task. If any task within a group is found to violate the scaling thresholds, the group count will be adjusted accordingly.

### How does Replicator prioritize cluster autoscaling and job autoscaling?

Replicator prioritized cluster scaling before job scaling to ensure the worker pool always has sufficient capacity to accommodate job scaling actions.

- If Replicator initiates a cluster scaling action, the daemon blocks until this action is complete. During this time, no job scaling activity will be undertaken.

## Contributing

Contributions to Replicator are very welcome! Please refer to our [contribution guide](https://github.com/elsevier-core-engineering/replicator/blob/master/.github/CONTRIBUTING.md) for details about hacking on Replicator.

For questions, please check out the [Elsevier Core Engineering/replicator](https://gitter.im/els-core-engineering/replicator) room in Gitter.
