package structs

// Config is the main configuration struct used to configure the replicator
// application.
type Config struct {
	// Consul is the location of the Consul instance or cluster endpoint to query
	// (may be an IP address or FQDN) with port.
	Consul string `mapstructure:"consul"`

	// Nomad is the location of the Nomad instance or cluster endpoint to query
	// (may be an IP address or FQDN) with port.
	Nomad string `mapstructure:"nomad"`

	// LogLevel is the level at which the application should log from.
	LogLevel string `mapstructure:"log_level"`

	// ScalingInterval is the duration in seconds between Replicator runs and thus
	// scaling requirement checks.
	ScalingInterval int `mapstructure:"scaling_interval"`

	// Region represents the AWS region the cluster resides in.
	Region string `mapstructure:"aws_region"`

	// ClusterScaling is the configuration struct that controls the basic Nomad
	// worker node scaling.
	ClusterScaling *ClusterScaling `mapstructure:"cluster_scaling"`

	// JobScaling is the configuration struct that controls the basic Nomad
	// job scaling.
	JobScaling *JobScaling `mapstructure:"job_scaling"`

	// Telemetry is the configuration struct that controls the telemetry settings.
	Telemetry *Telemetry `mapstructure:"telemetry"`

	// ConsulClient provides a client to interact with the Consul API.
	ConsulClient ConsulClient

	// NomadClient provides a client to interact with the Nomad API.
	NomadClient NomadClient
}

// ClusterScaling is the configuration struct for the Nomad worker node scaling
// activites.
type ClusterScaling struct {
	// Enabled indicates whether cluster scaling actions are permitted.
	Enabled bool `mapstructure:"enabled"`

	// MaxSize in the maximum number of instances the nomad node worker count is
	// allowed to reach. This stops runaway increases in size due to misbehaviour
	// but should be set high enough to accomodate usual workload peaks.
	MaxSize int `mapstructure:"max_size"`

	// MinSize is the minimum number of instances that should be present within
	// the nomad node worker pool.
	MinSize int `mapstructure:"min_size"`

	// CoolDown is the number of seconds after a scaling activity completes before
	// another can begin.
	CoolDown float64 `mapstructure:"cool_down"`

	// NodeFaultTolerance is the number of Nomad worker nodes the cluster can
	// support losing, whilst still maintaining all existing workload.
	NodeFaultTolerance int `mapstructure:"node_fault_tolerance"`

	// AutoscalingGroup is the name of the ASG assigned to the Nomad worker nodes.
	AutoscalingGroup string `mapstructure:"autoscaling_group"`
}

// JobScaling is the configuration struct for the Nomad job scaling activities.
type JobScaling struct {
	// Enabled indicates whether job scaling actions are permitted.
	Enabled bool `mapstructure:"enabled"`

	// ConsulToken is the Consul ACL token used to access KeyValues from a
	// secure Consul installation.
	ConsulToken string `mapstructure:"consul_token"`

	// ConsulKeyLocation is the Consul key location where scaling policies are
	// defined.
	ConsulKeyLocation string `mapstructure:"consul_key_location"`
}

// Telemetry is the struct that control the telemetry configuration. If a value
// is present then telemetry is enabled. Currently statsd is only supported for
// sending telemetry.
type Telemetry struct {
	// StatsdAddress specifies the address of a statsd server to forward metrics
	// to and should include the port.
	StatsdAddress string `mapstructure:"statsd_address"`
}

// Merge merges two configurations.
func (c *Config) Merge(b *Config) *Config {
	config := *c

	if b.Nomad != "" {
		config.Nomad = b.Nomad
	}

	if b.Consul != "" {
		config.Consul = b.Consul
	}

	if b.LogLevel != "" {
		config.LogLevel = b.LogLevel
	}

	if b.ScalingInterval > 0 {
		config.ScalingInterval = b.ScalingInterval
	}

	if b.Region != "" {
		config.Region = b.Region
	}

	// Apply the ClusterScaling config
	if config.ClusterScaling == nil && b.ClusterScaling != nil {
		clusterScaling := *b.ClusterScaling
		config.ClusterScaling = &clusterScaling
	} else if b.ClusterScaling != nil {
		config.ClusterScaling = config.ClusterScaling.Merge(b.ClusterScaling)
	}

	// Apply the JobScaling config
	if config.JobScaling == nil && b.JobScaling != nil {
		jobScaling := *b.JobScaling
		config.JobScaling = &jobScaling
	} else if b.JobScaling != nil {
		config.JobScaling = config.JobScaling.Merge(b.JobScaling)
	}

	// Apply the Telemetry config
	if config.Telemetry == nil && b.Telemetry != nil {
		telemetry := *b.Telemetry
		config.Telemetry = &telemetry
	} else if b.Telemetry != nil {
		config.Telemetry = config.Telemetry.Merge(b.Telemetry)
	}

	return &config
}

// Merge is used to merge two ClusterScaling configurations together.
func (c *ClusterScaling) Merge(b *ClusterScaling) *ClusterScaling {
	config := *c

	if b.Enabled {
		config.Enabled = b.Enabled
	}

	if b.MaxSize != 0 {
		config.MaxSize = b.MaxSize
	}

	if b.MinSize != 0 {
		config.MinSize = b.MinSize
	}

	if b.CoolDown != 0 {
		config.CoolDown = b.CoolDown
	}

	if b.NodeFaultTolerance != 0 {
		config.NodeFaultTolerance = b.NodeFaultTolerance
	}

	if b.AutoscalingGroup != "" {
		config.AutoscalingGroup = b.AutoscalingGroup
	}

	return &config
}

// Merge is used to merge two JobScaling configurations together.
func (j *JobScaling) Merge(b *JobScaling) *JobScaling {
	config := *j

	if b.Enabled {
		config.Enabled = b.Enabled
	}

	if b.ConsulToken != "" {
		config.ConsulToken = b.ConsulToken
	}

	if b.ConsulKeyLocation != "" {
		config.ConsulKeyLocation = b.ConsulKeyLocation
	}

	return &config
}

// Merge is used to merge two Telemetry configurations together.
func (t *Telemetry) Merge(b *Telemetry) *Telemetry {
	config := *t

	if t.StatsdAddress != "" {
		config.StatsdAddress = b.StatsdAddress
	}

	return &config
}
