package notifier

import (
	"fmt"

	"github.com/PagerDuty/go-pagerduty"
)

// PagerDutyProvider contains the required configuration to send PagerDuty
// notifications.
type PagerDutyProvider struct {
	config map[string]string
}

// Name returns the name of the notification endpoint in a lowercase, human
// readable format.
func (p *PagerDutyProvider) Name() string {
	return "pagerduty"
}

// NewPagerDutyProvider creates the PagerDuty notification provider.
func NewPagerDutyProvider(c map[string]string) (Notifier, error) {

	p := &PagerDutyProvider{
		config: c,
	}

	return p, nil
}

// SendNotification will send a notification to PagerDuty using the Event
// library call to create a new incident.
func (p *PagerDutyProvider) SendNotification(message FailureMessage) (key string, err error) {

	// Format the message description.
	d := fmt.Sprintf("%s %s_%s",
		message.AlertUID, message.ClusterIdentifier, message.IncidentReason)

	// Setup the PagerDuty event structure which will then be used to trigger
	// the event call.
	event := pagerduty.Event{
		ServiceKey:  p.config["PagerDutyServiceKey"],
		Type:        "trigger",
		Description: d,
		Details:     message,
	}

	resp, err := pagerduty.CreateEvent(event)
	if err != nil {
		return "", err
	}

	return resp.IncidentKey, nil
}
