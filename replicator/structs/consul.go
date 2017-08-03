package structs

// The ConsulClient interface is used to provide common method signatures for
// interacting with the Consul API.
type ConsulClient interface {
	// AcquireLeadership attempts to acquire a Consul leadersip lock using the
	// provided session. If the lock is already taken this will return false in
	// a show that there is already a leader.
	AcquireLeadership(string, string) bool

	// CreateSession creates a Consul session for use in the Leadership locking
	// process and will spawn off the renewing of the session in order to ensure
	// leadership can be maintained.
	CreateSession(int, chan struct{}) (string, error)

	// LoadState attempts to read state tracking information from the Consul
	// Key/Value Store. If state tracking information is present, it will be
	// preferred. If no persistent data is available, the method returns the
	// state tracking object unmodified.
	LoadState(*Config, *State) *State

	// ResignLeadership attempts to remove the leadership lock upon shutdown of the
	// replicator daemon. If this is unsuccessful there is not too much we can do
	// therefore there is no return.
	ResignLeadership(string, string)

	// WriteState is responsible for persistently storing state tracking
	// information in the Consul Key/Value Store.
	WriteState(*Config, *State) error
}
