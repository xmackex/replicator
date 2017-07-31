package helper

import (
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
