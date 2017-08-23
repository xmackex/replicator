package replicator

import (
	"testing"

	"github.com/elsevier-core-engineering/replicator/client"
	"github.com/elsevier-core-engineering/replicator/replicator/structs"
	"github.com/elsevier-core-engineering/replicator/testutil"
)

func TestClusterScaling_scalingThreshold(t *testing.T) {
	t.Parallel()

	c, s := testutil.MakeClientWithConfig(t)
	defer s.Stop()

	consul, _ := client.NewConsulClient(s.HTTPAddr, "")
	c.ConsulClient = consul

	state := &structs.ScalingState{}

	workerPool := &structs.WorkerPool{}
	workerPool.Name = "example-pool"
	workerPool.ScalingThreshold = 3

	state.StatePath = "replicator/config/state/nodes/" + workerPool.Name

	// Check ScaleOut scenarios.
	state.ScaleOutRequests = 2
	if !checkPoolScalingThreshold(state, client.ScalingDirectionOut, workerPool, c) {
		t.Fatal("expected ClusterScaleOut to answer true but got false")
	}

	state.ScaleOutRequests = 1
	if checkPoolScalingThreshold(state, client.ScalingDirectionOut, workerPool, c) {
		t.Fatal("expected ClusterScaleOut to answer false but got true")
	}

	// Check ScaleIn scenarios.
	state.ScaleInRequests = 2
	if !checkPoolScalingThreshold(state, client.ScalingDirectionIn, workerPool, c) {
		t.Fatal("expected ClusterScaleIn to answer true but got false")
	}

	state.ScaleInRequests = 1
	if checkPoolScalingThreshold(state, client.ScalingDirectionIn, workerPool, c) {
		t.Fatal("expected ClusterScaleIn to answer false but got true")
	}

	// Check the default return and state setting.
	if checkPoolScalingThreshold(state, client.ScalingDirectionNone, workerPool, c) {
		t.Fatal("expected ClusterScalingNone to answer false but got true")
	}

	if state.ScaleInRequests != 0 || state.ScaleOutRequests != 0 {
		t.Fatalf("expected state scale requests to be 0, got %v and %v",
			state.ScaleInRequests, state.ScaleOutRequests)
	}
}
