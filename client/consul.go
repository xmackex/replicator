package client

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
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
func NewConsulClient(addr string) (structs.ConsulClient, error) {
	// TODO (e.westfall): Add a quick health check call to an API endpoint to
	// validate connectivity or return an error back to the caller.
	config := consul.DefaultConfig()
	config.Address = addr
	c, err := consul.NewClient(config)
	if err != nil {
		// TODO (e.westfall): Raise error here.
		return nil, err
	}

	return &consulClient{consul: c}, nil
}

// GetJobScalingPolicies provides a list of Nomad jobs with a defined scaling
// policy document at a specified Consuk Key/Value Store location. Supports
// the use of an ACL token if required by the Consul cluster.
func (c *consulClient) GetJobScalingPolicies(config *structs.Config, nomadClient structs.NomadClient) ([]*structs.JobScalingPolicy, error) {

	defer metrics.MeasureSince([]string{"job", "config_read"}, time.Now())

	var entries []*structs.JobScalingPolicy
	keyPath := config.ConsulKeyLocation + "/" + "jobs"

	// Setup the QueryOptions to include the aclToken if this has been set, if not
	// procede with empty QueryOptions struct.
	qop := &consul.QueryOptions{}
	if config.ConsulToken != "" {
		qop.Token = config.ConsulToken
	}

	kvClient := c.consul.KV()
	resp, _, err := kvClient.List(keyPath, qop)
	if err != nil {
		return entries, err
	}

	// Loop the returned list to gather information on each and every job that
	// has a scaling document.
	for _, job := range resp {
		// The results Value is base64 encoded. It is decoded and marshalled into
		// the appropriate struct.
		uEnc := base64.URLEncoding.EncodeToString([]byte(job.Value))
		uDec, _ := base64.URLEncoding.DecodeString(uEnc)
		s := &structs.JobScalingPolicy{}
		json.Unmarshal(uDec, s)

		// Trim the Key and its trailing slash to find the job name.
		s.JobName = strings.TrimPrefix(job.Key, keyPath+"/")

		// Check to see whether the job has running task groups before appending
		// to the return.
		if nomadClient.IsJobRunning(s.JobName) {
			// Each scaling policy document is then appended to a list to form a full
			// view of all scaling documents available to the cluster.
			entries = append(entries, s)
		}
	}

	return entries, nil
}

// LoadState attempts to read state tracking information from the Consul
// Key/Value Store. If state tracking information is present, it will be
// deserialized and returned as a state tracking object. If no persistent
// data is available, the method returns the state tracking object unmodified.
func (c *consulClient) LoadState(config *structs.Config, state *structs.ScalingState) *structs.ScalingState {
	// TODO (e.westfall): Convert to using base path from configuration, see
	// GH-94 for further details.
	stateKey := "replicator/config/state"

	logging.Debug("client/consul: attempting to load state tracking "+
		"information from Consul at location %v", stateKey)

	// Create new scaling state struct to hold state data retrieved from Consul.
	updatedState := &structs.ScalingState{}

	// Setup the Consul QueryOptions to include an ACL token if on has been set;
	// if not proceed with an empty options struct.
	opts := &consul.QueryOptions{}
	if config.ConsulToken != "" {
		opts.Token = config.ConsulToken
	}

	// Instantiate new Consul Key/Value client.
	kv := c.consul.KV()

	// Retrieve state tracking information from Consul.
	pair, _, err := kv.Get(stateKey, opts)
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
func (c *consulClient) WriteState(config *structs.Config, state *structs.ScalingState) (err error) {
	// TODO (e.westfall): Convert to using base path from configuration, see
	// GH-94 for further details.
	stateKey := "replicator/config/state"

	logging.Debug("client/consul: attempting to persistently store scaling "+
		"state in Consul at location %v", stateKey)

	// Setup the Consul WriteOptions to include an ACL token if on has been set;
	// if not proceed with an empty options struct.
	opts := &consul.WriteOptions{}
	if config.ConsulToken != "" {
		opts.Token = config.ConsulToken
	}

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
	_, err = kv.Put(d, opts)
	if err != nil {
		err = fmt.Errorf("client/consul: an error occurred when attempting to "+
			"write scaling state data to Consul: %v", err)
		return
	}

	logging.Debug("client/consul: successfully stored scaling state in Consul "+
		"at location %v", stateKey)

	return
}
