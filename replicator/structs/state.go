package structs

import (
	"sync"
	"time"
)

// ScalingState provides a state object that represents the state
// of a scaleable worker pool or job group.
type ScalingState struct {
	// FailsafeMode tracks whether the daemon has exceeded the fault threshold
	// while attempting to perform scaling operations. When operating in failsafe
	// mode, the daemon will decline to take scaling actions of any type.
	FailsafeMode bool `json:"failsafe_mode"`

	// FailureCount tracks the number of worker nodes that have failed to
	// successfully join the worker pool after a scale-out operation.
	FailureCount int `json:"failure_count"`

	// LastNotificationEvent tracks the time of the last notification send run
	// for this state object.
	LastNotificationEvent time.Time `json:"last_notification_event"`

	// LastScalingEvent represents the last time the daemon successfully
	// completed a cluster scaling action.
	LastScalingEvent time.Time `json:"last_scaling_event"`

	// LastUpdated tracks the last time the state tracking data was updated.
	LastUpdated time.Time `json:"last_updated"`

	// Lock provides a mutex lock to protect concurrent read/write
	// access to the object.
	Lock sync.RWMutex `json:"-"`

	// ProtectedNode represents the Nomad agent node on which the Replicator
	// leader is running. This node will be excluded when identifying an eligible
	// node for termination during scaling actions.
	ProtectedNode string `json:"protected_node"`

	// ScaleInRequests tracks the number of consecutive times replicator
	// has indicated the cluster worker pool should be scaled in.
	ScaleInRequests int `json:"scalein_requests"`

	// ScaleOutRequests tracks the number of consecutive times replicator
	// has indicated the cluster worker pool should be scaled out.
	ScaleOutRequests int `json:"scaleout_requests"`
}
