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

The agent command is the main entry point into Replicator. A subset of the available replicator agent configuration can optionally be passed in via CLI arguments and the configuration parameters passed via CLI flags will always take precedent over parameters specified in configuration files. Detailed information regarding the available CLI flags can be found in the Replicator [Agent Configuration wiki page](https://github.com/elsevier-core-engineering/replicator/wiki/Agent_Configuration#command-line-flags).

### Command: init

The init command creates an example job scaling document in the current directory. This document can then be manipulated to meet your requirements, or be used to test replicator against the [Nomad init](https://www.nomadproject.io/docs/commands/init.html) example job.

### Command: version

The version command displays build information about the running binary, including the release version.

## Configuration File Syntax

Replicator uses the [HashiCorp Configuration Language](https://github.com/hashicorp/hcl) for configuration files. By proxy, this means the configuration is also JSON compatible. Additional information can be found on the Replicator [Agent Configuration Configuration Files ](https://github.com/elsevier-core-engineering/replicator/wiki/Agent_Configuration#configuration-files) wiki section.

## Job Scaling Policy Syntax

Replicator uses JSON documents stored in the Consul Key/Value Store for job scaling policies. This should be placed under /jobs of the `consul-key-location` value, meaning if the default `consul-key-location` is used scaling documents should be written to `replicator/config/jobs`. The key value should match the name of the job it should interact with as well as the groups name sections:

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
