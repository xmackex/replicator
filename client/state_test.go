package client

import (
	"reflect"
	"testing"
	"time"

	"github.com/elsevier-core-engineering/replicator/replicator/structs"
	"github.com/elsevier-core-engineering/replicator/testutil"
)

func TestClient_StateTracking(t *testing.T) {
	t.Parallel()

	c, s := testutil.MakeClientWithConfig(t)
	defer s.Stop()

	consul, _ := NewConsulClient(s.HTTPAddr, "")
	c.ConsulClient = consul

	timeNow := time.Now()
	path := c.ConsulKeyLocation + "/state/nodes/example-pool"
	returnState := &structs.ScalingState{}

	expected := &structs.ScalingState{
		FailsafeMode:     false,
		FailureCount:     0,
		LastScalingEvent: timeNow,
	}

	err := c.ConsulClient.PersistState(path, expected)
	if err != nil {
		t.Fatalf("error writing state to Consul: %v", err)
	}

	c.ConsulClient.ReadState(path, returnState)

	if !reflect.DeepEqual(returnState, expected) {
		t.Fatalf("expected \n%#v\n\n, got \n\n%#v\n\n", expected, returnState)
	}

}
