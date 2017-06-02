package replicator

import (
	"time"

	metrics "github.com/armon/go-metrics"
	"github.com/elsevier-core-engineering/replicator/client"
	"github.com/elsevier-core-engineering/replicator/logging"
	"github.com/elsevier-core-engineering/replicator/replicator/structs"
)

// Runner is the main runner struct.
type Runner struct {
	// doneChan is where finish notifications occur.
	doneChan chan bool

	// config is the Config that created this Runner. It is used internally to
	// construct other objects and pass data.
	config *structs.Config

	// candaidate is our LeaderCandidate for the runner instance.
	candidate *LeaderCandidate
}

// NewRunner sets up the Runner type.
func NewRunner(config *structs.Config) (*Runner, error) {
	runner := &Runner{
		doneChan: make(chan bool),
		config:   config,
	}
	return runner, nil
}

// Start creates a new runner and uses a ticker to block until the doneChan is
// closed at which point the ticker is stopped.
func (r *Runner) Start() {
	ticker := time.NewTicker(time.Second * time.Duration(r.config.ScalingInterval))

	// Initialize the state tracking object for scaling operations.
	scalingState := &structs.ScalingState{}

	// Setup our LeaderCandidate object for leader elections and session renewl.
	leaderKey := r.config.ConsulKeyLocation + "/" + "leader"
	r.candidate = newLeaderCandidate(r.config.ConsulClient, leaderKey, r.config.ScalingInterval)

	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:

			// Perform the leadership locking and continue if we have confirmed that
			// we are running as the replicator leader.
			r.candidate.leaderElection()
			if r.candidate.isLeader() {

				// ClusterScaling blocks Replicator when it runs, we do not want job
				// scaling to be invoked if we are moving workloads around or adding
				// new capacity.
				clusterChan := make(chan bool)
				go r.clusterScaling(clusterChan, scalingState)
				<-clusterChan

				r.jobScaling()
			}

		case <-r.doneChan:
			return
		}
	}
}

// Stop halts the execution of this runner.
func (r *Runner) Stop() {
	r.candidate.endCampaign()
	r.doneChan <- true
	close(r.doneChan)
}

// clusterScaling is the main entry point into the cluster scaling functionality
// and ties numerous functions together to create an asynchronus function which
// can be called from the runner.
func (r *Runner) clusterScaling(done chan bool, scalingState *structs.ScalingState) {
	// Initialize clients for Nomad and Consul.
	nomadClient := r.config.NomadClient
	consulClient := r.config.ConsulClient

	// Attempt to load state tracking information from the distributed key/value
	// store and reset permitted flag.
	scalingState = consulClient.LoadState(r.config, scalingState)

	// Retrieve value of cluster scaling enabled flag.
	scalingEnabled := r.config.ClusterScaling.Enabled

	// If a region has not been specified, attempt to dynamically determine what
	// region we are running in.
	if r.config.Region == "" {
		if region, err := client.DescribeAWSRegion(); err == nil {
			r.config.Region = region
		}
	}

	// Initialize a new disposable cluster capacity object.
	clusterCapacity := &structs.ClusterCapacity{}

	//if scale, err := nomadClient.EvaluateClusterCapacity(clusterCapacity, r.config); err != nil || !scale {
	scale, err := nomadClient.EvaluateClusterCapacity(clusterCapacity, r.config)
	if err != nil || !scale {
		logging.Debug("core/runner: scaling operation not required or permitted")

		done <- true
		return
	}

	// If we reached this point we will be performing AWS interaction so we
	// create an client connection.
	asgSess := client.NewAWSAsgService(r.config.Region)

	// Calculate the scaling cooldown threshold.
	if !scalingState.LastScalingEvent.IsZero() {
		cooldown := scalingState.LastScalingEvent.Add(
			time.Duration(r.config.ClusterScaling.CoolDown) * time.Second)

		if time.Now().Before(cooldown) {
			logging.Info("core/runner: cluster scaling cooldown threshold has "+
				"not been reached: %v, scaling operations will not be permitted",
				cooldown)

			done <- true
			return
		}

		logging.Debug("core/runner: cluster scaling cooldown threshold %v has "+
			"been reached, scaling operations will be permitted", cooldown)
	} else {
		logging.Info("core/runner: no previous scaling operations have " +
			"occurred, scaling operations will be permitted.")
	}

	if clusterCapacity.ScalingDirection == client.ScalingDirectionOut {
		// If cluster scaling has been disabled, report but do not initiate a
		// scaling operation.
		if !scalingEnabled {
			logging.Debug("core/runner: cluster scaling disabled, a scaling " +
				"operation [scale-out] will not be initiated")

			done <- true
			return
		}

		// Attempt to increment the desired count of the autoscaling group. If
		// this fails, log an error and stop further processing.
		err := client.ScaleOutCluster(r.config.ClusterScaling.AutoscalingGroup, asgSess)
		if err != nil {
			logging.Error("core/runner: unable to successfully initiate a "+
				"scaling operation against autoscaling group %v: %v",
				r.config.ClusterScaling.AutoscalingGroup, err)
			done <- true
			return
		}

		// Attempt to add a new node to the worker pool until we reach the
		// retry threshold.
		for scalingState.NodeFailureCount <= r.config.ClusterScaling.RetryThreshold {
			if scalingState.NodeFailureCount > 0 {
				logging.Info("core/runner: attempting to launch a new worker node, "+
					"previous node failures: %v", scalingState.NodeFailureCount)
			}

			// We've verified the autoscaling group operation completed
			// successfully. Next we'll identify the most recently launched EC2
			// instance from the worker pool ASG.
			newestNode, err := client.GetMostRecentInstance(
				r.config.ClusterScaling.AutoscalingGroup,
				r.config.Region,
			)
			if err != nil {
				logging.Error("core/runner: Failed to identify the most recently "+
					"launched instance: %v", err)
				scalingState.NodeFailureCount++
				scalingState.LastNodeFailure = time.Now()

				// Attempt to update state tracking information in Consul.
				err = consulClient.WriteState(r.config, scalingState)
				if err != nil {
					logging.Error("core/runner: %v", err)
				}
				continue
			}

			// Attempt to verify the new worker node has completed bootstrapping
			// and successfully joined the worker pool.
			if healthy := nomadClient.VerifyNodeHealth(newestNode); healthy {
				// Reset node failure count once we have a verified healthy worker.
				scalingState.NodeFailureCount = 0

				// Update the last scaling event timestamp.
				scalingState.LastScalingEvent = time.Now()

				// Attempt to update state tracking information in Consul.
				err = consulClient.WriteState(r.config, scalingState)
				if err != nil {
					logging.Error("core/runner: %v", err)
				}

				done <- true
				return
			}

			scalingState.NodeFailureCount++
			scalingState.LastNodeFailure = time.Now()
			logging.Error("core/runner: new node %v failed to successfully join "+
				"the worker pool, incrementing node failure count to %v and "+
				"terminating instance", newestNode, scalingState.NodeFailureCount)

			// Attempt to update state tracking information in Consul.
			err = consulClient.WriteState(r.config, scalingState)
			if err != nil {
				logging.Error("core/runner: %v", err)
			}

			metrics.IncrCounter([]string{"cluster", "scale_out_failed"}, 1)

			// Translate the IP address of the most recent instance to the EC2
			// instance ID.
			instanceID := client.TranslateIptoID(newestNode, r.config.Region)

			// If we've reached the retry threshold, disable cluster scaling and
			// halt.
			if disabled := r.disableClusterScaling(scalingState); disabled {
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
		}
	}

	if clusterCapacity.ScalingDirection == client.ScalingDirectionIn {
		// Attempt to identify the least-allocated node in the worker pool.
		nodeID, nodeIP := nomadClient.LeastAllocatedNode(clusterCapacity)
		if nodeIP != "" && nodeID != "" {
			if !scalingEnabled {
				logging.Debug("core/runner: cluster scaling disabled, not " +
					"initiating scaling operation (scale-in)")
				done <- true
				return
			}

			// Attempt to place the least-allocated node in drain mode.
			if err := nomadClient.DrainNode(nodeID); err != nil {
				logging.Error("core/runner: an error occurred while attempting to "+
					"place node %v in drain mode: %v", nodeID, err)
				done <- true
				return
			}

			logging.Info("core/runner: terminating AWS instance %v", nodeIP)
			err := client.ScaleInCluster(
				r.config.ClusterScaling.AutoscalingGroup, nodeIP, asgSess)
			if err != nil {
				logging.Error("core/runner: unable to successfully terminate AWS "+
					"instance %v: %v", nodeID, err)
				done <- true
				return
			}

			// Update the last scaling event timestamp.
			scalingState.LastScalingEvent = time.Now()

			// Attempt to update state tracking information in Consul.
			err = consulClient.WriteState(r.config, scalingState)
			if err != nil {
				logging.Error("core/runner: %v", err)
			}
		}
	}
	done <- true
	metrics.IncrCounter([]string{"cluster", "scale_out_success"}, 1)
	return
}

func (r *Runner) disableClusterScaling(scalingState *structs.ScalingState) (disabled bool) {
	// If we've reached the retry threshold, disable cluster scaling and
	// halt.
	if scalingState.NodeFailureCount == r.config.ClusterScaling.RetryThreshold {
		disabled = true
		r.config.ClusterScaling.Enabled = false

		logging.Error("core/runner: attempts to add new nodes to the "+
			"worker pool have failed %v times. Cluster scaling will be "+
			"disabled.", r.config.ClusterScaling.RetryThreshold)
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

	// Pull the list of all currently running jobs which have a defined scaling
	// document. Fail quickly if we can't retrieve this list.
	resp, err := consulClient.GetJobScalingPolicies(r.config, nomadClient)
	if err != nil {
		logging.Error("core/runner: failed to determine if any jobs have scaling "+
			"policies enabled \n%v", err)
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
				if job.Enabled && r.config.JobScaling.Enabled {
					logging.Debug("core/runner: scaling for job \"%v\" is enabled; a "+
						"scaling operation (%v) will be requested for group \"%v\"",
						job.JobName, group.Scaling.ScaleDirection, group.GroupName)
					i++
				} else {
					logging.Debug("core/runner: job scaling has been disabled; a "+
						"scaling operation (%v) would have been requested for \"%v\" "+
						"and group \"%v\"", group.Scaling.ScaleDirection, job.JobName,
						group.GroupName)
				}
			}
		}

		// If 1 or more groups need to be scaled we submit the whole job for
		// scaling as to scale you must submit the whole job file currently. The
		// JobScale function takes care of scaling groups independently.
		if i > 0 {
			nomadClient.JobScale(job)
		}
	}
}
