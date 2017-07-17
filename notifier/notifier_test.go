package notifier

import (
	"strings"
	"testing"
)

func TestNotifier_NewProvider(t *testing.T) {
	fakeProv := make(map[string]string)

	_, err := NewProvider("OperationsOnlyOnCall", fakeProv)
	fakeNotExpected := "the notifications provider OperationsOnlyOnCall is not supported"

	if !strings.Contains(err.Error(), fakeNotExpected) {
		t.Fatalf("expected %q to include %q", err.Error(), fakeNotExpected)
	}

	pdProv := make(map[string]string)

	pd, err := NewProvider("pagerduty", pdProv)
	if err != nil {
		t.Fatalf("expected pdProv error to be nil, got %v", err)
	}
	pdName := pd.Name()
	if pdName != "pagerduty" {
		t.Fatalf("expected pdProv Name to be pagerduty, got %v", pdName)
	}
}
