package replicator

import (
	"fmt"
	"sync"
	"time"

	"github.com/elsevier-core-engineering/replicator/client"
	"github.com/elsevier-core-engineering/replicator/logging"
	"github.com/elsevier-core-engineering/replicator/replicator/structs"
)

// newJobScalingPolicy returns a new JobScalingPolicies struct for Replicator
// to track and view Nomad job scaling.
func newJobScalingPolicy() *structs.JobScalingPolicies {

	return &structs.JobScalingPolicies{
		Policies: make(map[string][]*structs.GroupScalingPolicy),
		Lock:     sync.RWMutex{},
	}
}

// jobScaling is the main entry point for the Nomad job scaling functionality
// and ties together a number of functions to be called from the runner.
func (r *Runner) jobScaling(jobScalingPolicies *structs.JobScalingPolicies) {

	// Scaling a Cluster Jobs requires access to both Consul and Nomad therefore
	// we setup the clients here.
	nomadClient := r.config.NomadClient
	consulClient := r.config.ConsulClient

	for job, groups := range jobScalingPolicies.Policies {

		if nomadClient.IsJobInDeployment(job) {
			logging.Debug("core/job_scaling: job %s is in deployment, no scaling evaluation will be triggerd", job)
			continue
		}

		// Launch a routine for each job so that we can concurrantly run job scaling
		// functions as much as a map will allow.
		go func(job string, groups []*structs.GroupScalingPolicy) {

			// EvaluateJobScaling performs read/write to our map therefore we wrap it
			// in a read/write lock and remove this as soon as possible as the
			// remianing functions only need a read lock.
			jobScalingPolicies.Lock.Lock()
			nomadClient.EvaluateJobScaling(job, groups)
			jobScalingPolicies.Lock.Unlock()

			jobScalingPolicies.Lock.RLock()
			for _, group := range groups {

				// Read or JobGroup state and check failsafe.
				p := r.config.ConsulKeyLocation + "/state/jobs/" + job + "/" + group.GroupName
				s := &structs.ScalingState{}
				consulClient.ReadState(p, s)

				if s.FailsafeMode {
					logging.Error("core/job_scaling: job \"%v\" and group \"%v\" is in failsafe mode", job, group.GroupName)
					continue
				}

				// Check the JobGroup scaling cooldown.
				cd := time.Duration(group.Cooldown) * time.Second

				if !s.LastScalingEvent.Before(time.Now().Add(-cd)) {
					logging.Debug("core/job_scaling: job \"%v\" and group \"%v\" has not reached scaling cooldown threshold of %s",
						job, group.GroupName, cd)
					continue
				}

				if group.ScaleDirection == client.ScalingDirectionOut || group.ScaleDirection == client.ScalingDirectionIn {
					if group.Enabled && r.config.JobScaling.Enabled {
						logging.Debug("core/job_scaling: scaling for job \"%v\" and group \"%v\" is enabled; a "+
							"scaling operation (%v) will be requested", job, group.GroupName, group.ScaleDirection)

						// Submit the job and group for scaling.
						nomadClient.JobGroupScale(job, group, s)

					} else {
						logging.Debug("core/job_scaling: job scaling has been disabled; a "+
							"scaling operation (%v) would have been requested for \"%v\" "+
							"and group \"%v\"", group.ScaleDirection, job, group.GroupName)
					}
				}

				if s.FailsafeMode {
					sendFailsafeNotification(fmt.Sprintf("%s_%s", job, group.GroupName), jobType, group.UID, s, r.config)
				}

				// Persist our state to Consul.
				consulClient.PersistState(p, s)

			}
			jobScalingPolicies.Lock.RUnlock()
		}(job, groups)
	}
}
