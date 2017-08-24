package base

import (
	"reflect"
	"testing"

	"github.com/elsevier-core-engineering/replicator/replicator/structs"
	"github.com/hashicorp/consul-template/test"
)

func TestConfigParse_LoadConfigFile(t *testing.T) {

	configFile := test.CreateTempfile([]byte(`
    consul                 = "consul.com:8500"
		consul_key_root        = "wosniak/jobs"
		consul_token           = "thisisafaketoken"
    nomad                  = "http://nomad.com:4646"
    log_level              = "info"
    scaling_interval       = 1

    telemetry {
      statsd_address = "10.0.0.10:8125"
    }

    notification {
      pagerduty_service_key = "thistooisafakekey"
      cluster_identifier    = "nomad-prod"
      cluster_scaling_uid   = "Nomad1"
    }

  `), t)
	defer test.DeleteTempfile(configFile, t)

	c, err := LoadConfig(configFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := &structs.Config{
		Consul:          "consul.com:8500",
		ConsulKeyRoot:   "wosniak/jobs",
		ConsulToken:     "thisisafaketoken",
		Nomad:           "http://nomad.com:4646",
		LogLevel:        "info",
		ScalingInterval: 1,

		Telemetry: &structs.Telemetry{
			StatsdAddress: "10.0.0.10:8125",
		},

		Notification: &structs.Notification{
			PagerDutyServiceKey: "thistooisafakekey",
			ClusterIdentifier:   "nomad-prod",
			ClusterScalingUID:   "Nomad1",
		},
	}
	if !reflect.DeepEqual(c, expected) {
		t.Fatalf("expected \n%#v\n\n, got \n\n%#v\n\n", expected, c)
	}
}
