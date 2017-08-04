package client

import (
	"time"

	metrics "github.com/armon/go-metrics"
	"github.com/elsevier-core-engineering/replicator/logging"
	"github.com/elsevier-core-engineering/replicator/replicator/structs"
	nomad "github.com/hashicorp/nomad/api"
)

// JobScale takes a Scaling Policy and then attempts to scale the desired job
// to the appropriate level whilst ensuring the event will not excede any job
// thresholds set.
func (c *nomadClient) JobScale(jobName string, jobScalingPolicies []*structs.GroupScalingPolicy) {

	// In order to scale the job, we need information on the current status of the
	// running job from Nomad.
	jobResp, _, err := c.nomad.Jobs().Info(jobName, &nomad.QueryOptions{})

	if err != nil {
		logging.Error("client/job_scaling: unable to determine job info of %v: %v", jobName, err)
		return
	}

	// Use the current task count in order to determine whether or not a scaling
	// event will violate the min/max job policy.
	for _, group := range jobScalingPolicies {

		if group.ScaleDirection == ScalingDirectionNone {
			continue
		}

		for i, taskGroup := range jobResp.TaskGroups {
			if group.ScaleDirection == ScalingDirectionOut && *taskGroup.Count >= group.Max ||
				group.ScaleDirection == ScalingDirectionIn && *taskGroup.Count <= group.Min {
				logging.Debug("client/job_scaling: scale %v not permitted due to constraints on job \"%v\" and group \"%v\"",
					group.ScaleDirection, *jobResp.ID, group.GroupName)
				return
			}

			logging.Info("client/job_scaling: scale %v will now be initiated against job \"%v\" and group \"%v\"",
				group.ScaleDirection, jobName, group.GroupName)

			// Depending on the scaling direction decrement/incrament the count;
			// currently replicator only supports addition/subtraction of 1.
			if *taskGroup.Name == group.GroupName && group.ScaleDirection == ScalingDirectionOut {
				metrics.IncrCounter([]string{"job", jobName, group.GroupName, "scale_out"}, 1)
				*jobResp.TaskGroups[i].Count++
				group.LastScalingEvent = time.Now()
			}

			if *taskGroup.Name == group.GroupName && group.ScaleDirection == ScalingDirectionIn {
				metrics.IncrCounter([]string{"job", jobName, group.GroupName, "scale_in"}, 1)
				*jobResp.TaskGroups[i].Count--
				group.LastScalingEvent = time.Now()
			}
		}
	}

	// Submit the job to the Register API endpoint with the altered count number
	// and check that no error is returned.
	if _, _, err = c.nomad.Jobs().Register(jobResp, &nomad.WriteOptions{}); err != nil {
		logging.Error("client/job_scaling: issue submitting job %s for scaling action: %v",
			jobName, err)
		return
	}

	logging.Info("client/job_scaling: scaling action successfully taken against job \"%v\"", *jobResp.ID)
	return
}
