package replicator

import (
	"sync"

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

	for job, groups := range jobScalingPolicies.Policies {

		// Launch a routine for each job so that we can concurrantly run job scaling
		// functions as much as a map will allow.
		go func(job string, groups []*structs.GroupScalingPolicy) {

			// EvaluateJobScaling performs read/write to our map therefore we wrap it
			// in a read/write lock and remove this as soon as possible as the
			// remianing functions only need a read lock.
			jobScalingPolicies.Lock.Lock()
			nomadClient.EvaluateJobScaling(job, groups)
			jobScalingPolicies.Lock.Unlock()

			// Due to the nested nature of the job and group Nomad definitions a dumb
			// metric is used to determine whether the job has 1 or more groups which
			// require scaling.
			i := 0

			jobScalingPolicies.Lock.RLock()
			for _, group := range groups {
				if group.ScaleDirection == client.ScalingDirectionOut || group.ScaleDirection == client.ScalingDirectionIn {
					if group.Enabled && r.config.JobScaling.Enabled {
						logging.Debug("core/job_scaling: scaling for job \"%v\" is enabled; a "+
							"scaling operation (%v) will be requested for group \"%v\"",
							job, group.ScaleDirection, group.GroupName)
						i++
					} else {
						logging.Debug("core/job_scaling: job scaling has been disabled; a "+
							"scaling operation (%v) would have been requested for \"%v\" "+
							"and group \"%v\"", group.ScaleDirection, job,
							group.GroupName)
					}
				}
			}

			// If 1 or more groups need to be scaled we submit the whole job for
			// scaling as to scale you must submit the whole job file currently. The
			// JobScale function takes care of scaling groups independently.
			if i > 0 {
				nomadClient.JobScale(job, groups)
			}
			jobScalingPolicies.Lock.RUnlock()
		}(job, groups)
	}
}
