package replicator

// import (
// 	"testing"
//
// 	"github.com/elsevier-core-engineering/replicator/client"
// 	"github.com/elsevier-core-engineering/replicator/replicator/structs"
// 	"github.com/elsevier-core-engineering/replicator/testutil"
// )
//
// func TestFailsafe_FaileSafeCheck(t *testing.T) {
// 	t.Parallel()
//
// 	c, s := testutil.MakeClientWithConfig(t)
// 	defer s.Stop()
//
// 	consul, _ := client.NewConsulClient(s.HTTPAddr, "")
// 	c.ConsulClient = consul
//
// 	state := &structs.State{}
//
// 	// Test circuit breaker.
// 	state.FailsafeMode = true
// 	if FailsafeCheck(state, c) {
// 		t.Fatal("expected FailsafeMode to answer false but got true")
// 	}
//
// 	// Test failsafe threshold not met.
// 	state.FailsafeMode = false
// 	state.NodeFailureCount = 1
// 	c.ClusterScaling.RetryThreshold = 3
//
// 	if !FailsafeCheck(state, c) {
// 		t.Fatal("expected FailsafeMode to answer true but got false")
// 	}
//
// 	// Test failsafe threshold not met.
// 	state.FailsafeMode = false
// 	state.NodeFailureCount = 3
// 	c.ClusterScaling.RetryThreshold = 3
//
// 	if FailsafeCheck(state, c) {
// 		t.Fatal("expected FailsafeMode to answer false but got true")
// 	}
// }
//
// func TestFailsafe_SetFailsafeMode(t *testing.T) {
// 	t.Parallel()
//
// 	c, s := testutil.MakeClientWithConfig(t)
// 	defer s.Stop()
//
// 	consul, _ := client.NewConsulClient(s.HTTPAddr, "")
// 	c.ConsulClient = consul
//
// 	state := &structs.State{}
//
// 	// Test enabled false.
// 	enabled := false
// 	SetFailsafeMode(state, c, enabled)
//
// 	if state.FailsafeMode != enabled {
// 		t.Fatalf("expected FailsafeMode to be %v but got %v", enabled, state.FailsafeMode)
// 	}
// }
