package structs

import "sync"

// JobScalingPolicies tracks replicators view of Job scaling policies and states
// with a Lock to safe guard read/write/deletes to the Policies map.
type JobScalingPolicies struct {
	LastChangeIndex uint64
	Lock            sync.RWMutex
	Policies        map[string][]*GroupScalingPolicy
}

// GroupScalingPolicy represents all the information needed to make JobTaskGroup
// scaling decisions.
type GroupScalingPolicy struct {
	Cooldown       int  `mapstructure:"replicator-cooldown"`
	Enabled        bool `mapstructure:"replicator-enabled"`
	GroupName      string
	Max            int            `mapstructure:"replicator-max"`
	Min            int            `mapstructure:"replicator-min"`
	ScaleDirection string         `hash:"ignore"`
	ScaleInCPU     float64        `mapstructure:"replicator-scalein-cpu"`
	ScaleInMem     float64        `mapstructure:"replicator-scalein-mem"`
	ScalingMetric  string         `hash:"ignore"`
	ScaleOutCPU    float64        `mapstructure:"replicator-scaleout-cpu"`
	ScaleOutMem    float64        `mapstructure:"replicator-scaleout-mem"`
	Tasks          TaskAllocation `hash:"ignore"`
}
