package structs

import (
	"reflect"
	"testing"
)

func TestStructs_Merge(t *testing.T) {
	c := &Config{
		Consul:            "localhost:8500",
		ConsulKeyLocation: "replicator/config",
		Nomad:             "http://localhost:4646",
		LogLevel:          "INFO",
		ScalingInterval:   10,

		ClusterScaling: &ClusterScaling{
			MaxSize:            10,
			MinSize:            5,
			CoolDown:           600,
			NodeFaultTolerance: 1,
			RetryThreshold:     2,
			ScalingThreshold:   3,
		},

		JobScaling: &JobScaling{},

		Telemetry:    &Telemetry{},
		Notification: &Notification{},
	}

	partialConfig := &Config{
		Consul:          "consul.ce.systems",
		ConsulToken:     "afb3bc3a-6acd-11e7-b70c-784f43a63381",
		Nomad:           "http://nomad.ce.systems:4646",
		LogLevel:        "ERROR",
		ScalingInterval: 20,
		Region:          "eu-west-2",
		ClusterScaling: &ClusterScaling{
			Enabled:          true,
			MaxSize:          1000,
			MinSize:          750,
			AutoscalingGroup: "nomad-ce-dev",
		},
		JobScaling: &JobScaling{
			Enabled: true,
		},
		Telemetry: &Telemetry{
			StatsdAddress: "8.8.8.8:8125",
		},
		Notification: &Notification{
			ClusterScalingUID:   "CE1",
			ClusterIdentifier:   "core-engineering",
			PagerDutyServiceKey: "onlyopsoncall",
		},
	}

	fullConfig := &Config{
		Consul:            "consul.ce.systems",
		ConsulKeyLocation: "jobs/woz",
		ConsulToken:       "afb3bc3a-6acd-11e7-b70c-784f43a63381",
		Nomad:             "http://nomad.ce.systems:4646",
		LogLevel:          "ERROR",
		ScalingInterval:   20,
		Region:            "eu-west-2",
		ClusterScaling: &ClusterScaling{
			Enabled:            true,
			MaxSize:            1000,
			MinSize:            750,
			AutoscalingGroup:   "nomad-ce-dev",
			CoolDown:           1000,
			NodeFaultTolerance: 100,
			RetryThreshold:     10,
			ScalingThreshold:   20,
		},
		JobScaling: &JobScaling{
			Enabled: true,
		},
		Telemetry: &Telemetry{
			StatsdAddress: "8.8.8.8:8125",
		},
		Notification: &Notification{
			ClusterScalingUID:   "CE1",
			ClusterIdentifier:   "core-engineering",
			PagerDutyServiceKey: "onlyopsoncall",
		},
	}

	partialExpected := &Config{
		Consul:            "consul.ce.systems",
		ConsulKeyLocation: "replicator/config",
		ConsulToken:       "afb3bc3a-6acd-11e7-b70c-784f43a63381",
		Nomad:             "http://nomad.ce.systems:4646",
		LogLevel:          "ERROR",
		ScalingInterval:   20,
		Region:            "eu-west-2",
		ClusterScaling: &ClusterScaling{
			Enabled:            true,
			MaxSize:            1000,
			MinSize:            750,
			AutoscalingGroup:   "nomad-ce-dev",
			CoolDown:           600,
			NodeFaultTolerance: 1,
			RetryThreshold:     2,
			ScalingThreshold:   3,
		},
		JobScaling: &JobScaling{
			Enabled: true,
		},
		Telemetry: &Telemetry{
			StatsdAddress: "8.8.8.8:8125",
		},
		Notification: &Notification{
			ClusterScalingUID:   "CE1",
			ClusterIdentifier:   "core-engineering",
			PagerDutyServiceKey: "onlyopsoncall",
		},
	}

	fullExpected := &Config{
		Consul:            "consul.ce.systems",
		ConsulKeyLocation: "jobs/woz",
		ConsulToken:       "afb3bc3a-6acd-11e7-b70c-784f43a63381",
		Nomad:             "http://nomad.ce.systems:4646",
		LogLevel:          "ERROR",
		ScalingInterval:   20,
		Region:            "eu-west-2",
		ClusterScaling: &ClusterScaling{
			Enabled:            true,
			MaxSize:            1000,
			MinSize:            750,
			AutoscalingGroup:   "nomad-ce-dev",
			CoolDown:           1000,
			NodeFaultTolerance: 100,
			RetryThreshold:     10,
			ScalingThreshold:   20,
		},
		JobScaling: &JobScaling{
			Enabled: true,
		},
		Telemetry: &Telemetry{
			StatsdAddress: "8.8.8.8:8125",
		},
		Notification: &Notification{
			ClusterScalingUID:   "CE1",
			ClusterIdentifier:   "core-engineering",
			PagerDutyServiceKey: "onlyopsoncall",
		},
	}

	partialResult := c.Merge(partialConfig)
	fullResult := c.Merge(fullConfig)

	if !reflect.DeepEqual(partialResult, partialExpected) {
		t.Fatalf("expected \n%#v\n\n, got \n\n%#v\n\n", partialExpected, partialResult)
	}
	if !reflect.DeepEqual(fullResult, fullExpected) {
		t.Fatalf("expected \n%#v\n\n, got \n\n%#v\n\n", fullExpected, fullResult)
	}
}
