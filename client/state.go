package client

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/elsevier-core-engineering/replicator/logging"
	"github.com/elsevier-core-engineering/replicator/replicator/structs"
	consul "github.com/hashicorp/consul/api"
)

// ReadState does stuff and things.
func (c *consulClient) ReadState(state *structs.ScalingState) {

	logging.Debug("client/state: attempting to read state tracking "+
		"information from Consul at location %v", state.StatePath)

	// Instantiate new Consul Key/Value client.
	kv := c.consul.KV()

	// Retrieve state tracking information from Consul.
	pair, _, err := kv.Get(state.StatePath, nil)
	if err != nil {
		logging.Error("client/state: an error occurred while attempting to read "+
			"state information from Consul at location %v: %v", state.StatePath, err)

		// We were unable to retrieve state data from Consul, so return the
		// unmodified struct back to the caller.
		return
	} else if pair == nil {
		logging.Debug("client/state: no state tracking information is present "+
			"in Consul at location %v, writing initial state object", state.StatePath)

		// There was no pre-existing state tracking information in Consul,
		// persist an initial state tracking object.
		c.PersistState(state)

		// Return unmodified struct back to the caller.
		return
	}

	// Deserialize state tracking data.
	err = json.Unmarshal(pair.Value, state)
	if err != nil {
		logging.Error("client/state: an error occurred while attempting to "+
			"deserialize scaling state retrieved from persistent storage: %v", err)

		// We were unable to deserialize state data from Consul, so return the
		// unmodified struct back to the caller.
		return
	}

	logging.Debug("client/consul: successfully loaded state tracking "+
		"information from Consul, data was last updated: %v",
		state.LastUpdated)

	logging.Debug("client/state: successfully updated state tracking "+
		"from Consul: %v", state)

	return
}

// WriteState is responsible for persistently storing state tracking
// information in the Consul Key/Value Store.
func (c *consulClient) PersistState(state *structs.ScalingState) (err error) {

	logging.Debug("client/state: attempting to persistently store scaling "+
		"state in Consul at location %v", state.StatePath)

	// Set the last_updated timestamp before serialization
	state.LastUpdated = time.Now()

	// Marshal the state struct into a JSON string for persistent storage.
	scalingState, err := json.Marshal(state)
	if err != nil {
		err = fmt.Errorf("client/state: an error occurred when attempting to "+
			"serialize scaling state for persistent storage: %v", err)
		return
	}

	// Build the key/value pair struct for persistent storage.
	d := &consul.KVPair{
		Key:   state.StatePath,
		Value: scalingState,
	}

	// Instantiate new Consul Key/Value client.
	kv := c.consul.KV()

	// Attempt to write scaling state to Consul Key/Value Store.
	_, err = kv.Put(d, nil)
	if err != nil {
		err = fmt.Errorf("client/state: an error occurred when attempting to "+
			"write scaling state data to Consul: %v", err)
		return
	}

	logging.Debug("client/state: successfully stored scaling state in Consul "+
		"at location %v", state.StatePath)

	return
}
