package agent

import (
	"reflect"
	"testing"

	"github.com/elsevier-core-engineering/replicator/replicator/structs"
	"github.com/hashicorp/consul-template/test"
)

func TestConfigParse_LoadConfigFile(t *testing.T) {

	configFile := test.CreateTempfile([]byte(`
    consul           = "consul.com:8500"
    nomad            = "http://nomad.com:4646"
    log_level        = "info"
    scaling_interval = 1
    aws_region       = "us-east-1"

    cluster_scaling {
      enabled              = true
      max_size             = 1000
      min_size             = 700
      cool_down            = 800
      node_fault_tolerance = 50
      autoscaling_group    = "nomad"
    }

    job_scaling {
      enabled             = true
      consul_key_location = "wosniak/jobs"
      consul_token        = "thisisafaketoken"
    }

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
		Nomad:           "http://nomad.com:4646",
		LogLevel:        "info",
		ScalingInterval: 1,
		Region:          "us-east-1",

		ClusterScaling: &structs.ClusterScaling{
			Enabled:            true,
			MaxSize:            1000,
			MinSize:            700,
			CoolDown:           800,
			NodeFaultTolerance: 50,
			AutoscalingGroup:   "nomad",
		},

		JobScaling: &structs.JobScaling{
			Enabled:           true,
			ConsulKeyLocation: "wosniak/jobs",
			ConsulToken:       "thisisafaketoken",
		},

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
