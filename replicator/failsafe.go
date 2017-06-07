package replicator

import (
	"fmt"

	"github.com/elsevier-core-engineering/replicator/logging"
	"github.com/elsevier-core-engineering/replicator/replicator/structs"
)

// FailsafeCheck determines if any critical failure modes have been detected
// and if so, automatically places Replicator in failsafe mode.
func FailsafeCheck(state *structs.State, config *structs.Config) (passing bool) {
	// Assume we're in a good state until proven otherwise.
	passing = true

	// If attempts to launch new worker pool nodes have failed and we've
	// reached or exceeded the retry threshold, we should put the daemon in
	// failsafe mode.
	if state.NodeFailureCount >= config.ClusterScaling.RetryThreshold {
		passing = false
	}

	switch passing {
	case true:
		logging.Debug("core/failsafe: the failsafe check passes, scaling " +
			"evaluations and operations will be permitted.")
	case false:
		SetFailsafeMode(state, config, true)
	}

	return
}

// SetFailsafeMode is used to toggle the global failsafe lock.
func SetFailsafeMode(state *structs.State, config *structs.Config, enabled bool) (err error) {
	switch enabled {
	case true:
		if !state.FailsafeMode {
			// TODO (e.westfall) Send notification here after we've sorted
			// notification client initialization.
		}

		// Suppress logging output if we're being called from the CLI tools.
		if !state.FailsafeModeAdmin {
			logging.Warning("core/failsafe: Replicator has been placed in failsafe " +
				"mode. No scaling evaluations or operations will be permitted from " +
				"any running copies of Replicator.")
		}

	case false:
		if !state.FailsafeModeAdmin {
			logging.Info("core/failsafe: exiting failsafe mode")
		}
	}

	// Set the failsafe mode flag in the state object.
	state.FailsafeMode = enabled

	// Attempt to update the persistent state tracking data.
	err = config.ConsulClient.WriteState(config, state)
	if err != nil {
		return fmt.Errorf("core/failsafe: an attempt to update the persistent "+
			"state tracking data failed: %v", err)
	}

	return nil
}
