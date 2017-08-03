package helper

import (
	"testing"

	"github.com/elsevier-core-engineering/replicator/replicator/structs"
)

func TestHelper_FindIpP(t *testing.T) {

	input := "10.0.0.10:4646"
	expected := "10.0.0.10"

	ip := FindIP(input)
	if ip != expected {
		t.Fatalf("expected %s got %s", expected, ip)
	}
}

func TestHelper_Max(t *testing.T) {

	expected := 13.12

	max := Max(13.12, 2.01, 6.4, 13.11, 1.01, 0.11)
	if max != expected {
		t.Fatalf("expected %v got %v", expected, max)
	}
}

func TestHelper_Min(t *testing.T) {

	expected := 1.01

	min := Min(13.12, 2.01, 6.4, 13.11, 1.01, 1.02)
	if min != expected {
		t.Fatalf("expected %v got %v", expected, min)
	}
}

func TestHelper_JobGroupScalingPolicyDiff(t *testing.T) {
	policyA := &structs.GroupScalingPolicy{
		GroupName:   "core-engineering",
		Enabled:     true,
		Max:         10,
		Min:         1,
		ScaleInMem:  10,
		ScaleInCPU:  20,
		ScaleOutMem: 90,
		ScaleOutCPU: 90,
	}
	policyB := &structs.GroupScalingPolicy{
		GroupName:   "core-engineering",
		Enabled:     true,
		Max:         10,
		Min:         1,
		ScaleInMem:  10,
		ScaleInCPU:  20,
		ScaleOutMem: 90,
		ScaleOutCPU: 90,
	}

	if !JobGroupScalingPolicyDiff(policyA, policyB) {
		t.Fatalf("expected true but got false")
	}

	policyB.ScaleDirection = "Out"
	if !JobGroupScalingPolicyDiff(policyA, policyB) {
		t.Fatalf("expected true but got false")
	}

	policyB.ScaleInMem = 20
	if JobGroupScalingPolicyDiff(policyA, policyB) {
		t.Fatalf("expected false but got true")
	}
}
