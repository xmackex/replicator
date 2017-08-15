package helper

import (
	"fmt"
	"reflect"
	"regexp"

	"github.com/elsevier-core-engineering/replicator/logging"
	"github.com/elsevier-core-engineering/replicator/replicator/structs"
	"github.com/mitchellh/hashstructure"
)

// FindIP will return the IP address from a string. This is used to deal with
// responses from the Nomad API that contain the port such as 127.0.0.1:4646.
func FindIP(input string) string {
	numBlock := "(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])"
	regexPattern := numBlock + "\\." + numBlock + "\\." + numBlock + "\\." + numBlock

	regEx := regexp.MustCompile(regexPattern)
	return regEx.FindString(input)
}

// Max returns the largest float from a variable length list of floats.
func Max(values ...float64) float64 {
	max := values[0]
	for _, i := range values[1:] {
		if i > max {
			max = i
		}
	}

	return max
}

// Min returns the smallest float from a variable length list of floats.
func Min(values ...float64) float64 {
	min := values[0]
	for _, i := range values[1:] {
		if i < min {
			min = i
		}
	}
	return min
}

// ParseMetaConfig parses meta parameters from a Nomad agent or job
// configuration and validates required keys are present. If any required
// keys are found to be missing, these are returned otherwise, an empty
// slice is returned.
func ParseMetaConfig(meta map[string]string, reqKeys []string) (missing []string) {
	// Iterate over the required configuration parameters and
	// record any that are missing.
	for _, reqKey := range reqKeys {
		if _, ok := meta[reqKey]; !ok {
			missing = append(missing, reqKey)
		}
	}
	return
}

// JobGroupScalingPolicyDiff performs a comparison between two GroupScalingPolicy
// structs to determine if they are the same or not.
func JobGroupScalingPolicyDiff(policyA, policyB *structs.GroupScalingPolicy) (isSame bool) {
	policyAHash, err := hashstructure.Hash(policyA, nil)
	if err != nil {
		logging.Error("helper/funcs: errror hashing policy %v: ", policyA.GroupName, err)
	}

	policyBHash, err := hashstructure.Hash(policyB, nil)
	if err != nil {
		logging.Error("helper/funcs: errror hashing policy %v: ", policyB.GroupName, err)
	}

	if policyAHash == policyBHash {
		isSame = true
	}
	return
}

// HasObjectChanged compares two objects to determine if they have changed.
func HasObjectChanged(objectA, objectB interface{}) (changed bool, err error) {
	objectAHash, err := hashstructure.Hash(objectA, nil)
	if err != nil {
		return false, fmt.Errorf("error hashing first object %v of type %v: %v",
			objectA, reflect.TypeOf(objectA), err)
	}

	objectBHash, err := hashstructure.Hash(objectB, nil)
	if err != nil {
		return false, fmt.Errorf("error hashing second object %v of type %v: %v",
			objectA, reflect.TypeOf(objectA), err)
	}

	if objectAHash != objectBHash {
		changed = true
	}

	return
}
