package helper

import "testing"

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
