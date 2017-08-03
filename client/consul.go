package client

import (
	"encoding/json"
	"fmt"
	"time"

	metrics "github.com/armon/go-metrics"
	"github.com/elsevier-core-engineering/replicator/logging"
	"github.com/elsevier-core-engineering/replicator/replicator/structs"
	consul "github.com/hashicorp/consul/api"
)

// The client object is a wrapper to the Consul client provided by the Consul
// API library.
type consulClient struct {
	consul *consul.Client
}

// NewConsulClient is used to construct a new Consul client using the default
// configuration and supporting the ability to specify a Consul API address
// endpoint in the form of address:port.
func NewConsulClient(addr, token string) (structs.ConsulClient, error) {
	// TODO (e.westfall): Add a quick health check call to an API endpoint to
	// validate connectivity or return an error back to the caller.
	config := consul.DefaultConfig()
	config.Address = addr

	if token != "" {
		config.Token = token
	}

	c, err := consul.NewClient(config)
	if err != nil {
		// TODO (e.westfall): Raise error here.
		return nil, err
	}

	return &consulClient{consul: c}, nil
}

// LoadState attempts to read state tracking information from the Consul
// Key/Value Store. If state tracking information is present, it will be
// deserialized and returned as a state tracking object. If no persistent
// data is available, the method returns the state tracking object unmodified.
func (c *consulClient) LoadState(config *structs.Config, state *structs.State) *structs.State {

	stateKey := config.ConsulKeyLocation + "/" + "state"

	logging.Debug("client/consul: attempting to load state tracking "+
		"information from Consul at location %v", stateKey)

	// Create new scaling state struct to hold state data retrieved from Consul.
	updatedState := &structs.State{}

	// Instantiate new Consul Key/Value client.
	kv := c.consul.KV()

	// Retrieve state tracking information from Consul.
	pair, _, err := kv.Get(stateKey, nil)
	if err != nil {
		logging.Error("client/consul: an error occurred while attempting to read "+
			"state information from Consul at location %v: %v", stateKey, err)

		// We were unable to retrieve state data from Consul, so return the
		// unmodified struct back to the caller.
		return state
	} else if pair == nil {
		logging.Debug("client/consul: no state tracking information is present "+
			"in Consul at location %v, falling back to in-memory state", stateKey)

		// No state tracking information was located in Consul, so return the
		// unmodified struct back to the caller.
		return state
	}

	// Deserialize state tracking data.
	err = json.Unmarshal(pair.Value, updatedState)
	if err != nil {
		logging.Error("client/consul: an error occurred while attempting to "+
			"deserialize scaling state retrieved from persistent storage: %v", err)

		// We were unable to deserialize state data from Consul, so return the
		// unmodified struct back to the caller.
		return state
	}

	logging.Debug("client/consul: successfully loaded state tracking "+
		"information from Consul, data was last updated: %v",
		updatedState.LastUpdated)

	return updatedState
}

// WriteState is responsible for persistently storing state tracking
// information in the Consul Key/Value Store.
func (c *consulClient) WriteState(config *structs.Config, state *structs.State) (err error) {

	stateKey := config.ConsulKeyLocation + "/" + "state"

	logging.Debug("client/consul: attempting to persistently store scaling "+
		"state in Consul at location %v", stateKey)

	// Set the last_updated timestamp before serialization
	state.LastUpdated = time.Now()

	// Marshal the state struct into a JSON string for persistent storage.
	scalingState, err := json.Marshal(state)
	if err != nil {
		err = fmt.Errorf("client/consul: an error occurred when attempting to "+
			"serialize scaling state for persistent storage: %v", err)
		return
	}

	// Build the key/value pair struct for persistent storage.
	d := &consul.KVPair{
		Key:   stateKey,
		Value: scalingState,
	}

	// Instantiate new Consul Key/Value client.
	kv := c.consul.KV()

	// Attempt to write scaling state to Consul Key/Value Store.
	_, err = kv.Put(d, nil)
	if err != nil {
		err = fmt.Errorf("client/consul: an error occurred when attempting to "+
			"write scaling state data to Consul: %v", err)
		return
	}

	logging.Debug("client/consul: successfully stored scaling state in Consul "+
		"at location %v", stateKey)

	return
}

// CreateSession creates a Consul session for use in the Leadership locking
// process and will spawn off the renewing of the session in order to ensure
// leadership can be maintained.
func (c *consulClient) CreateSession(ttl int, stopCh chan struct{}) (id string, err error) {

	entry := &consul.SessionEntry{
		TTL:  fmt.Sprintf("%vs", ttl),
		Name: "replicator_leader_lock",
	}

	// Create the key session.
	logging.Debug("client/consul: obtaining Consul session with %vs TTL", ttl)
	resp, _, err := c.consul.Session().Create(entry, nil)
	if err != nil {
		return "", err
	}

	// Spawn off to continue renewing our session.
	c.renewSession(entry.TTL, resp, stopCh)

	return resp, nil
}

// AcquireLeadership attempts to acquire a Consul leadersip lock using the
// provided session. If the lock is already taken this will return false in
// a show that there is already a leader.
func (c *consulClient) AcquireLeadership(key, session string) (aquired bool) {

	// Attempt to inspect the leadership key if it is available and present.
	k, _, err := c.consul.KV().Get(key, nil)

	if err != nil {
		logging.Error("client/consul: unable to read the leader key at %s", key)
		return false
	}

	// On a fresh cluster the KV might not exist yet, so we need to check for nil
	// return. If the leadership lock is tied to our session then we can exit and
	// confirm we are running as the replicator leader without having to make on
	// further calls.
	if k != nil && k.Session == session {
		return true
	}

	kp := &consul.KVPair{
		Key:     key,
		Session: session,
	}

	logging.Debug("client/consul: attempting to aquire leadership lock at %s", key)
	resp, _, err := c.consul.KV().Acquire(kp, nil)

	if err != nil {
		logging.Error("client/consul: issue requesting leadership lock: %v", err)
		return false
	}

	// We have successfully obtained the leadership and can now be considered as
	// the replicator leader.
	if resp {
		logging.Info("client/consul: leadership lock successfully obtained at %s", key)
		metrics.IncrCounter([]string{"cluster", "leadership", "election"}, 1)
		return true
	}

	return false
}

// ResignLeadership attempts to remove the leadership lock upon shutdown of the
// replicator daemon. If this is unsuccessful there is not too much we can do
// therefore there is no return.
func (c *consulClient) ResignLeadership(key, session string) {

	kp := &consul.KVPair{
		Key:     key,
		Session: session,
	}

	resp, _, err := c.consul.KV().Release(kp, nil)
	if err != nil {
		logging.Error("client/consul: unable to successfully release leadership lock: %v", err)
		return
	}

	// If we get a successful response we should log it.
	if resp {
		logging.Info("client/consul: the leadership lock has now been released")
	}

	return
}

// renewSession is used for renewing a Consul session and accepts a channel
// within which a signal can be sent which will stop the renawl process and
// attempt to clean up the session.
func (c *consulClient) renewSession(ttl string, session string, renewChan chan struct{}) {

	sessionDestroyAttempts := 0
	maxSessionDestroyAttempts := 5

	parsedTTL, err := time.ParseDuration(ttl)
	if err != nil {
		return
	}

	go func() {
		for {
			select {
			case <-time.After(parsedTTL / 2):
				entry, _, err := c.consul.Session().Renew(session, nil)
				if err != nil {
					continue
				}
				if entry == nil {
					return
				}

				// Consul may return a TTL value higher than the one specified during
				// session creation. This indicates the server is under high load and
				// is requesting clients renew less often. If this happens we need to
				// ensure we track the new TTL.
				parsedTTL, _ = time.ParseDuration(entry.TTL)
				logging.Debug("client/consul: the Consul session %s has been renewed", session)

			case <-renewChan:
				_, err := c.consul.Session().Destroy(session, nil)
				if err == nil {
					logging.Info("client/consul: the Consul session %s has been released", session)
					return
				}

				if sessionDestroyAttempts >= maxSessionDestroyAttempts {
					logging.Error("client/consul: unable to successfully destroy the Consul session %s", session)
					return
				}

				// We can't destroy the session so we will wait and attempt again until
				// we hit the threshold.
				sessionDestroyAttempts++
				time.Sleep(parsedTTL)
			}
		}
	}()
}
