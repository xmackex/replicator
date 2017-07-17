package replicator

import (
	"testing"

	"github.com/elsevier-core-engineering/replicator/client"
	"github.com/elsevier-core-engineering/replicator/replicator/structs"
	"github.com/hashicorp/consul/testutil"
)

func makeClientWithConfig(t *testing.T) (*structs.Config, *testutil.TestServer) {

	srv1 := testutil.NewTestServer(t)

	consul, _ := client.NewConsulClient(srv1.HTTPAddr, "")

	config := &structs.Config{
		ConsulClient:      consul,
		ConsulKeyLocation: "replicator/config",
		ClusterScaling:    &structs.ClusterScaling{},
		Notification:      &structs.Notification{},
	}

	return config, srv1
}
