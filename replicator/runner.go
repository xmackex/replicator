package replicator

import (
	"time"

	"github.com/elsevier-core-engineering/replicator/client"
	"github.com/elsevier-core-engineering/replicator/logging"
	"github.com/elsevier-core-engineering/replicator/replicator/structs"
)

// Runner is the main runner struct.
type Runner struct {
	// doneChan is where finish notifications occur.
	doneChan chan struct{}

	// config is the Config that created this Runner. It is used internally to
	// construct other objects and pass data.
	config *structs.Config
}

// NewRunner sets up the Runner type.
func NewRunner(config *structs.Config) (*Runner, error) {
	runner := &Runner{
		doneChan: make(chan struct{}),
		config:   config,
	}
	return runner, nil
}

// Start creates a new runner and uses a ticker to block until the doneChan is
// closed at which point the ticker is stopped.
func (r *Runner) Start() {
	ticker := time.NewTicker(time.Second * time.Duration(r.config.ScalingInterval))

	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:

			clusterChan := make(chan bool)
			go r.clusterScaling(clusterChan)
			<-clusterChan

			r.jobScaling()

		case <-r.doneChan:
			return
		}
	}
}

// Stop halts the execution of this runner.
func (r *Runner) Stop() {
	close(r.doneChan)
}

// clusterScaling is the main entry point into the cluster scaling functionality
// and ties numerous functions together to create an asynchronus function which
// can be called from the runner.
func (r *Runner) clusterScaling(done chan bool) {
	nomadClient := r.config.NomadClient
	scalingEnabled := r.config.ClusterScaling.Enabled

	// Determine if we are running on the leader node, halt if not.
	if haveLeadership := nomadClient.LeaderCheck(); !haveLeadership {
		logging.Debug("core/runner: replicator is not running on the known leader, no cluster scaling actions will be taken")
		done <- true
		return
	}

	// If a region has not been specified, attempt to dynamically determine what
	// region we are running in.
	if r.config.Region == "" {
		if region, err := client.DescribeAWSRegion(); err == nil {
			r.config.Region = region
		}
	}

	clusterCapacity := &structs.ClusterStatus{}

	if scale, err := nomadClient.EvaluateClusterCapacity(clusterCapacity, r.config); err != nil || !scale {
		logging.Debug("core/runner: scaling operation not required or permitted")
	} else {
		// If we reached this point we will be performing AWS interaction so we
		// create an client connection.
		asgSess := client.NewAWSAsgService(r.config.Region)

		if clusterCapacity.ScalingDirection == client.ScalingDirectionOut {
			// If cluster scaling has been disabled, report but do not initiate a
			// scaling operation.
			if !scalingEnabled {
				logging.Debug("core/runner: cluster scaling disabled, not initiating " +
					"scaling operation (scale-out)")
				done <- true
				return
			}

			// Attempt to increment the desired count of the autoscaling group. If
			// this fails, log an error and stop further processing.
			if err := client.ScaleOutCluster(r.config.ClusterScaling.AutoscalingGroup, asgSess); err != nil {
				logging.Error("core/runner: unable to successfully initiate a "+
					"scaling operation against autoscaling group %v: %v",
					r.config.ClusterScaling.AutoscalingGroup, err)
				done <- true
				return
			}

			// Attempt to add a new node to the worker pool until we reach the
			// retry threshold.
			// TODO (e.westfall): Make the node failure retry threshold a config
			// option. Waiting on this until after the merge to take advantage of
			// config flag changes.
			for clusterCapacity.NodeFailureCount <= 3 {
				if clusterCapacity.NodeFailureCount > 0 {
					logging.Info("core/runner: attempting to launch a new worker node, "+
						"previous node failures: %v", clusterCapacity.NodeFailureCount)
				}

				// We've verified the autoscaling group operation completed successfully.
				// Next we'll identify the most recently launched EC2 instance from the
				// worker pool ASG.
				newestNode, err := client.GetMostRecentInstance(
					r.config.ClusterScaling.AutoscalingGroup,
					r.config.Region,
				)
				if err != nil {
					logging.Error("core/runner: Failed to identify the most recently "+
						"launched instance: %v", err)
					clusterCapacity.NodeFailureCount++
					continue
				}

				// Attempt to verify the new worker node has completed bootstrapping and
				// successfully joined the worker pool.
				healthy := nomadClient.VerifyNodeHealth(newestNode)
				if healthy {
					// Reset node failure count once we have a verified healthy worker.
					clusterCapacity.NodeFailureCount = 0
					done <- true
					return
				}

				clusterCapacity.NodeFailureCount++
				logging.Error("core/runner: new node %v failed to successfully join "+
					"the worker pool, incrementing node failure count to %v and "+
					"terminating instance", newestNode, clusterCapacity.NodeFailureCount)

				// Translate the IP address of the most recent instance to the EC2
				// instance ID.
				instanceID := client.TranslateIptoID(newestNode, r.config.Region)

				// If we've reached the retry threshold, disable cluster scaling and
				// halt.
				if disabled := r.disableClusterScaling(clusterCapacity); disabled {
					// Detach the last failed instance and decrement the desired count
					// of the autoscaling group. This will leave the instance around
					// for debugging purposes but allow us to cleanly resume cluster
					// scaling without intervention.
					err := client.DetachInstance(
						r.config.ClusterScaling.AutoscalingGroup, instanceID, asgSess,
					)
					if err != nil {
						logging.Error("core/runner: an error occurred while attempting "+
							"to detach the failed instance from the ASG: %v", err)
					}

					done <- true
					return
				}

				// Attempt to clean up the most recent instance.
				if err := client.TerminateInstance(instanceID, r.config.Region); err != nil {
					logging.Error("core/runner: an error occurred while attempting "+
						"to terminate instance %v: %v", instanceID, err)
				}

				// Pause after initiating termination of instance before retrying. This
				// is a safety mechanism to deal with how slow the AWS autoscaling
				// system is to respond to external facters such as an instance being
				// terminated.
				//
				// TODO (e.westfall): Remove this after we place a ticker wait in
				// the TerminateInstance() method to verify the instance has actually
				// been terminated.
				time.Sleep(45 * time.Second)
			}
		}

		if clusterCapacity.ScalingDirection == client.ScalingDirectionIn {
			nodeID, nodeIP := nomadClient.LeastAllocatedNode(clusterCapacity)
			if nodeIP != "" && nodeID != "" {
				if !scalingEnabled {
					logging.Debug("core/runner: cluster scaling disabled, not " +
						"initiating scaling operation (scale-in)")
					done <- true
					return
				}

				if err := nomadClient.DrainNode(nodeID); err == nil {
					logging.Info("core/runner: terminating AWS instance %v", nodeIP)
					err := client.ScaleInCluster(r.config.ClusterScaling.AutoscalingGroup, nodeIP, asgSess)
					if err != nil {
						logging.Error("core/runner: unable to successfully terminate AWS instance %v: %v", nodeID, err)
					}
				}
			}
		}
	}
	done <- true
	return
}

func (r *Runner) disableClusterScaling(clusterStatus *structs.ClusterStatus) (disabled bool) {
	// If we've reached the retry threshold, disable cluster scaling and
	// halt.
	if clusterStatus.NodeFailureCount == 3 {
		disabled = true
		r.config.ClusterScaling.Enabled = false

		logging.Error("core/runner: attempts to add new nodes to the "+
			"worker pool have failed %v times. Cluster scaling will be "+
			"disabled.", 3)
	}

	return
}

// jobScaling is the main entry point for the Nomad job scaling functionality
// and ties together a number of functions to be called from the runner.
func (r *Runner) jobScaling() {

	// Scaling a Cluster Jobs requires access to both Consul and Nomad therefore
	// we setup the clients here.
	consulClient := r.config.ConsulClient

	nomadClient := r.config.NomadClient

	// Determine if we are running on the leader node, halt if not.
	if haveLeadership := nomadClient.LeaderCheck(); !haveLeadership {
		logging.Debug("core/runner: replicator is not running on the known leader, no job scaling actions will be taken")
		return
	}

	// Pull the list of all currently running jobs which have a defined scaling
	// document. Fail quickly if we can't retrieve this list.
	resp, err := consulClient.GetJobScalingPolicies(r.config, nomadClient)
	if err != nil {
		logging.Error("core/runner: failed to determine if any jobs have scaling policies enabled \n%v", err)
		return
	}

	// EvaluateJobScaling identifies whether each of the Job.Groups requires a
	// scaling event to be triggered. This is then iterated so the individual
	// groups can be assesed.
	nomadClient.EvaluateJobScaling(resp)
	for _, job := range resp {

		// Due to the nested nature of the job and group Nomad definitions a dumb
		// metric is used to determine whether the job has 1 or more groups which
		// require scaling.
		i := 0

		for _, group := range job.GroupScalingPolicies {
			if group.Scaling.ScaleDirection == client.ScalingDirectionOut || group.Scaling.ScaleDirection == client.ScalingDirectionIn {
				if job.Enabled {
					logging.Debug("core/runner: scaling for job \"%v\" is enabled; a scaling operation (%v) will be requested for group \"%v\"",
						job.JobName, group.Scaling.ScaleDirection, group.GroupName)
					i++
				} else {
					logging.Debug("core/runner: scaling for job \"%v\" has been disabled; a scaling operation (%v) would have been requested for group \"%v\"",
						job.JobName, group.Scaling.ScaleDirection, group.GroupName)
				}
			}
		}

		// If 1 or more groups need to be scaled we submit the whole job for scaling
		// as to scale you must submit the whole job file currently. The JobScale
		// function takes care of scaling groups independently.
		if i > 0 {
			nomadClient.JobScale(job)
		}
	}
}
