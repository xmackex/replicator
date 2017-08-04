package client

import (
	"github.com/elsevier-core-engineering/replicator/helper"
	"github.com/elsevier-core-engineering/replicator/logging"
	"github.com/elsevier-core-engineering/replicator/replicator/structs"
	nomad "github.com/hashicorp/nomad/api"
	"github.com/mitchellh/mapstructure"
)

// JobWatcher is the main entry point into Replicators process of reading and
// updating its JobScalingPolicies tracking.
func (c *nomadClient) JobWatcher(jobScalingPolicies *structs.JobScalingPolicies) {

	q := &nomad.QueryOptions{WaitIndex: 1, AllowStale: true}

	for {
		jobs, meta, err := c.nomad.Jobs().List(q)
		if err != nil {
			logging.Error("client/job_scaling_policies: unable to list Nomad jobs: %v", err)
			return
		}

		// If the LastIndex is not greater than our stored LastChangeIndex, we don't
		// need to do anything. On the initial run this will always result in a full
		// run as the LastChangeIndex is initialized to 0.
		if meta.LastIndex <= jobScalingPolicies.LastChangeIndex {
			continue
		}

		// Iterate jobs and find events that have changed since last run
		for _, job := range jobs {
			if job.ModifyIndex > jobScalingPolicies.LastChangeIndex {

				// Dpending on the status of the job, take different action on the scaling
				// policy struct.
				switch job.Status {
				case StateRunning:
					go c.jobScalingPolicyProcessor(job.Name, jobScalingPolicies)
				case StateDead:
					go removeJobScalingPolicy(job.Name, jobScalingPolicies)
				default:
					continue
				}
			}
		}

		// Persist the LastIndex into our scaling policy struct.
		logging.Debug("client/job_scaling_policies: updating LastChangeIndex from %v to %v",
			jobScalingPolicies.LastChangeIndex, meta.LastIndex)
		jobScalingPolicies.LastChangeIndex = meta.LastIndex
		q.WaitIndex = meta.LastIndex
	}
}

// jobScalingPolicyProcessor triggers an iteation of the job groups to determine
// their meta paramerters scaling policy status.
func (c *nomadClient) jobScalingPolicyProcessor(jobID string, scaling *structs.JobScalingPolicies) {

	jobInfo, _, err := c.nomad.Jobs().Info(jobID, &nomad.QueryOptions{})
	if err != nil {
		logging.Error("client/job_scaling_policies: unable to call Nomad job info: %v", err)
	}

	// It seems when a job is stopped Nomad notifies twice; once indicates the job
	// is in running state, the second time is that the job is dead. This check
	// is to catch that.
	if *jobInfo.Status != StateRunning {
		return
	}

	// Run the checkOrphanedGroup function.
	go checkOrphanedGroup(jobID, jobInfo.TaskGroups, scaling)

	for _, group := range jobInfo.TaskGroups {
		missedKeys := parseMeta(group.Meta)

		// If all 7 keys missed, then the job group does not have scaling enabled,
		// this is logged for operator clarity.
		if len(missedKeys) == 7 {
			logging.Debug("client/job_scaling_policies: job %s and group %v is not configured for autoscaling",
				jobID, *group.Name)
			go removeGroupScalingPolicy(jobID, *group.Name, scaling)
			continue
		}

		// If some keys missed, the operator has made an effort to enable job scaling
		// but potentially made a typo. This is logged as an error so operators can
		// see and quickly resolve these issues.
		if len(missedKeys) > 0 && len(missedKeys) < 7 {
			logging.Error("client/job_scaling_policies: job %s and group %v is missing meta scaling key(s): %v",
				jobID, *group.Name, missedKeys)
			continue
		}

		// If all keys were matched we update the job scaling policy struct with the
		// information.
		if len(missedKeys) == 0 {
			logging.Debug("client/job_scaling_policies: job %s and group %v has all meta required for autoscaling",
				jobID, *group.Name)
			go func() {
				err := updateScalingPolicy(jobID, *group.Name, group.Meta, scaling)
				if err != nil {
					logging.Error("client/job_scaling_policies: unable to update scaling policy for job %v and group %v: %v",
						jobID, group.Name, err)
				}
			}()
		}
	}
}

// updateScalingPolicy takes a JobGroups meta parameter and updates Replicators
// JobScaling entry if required.
func updateScalingPolicy(jobName, groupName string, groupMeta map[string]string,
	s *structs.JobScalingPolicies) (err error) {

	result := &structs.GroupScalingPolicy{}
	found := false

	// Make use of mapstructures WeaklyTypedInput and setup the decoder. We use
	// WeaklyTypedInput as all meta KVs are strings, whereas we need most of them
	// to differ from this.
	decodeConf := &mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           result,
	}
	decoder, err := mapstructure.NewDecoder(decodeConf)
	if err != nil {
		return
	}

	// Decode the meta and add the group name to the correct field as this is not
	// availble in the meta.
	if err = decoder.Decode(groupMeta); err != nil {
		return
	}

	result.GroupName = groupName
	s.Lock.Lock()

	// If the job already has an entry in the scaling policies, attempt to find
	// the group and overwrite with the new policy. If the job is found, but no
	// group policy is found we append the new group policy to the job.
	if val, ok := s.Policies[jobName]; ok {
		for i, group := range val {
			if group.GroupName == groupName && helper.JobGroupScalingPolicyDiff(result, val[i]) {
				found = true
				continue
			} else if group.GroupName == groupName && !helper.JobGroupScalingPolicyDiff(result, val[i]) {
				found = true
				val[i] = result
				logging.Info("client/job_scaling_policies: updated scaling policy for job %s and group %s",
					jobName, groupName)
			} else {
				s.Policies[jobName] = append(s.Policies[jobName], result)
				logging.Info("client/job_scaling_policies: added new scaling policy for job %s and group %s",
					jobName, groupName)
				found = true
			}
		}
	}

	// If the job and group have not been found, create a new entry for the job
	// and add the group policy.
	if !found {
		s.Policies[jobName] = append(s.Policies[jobName], result)
		logging.Info("client/job_scaling_policies: added new policy for job %s and group %s",
			jobName, groupName)
	}
	s.Lock.Unlock()
	return
}

// removeScalingPolicy will remove a particular JobGroups scaling policy and
// will also remove the Job entry from the map if there are no longer any Group
// policies assosiated to it. This is used for jobs which are still running.
func removeGroupScalingPolicy(jobName, groupName string, scaling *structs.JobScalingPolicies) {
	if val, ok := scaling.Policies[jobName]; ok {
		for i, group := range val {
			if group.GroupName == groupName {
				scaling.Lock.Lock()
				scaling.Policies[jobName] = append(scaling.Policies[jobName][:i], scaling.Policies[jobName][i+1:]...)
				scaling.Lock.Unlock()
				logging.Info("client/job_scaling_policies: removed policy for job %s and group %s",
					jobName, groupName)
			}
		}
		if len(scaling.Policies[jobName]) == 0 {
			scaling.Lock.Lock()
			delete(scaling.Policies, jobName)
			scaling.Lock.Unlock()
		}
	}
}

// removeJobScalingPolicy deletes the job entry within the the policies map.
func removeJobScalingPolicy(jobName string, scaling *structs.JobScalingPolicies) {
	if _, ok := scaling.Policies[jobName]; ok {
		scaling.Lock.Lock()
		delete(scaling.Policies, jobName)
		scaling.Lock.Unlock()
		logging.Info("client/job_scaling_policies: deleted job scaling entries for job %v", jobName)
	}
}

// checkOrphanedGroup checks whether a job has been updated and removed a group
// which has a scaling policy; thus removing the entry.
func checkOrphanedGroup(jobName string, groups []*nomad.TaskGroup, scaling *structs.JobScalingPolicies) {

	taskGroupNames := make([]string, 0)
	taskGroupPolicyNames := make([]string, 0)

	scaling.Lock.RLock()
	if val, ok := scaling.Policies[jobName]; ok {
		for _, g := range val {
			taskGroupPolicyNames = append(taskGroupPolicyNames, g.GroupName)
		}
	}

	for _, g := range groups {
		taskGroupNames = append(taskGroupNames, *g.Name)
	}

	for _, g := range taskGroupNames {
		for i, gp := range taskGroupPolicyNames {
			if g == gp {
				taskGroupPolicyNames = append(taskGroupPolicyNames[:i], taskGroupPolicyNames[i+1:]...)
			}
		}
	}
	scaling.Lock.RUnlock()

	for _, g := range taskGroupPolicyNames {
		removeGroupScalingPolicy(jobName, g, scaling)
	}
}

// parseMeta parses a Nomad JobGroup meta map to discover if replciator keys
// are present. A list of requiredKeys that were not found is returned so that
// full logging can be made about the autoscaling state of each JobGroup.
func parseMeta(meta map[string]string) []string {

	// These are our required key for Replicator
	requiredKeys := []string{
		"replicator_cooldown",
		"replicator_enabled",
		"replicator_min",
		"replicator_max",
		"replicator_scalein_mem",
		"replicator_scalein_cpu",
		"replicator_scaleout_mem",
		"replicator_scaleout_cpu",
	}

	// Iterate over the group meta and remove found keys from requiredKeys in order
	// to gain a view of the configuration.
	for key := range meta {
		for i, rKey := range requiredKeys {
			if key == rKey {
				requiredKeys = append(requiredKeys[:i], requiredKeys[i+1:]...)
			}
		}
	}
	return requiredKeys
}
