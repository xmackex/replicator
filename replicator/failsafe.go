package replicator

import (
	"fmt"

	"github.com/elsevier-core-engineering/replicator/logging"
	"github.com/elsevier-core-engineering/replicator/replicator/structs"
)

// FailsafeCheck implements the failsafe mode circuit breaker that will
// trip automatically if enough critical failures are detected. Once
// tripped, the circuit breaker must be reset by a human operator.
func FailsafeCheck(state *structs.State, config *structs.Config) (passing bool) {
	// Assume we're in a good state until proven otherwise.
	passing = true

	// If the failsafe circuit breaker has been tripped already, we can fail
	// quickly here.
	if state.FailsafeMode {
		return false
	}

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

// SetFailsafeMode is used to toggle the distributed failsafe mode lock.
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

	// Set the failsafe mode lock state in the state tracking object.
	state.FailsafeMode = enabled

	// Attempt to update the persistent state tracking information.
	err = config.ConsulClient.WriteState(config, state)
	if err != nil {
		return fmt.Errorf("core/failsafe: an attempt to update the persistent "+
			"state tracking information failed: %v", err)
	}

	return nil
}
