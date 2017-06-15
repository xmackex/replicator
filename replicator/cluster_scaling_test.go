package replicator

import (
	"testing"

	"github.com/elsevier-core-engineering/replicator/client"
	"github.com/elsevier-core-engineering/replicator/replicator/structs"
)

func TestClusterScaling_scalingThreshold(t *testing.T) {

	cluster := &structs.ClusterScaling{}
	state := &structs.State{}
	cluster.ScalingThreshold = 3

	// Check ScaleOut scenarios.
	state.ClusterScaleOutRequests = 2
	if !checkScalingThreshold(state, client.ScalingDirectionOut, cluster) {
		t.Fatal("expected ClusterScaleOut to answer true but got false")
	}

	state.ClusterScaleOutRequests = 1
	if checkScalingThreshold(state, client.ScalingDirectionOut, cluster) {
		t.Fatal("expected ClusterScaleOut to answer false but got true")
	}

	// Check ScaleIn scenarios.
	state.ClusterScaleInRequests = 2
	if !checkScalingThreshold(state, client.ScalingDirectionIn, cluster) {
		t.Fatal("expected ClusterScaleIn to answer true but got false")
	}

	state.ClusterScaleInRequests = 1
	if checkScalingThreshold(state, client.ScalingDirectionIn, cluster) {
		t.Fatal("expected ClusterScaleIn to answer false but got true")
	}

	// Check the default return and state setting.
	if checkScalingThreshold(state, client.ScalingDirectionNone, cluster) {
		t.Fatal("expected ClusterScalingNone to answer false but got true")
	}

	if state.ClusterScaleInRequests != 0 || state.ClusterScaleOutRequests != 0 {
		t.Fatalf("expected state scale requests to be 0, got %v and %v",
			state.ClusterScaleInRequests, state.ClusterScaleOutRequests)
	}
}
