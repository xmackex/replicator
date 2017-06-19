package notifier

import (
	"fmt"
)

// FailureMessage is the notifier struct that contains all relevant notification
// information to provide to operators and developers.
type FailureMessage struct {
	AlertUID          string
	ClusterIdentifier string
	Reason            string
	FailedResource    string
}

// Notifier is the interface to the Notifiers functions. All notifers are
// expected to implament this set of functions.
type Notifier interface {
	Name() string
	SendNotification(FailureMessage)
}

// NewProvider is the factory entrace to the notifications backends.
func NewProvider(t string, c map[string]string) (Notifier, error) {

	var n Notifier
	var err error

	switch t {
	case "pagerduty":
		n, err = NewPagerDutyProvider(c)
	default:
		err = fmt.Errorf("the notifications provider %s is not supported", t)
	}
	return n, err
}
