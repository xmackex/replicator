package replicator

import (
	"github.com/elsevier-core-engineering/replicator/client"
	"github.com/elsevier-core-engineering/replicator/logging"
	"github.com/elsevier-core-engineering/replicator/replicator/structs"
)

func checkScalingThreshold(state *structs.State, direction string, clusterScaling *structs.ClusterScaling) (scale bool) {

	switch direction {
	case client.ScalingDirectionIn:
		state.ClusterScaleInRequests++
		state.ClusterScaleOutRequests = 0
		if state.ClusterScaleInRequests == clusterScaling.ScalingThreshold {
			state.ClusterScaleInRequests = 0
			logging.Debug("core/cluster_scaling: scale in requests %v has reached threshold %v",
				state.ClusterScaleInRequests, clusterScaling.ScalingThreshold)
			return true
		}

		logging.Debug("core/cluster_scaling: scale in requests %v has not been reached threshold %v",
			state.ClusterScaleInRequests, clusterScaling.ScalingThreshold)

	case client.ScalingDirectionOut:
		state.ClusterScaleOutRequests++
		state.ClusterScaleInRequests = 0
		if state.ClusterScaleOutRequests == clusterScaling.ScalingThreshold {
			state.ClusterScaleOutRequests = 0
			logging.Debug("core/cluster_scaling: scale out requests %v has reached threshold %v",
				state.ClusterScaleInRequests, clusterScaling.ScalingThreshold)
			return true
		}

		logging.Debug("core/cluster_scaling: scale out requests %v has not been reached threshold %v",
			state.ClusterScaleOutRequests, clusterScaling.ScalingThreshold)

	default:
		state.ClusterScaleInRequests = 0
		state.ClusterScaleOutRequests = 0
	}
	return
}
