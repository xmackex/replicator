package replicator

import (
	"os"
	"time"

	"github.com/elsevier-core-engineering/replicator/logging"
	"github.com/elsevier-core-engineering/replicator/replicator/structs"
)

// Runner is the main runner struct.
type Runner struct {
	// candaidate is our LeaderCandidate for the runner instance.
	candidate *LeaderCandidate

	// config is the Config that created this Runner. It is used internally to
	// construct other objects and pass data.
	config *structs.Config

	// doneChan is where finish notifications occur.
	doneChan chan bool
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

	// Setup our LeaderCandidate object for leader elections and session renewl.
	leaderKey := r.config.ConsulKeyRoot + "/" + "leader"
	r.candidate = newLeaderCandidate(r.config.ConsulClient, leaderKey,
		r.config.ScalingInterval)

	defer ticker.Stop()

	// Setup our JobScalingPolicy Watcher and start running this.
	jobScalingPolicy := newJobScalingPolicy()
	go r.config.NomadClient.JobWatcher(jobScalingPolicy)

	// Setup the node registry and initiate worker pool and node discovery.
	NodeRegistry := newNodeRegistry()
	go r.config.NomadClient.NodeWatcher(NodeRegistry)

	for {
		select {
		case <-ticker.C:
			// Perform the leadership locking and continue if we have confirmed that
			// we are running as the replicator leader.
			r.candidate.leaderElection()
			if r.candidate.isLeader() {
				// If we're running as a Nomad job, perform a reverse lookup to
				// identify the node on which we're running and register it as
				// protected.
				if allocID := os.Getenv("NOMAD_ALLOC_ID"); len(allocID) > 0 {
					host, err := r.config.NomadClient.NodeReverseLookup(allocID)
					if err != nil {
						logging.Error("core/runner: Running as a Nomad job but unable "+
							"to determine the ID of our host node: %v", err)
					}

					if len(host) > 0 {
						NodeRegistry.Lock.Lock()
						// Reverse lookup our worker pool name by node ID and register
						// the node as protected.
						if pool, ok := NodeRegistry.RegisteredNodes[host]; ok {
							logging.Debug("core/runner: registering node %v in worker "+
								"pool %v as protected", host, pool)
							NodeRegistry.WorkerPools[pool].ProtectedNode = host
						}
						NodeRegistry.Lock.Unlock()
					}
				}

				if !r.config.ClusterScalingDisable {
					// Initiate cluster scaling for each known scaleable worker pool.
					r.asyncClusterScaling(NodeRegistry, jobScalingPolicy)
				}

				if !r.config.JobScalingDisable {
					// Initiate job scaling for each known scaleable job.
					r.jobScaling(jobScalingPolicy)
				}
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
