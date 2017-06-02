package structs

// JobScalingPolicies is a list of ScalingPolicy objects.
type JobScalingPolicies []*JobScalingPolicy

// JobScalingPolicy is a struct which represents an individual job scaling policy
// document.
type JobScalingPolicy struct {
	// JobName is the name of the Nomad job represented by the Consul Key/Value.
	JobName string

	// Enabled is a boolean falg which dictates whether scaling events for the job
	// should be enforced and is used for testing purposes.
	Enabled bool `json:"enabled"`

	// GroupScalingPolicies is a list of GroupScalingPolicy objects.
	GroupScalingPolicies []*GroupScalingPolicy `json:"groups"`
}

// GroupScalingPolicy represents the scaling policy of an individual group within
// a signle job.
type GroupScalingPolicy struct {
	// GroupName is the jobs Group name which this scaling policy represents.
	GroupName string `json:"name"`

	// TaskResources is a list
	Tasks TaskAllocation `json:"task_resources"`

	// ScalingMetric represents the most-utilized resource within the task group.
	ScalingMetric string

	// Scaling is a list of Scaling objects.
	Scaling *Scaling
}

// Scaling struct represents the scaling policy of a Nomad Job Group as well as
// details of any scaling activities which should take place during the current
// deamon run.
type Scaling struct {
	// Min in the minimum number of tasks the job should have running at any one
	// time.
	Min int `json:"min"`

	// Max in the maximum number of tasks the job should have running at any one
	// time.
	Max int `json:"max"`

	// ScaleDirection is populated by either out/in/none depending on the evalution
	// of a scaling event happening.
	ScaleDirection string

	// ScaleOut is the job scaling out policy which will contain the thresholds
	// which control scaling activies.
	ScaleOut *scaleout `json:"scaleout"`

	// ScaleIn is the job scaling in policy which will contain the thresholds
	// which control scaling activies.
	ScaleIn *scalein `json:"scalein"`
}

type scaleout struct {
	CPU float64 `json:"cpu"`
	MEM float64 `json:"mem"`
}

type scalein struct {
	CPU float64 `json:"cpu"`
	MEM float64 `json:"mem"`
}

// The ConsulClient interface is used to provide common method signatures for
// interacting with the Consul API.
type ConsulClient interface {
	// GetJobScalingPolicies provides a list of Nomad jobs with a defined scaling
	// policy document at a specified Consuk Key/Value Store location. Supports
	// the use of an ACL token if required by the Consul cluster.
	GetJobScalingPolicies(*Config, NomadClient) ([]*JobScalingPolicy, error)

	// WriteState is responsible for persistently storing state tracking
	// information in the Consul Key/Value Store.
	WriteState(*Config, *ScalingState) error

	// LoadState attempts to read state tracking information from the Consul
	// Key/Value Store. If state tracking information is present, it will be
	// preferred. If no persistent data is available, the method returns the
	// state tracking object unmodified.
	LoadState(*Config, *ScalingState) *ScalingState

	CreateSession(int, chan struct{}) (string, error)

	AcquireLeadership(string, string) bool

	ResignLeadership(string, string)
}
