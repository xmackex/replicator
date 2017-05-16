package agent

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/elsevier-core-engineering/replicator/replicator/structs"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/hcl"
	"github.com/hashicorp/hcl/hcl/ast"
	"github.com/mitchellh/mapstructure"
)

// ParseConfigFile parses the given path as a config file.
func ParseConfigFile(path string) (*structs.Config, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	config, err := ParseConfig(f)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// ParseConfig parses the config from the given io.Reader.
func ParseConfig(r io.Reader) (*structs.Config, error) {

	// Copy the reader into an in-memory buffer first since HCL requires it.
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		return nil, err
	}

	// Parse the buffer
	root, err := hcl.Parse(buf.String())
	if err != nil {
		return nil, fmt.Errorf("error parsing: %s", err)
	}
	buf.Reset()

	// The top-level item should be a list.
	list, ok := root.Node.(*ast.ObjectList)
	if !ok {
		return nil, fmt.Errorf("error parsing: root should be an object")
	}

	var config structs.Config
	if err := parseConfig(&config, list); err != nil {
		return nil, fmt.Errorf("error parsing 'config': %v", err)
	}

	return &config, nil
}

func parseConfig(result *structs.Config, list *ast.ObjectList) error {

	// Check for invalid keys
	valid := []string{
		"nomad",
		"consul",
		"log_level",
		"log_level",
		"scaling_interval",
		"aws_region",
		"cluster_scaling",
		"job_scaling",
		"telemetry",
		"notification",
	}
	if err := checkHCLKeys(list, valid); err != nil {
		return multierror.Prefix(err, "config:")
	}

	// Decode the full thing into a map[string]interface, removing these top
	// levels before continuing to decode the remaining configuraiton.
	var m map[string]interface{}
	if err := hcl.DecodeObject(&m, list); err != nil {
		return err
	}

	delete(m, "cluster_scaling")
	delete(m, "job_scaling")
	delete(m, "telemetry")
	delete(m, "notification")

	if err := mapstructure.WeakDecode(m, result); err != nil {
		return err
	}

	// Parse the nested configuration portions which currently is ClusterScaling,
	// JobScaling and Telemetry.
	if o := list.Filter("cluster_scaling"); len(o.Items) > 0 {
		if err := parseClusterScaling(&result.ClusterScaling, o); err != nil {
			return multierror.Prefix(err, "cluster_scaling ->")
		}
	}

	if o := list.Filter("job_scaling"); len(o.Items) > 0 {
		if err := parseJobScaling(&result.JobScaling, o); err != nil {
			return multierror.Prefix(err, "job_scaling ->")
		}
	}

	if o := list.Filter("telemetry"); len(o.Items) > 0 {
		if err := parseTelemetry(&result.Telemetry, o); err != nil {
			return multierror.Prefix(err, "telemetry ->")
		}
	}

	if o := list.Filter("notification"); len(o.Items) > 0 {
		if err := parseNotification(&result.Notification, o); err != nil {
			return multierror.Prefix(err, "notification ->")
		}
	}

	return nil
}

func parseClusterScaling(result **structs.ClusterScaling, list *ast.ObjectList) error {
	list = list.Elem()
	if len(list.Items) > 1 {
		return fmt.Errorf("only one 'cluster_scaling' block allowed")
	}

	listVal := list.Items[0].Val

	// Check for invalid keys
	valid := []string{
		"enabled",
		"max_size",
		"min_size",
		"cool_down",
		"node_fault_tolerance",
		"autoscaling_group",
	}
	if err := checkHCLKeys(listVal, valid); err != nil {
		return err
	}

	var m map[string]interface{}
	if err := hcl.DecodeObject(&m, listVal); err != nil {
		return err
	}

	var cluster structs.ClusterScaling
	if err := mapstructure.WeakDecode(m, &cluster); err != nil {
		return err
	}
	*result = &cluster
	return nil
}

func parseJobScaling(result **structs.JobScaling, list *ast.ObjectList) error {
	list = list.Elem()
	if len(list.Items) > 1 {
		return fmt.Errorf("only one 'job_scaling' block allowed")
	}

	listVal := list.Items[0].Val

	// Check for invalid keys
	valid := []string{
		"enabled",
		"consul_token",
		"consul_key_location",
	}
	if err := checkHCLKeys(listVal, valid); err != nil {
		return err
	}

	var m map[string]interface{}
	if err := hcl.DecodeObject(&m, listVal); err != nil {
		return err
	}

	var job structs.JobScaling
	if err := mapstructure.WeakDecode(m, &job); err != nil {
		return err
	}
	*result = &job
	return nil
}

func parseTelemetry(result **structs.Telemetry, list *ast.ObjectList) error {
	list = list.Elem()
	if len(list.Items) > 1 {
		return fmt.Errorf("only one 'Telemetry' block allowed")
	}

	listVal := list.Items[0].Val

	// Check for invalid keys
	valid := []string{
		"statsd_address",
	}
	if err := checkHCLKeys(listVal, valid); err != nil {
		return err
	}

	var m map[string]interface{}
	if err := hcl.DecodeObject(&m, listVal); err != nil {
		return err
	}

	var telemetry structs.Telemetry
	if err := mapstructure.WeakDecode(m, &telemetry); err != nil {
		return err
	}
	*result = &telemetry
	return nil
}

func parseNotification(result **structs.Notification, list *ast.ObjectList) error {
	list = list.Elem()
	if len(list.Items) > 1 {
		return fmt.Errorf("only one 'Notification' block allowed")
	}

	listVal := list.Items[0].Val

	// Check for invalid keys
	valid := []string{
		"cluster_scaling_uid",
		"cluster_identifier",
		"pagerduty_service_key",
	}
	if err := checkHCLKeys(listVal, valid); err != nil {
		return err
	}

	var m map[string]interface{}
	if err := hcl.DecodeObject(&m, listVal); err != nil {
		return err
	}

	var notification structs.Notification
	if err := mapstructure.WeakDecode(m, &notification); err != nil {
		return err
	}
	*result = &notification
	return nil
}

func checkHCLKeys(node ast.Node, valid []string) error {
	var list *ast.ObjectList
	switch n := node.(type) {
	case *ast.ObjectList:
		list = n
	case *ast.ObjectType:
		list = n.List
	default:
		return fmt.Errorf("cannot check HCL keys of type %T", n)
	}

	validMap := make(map[string]struct{}, len(valid))
	for _, v := range valid {
		validMap[v] = struct{}{}
	}

	var result error
	for _, item := range list.Items {
		key := item.Keys[0].Token.Value().(string)
		if _, ok := validMap[key]; !ok {
			result = multierror.Append(result, fmt.Errorf(
				"invalid key: %s", key))
		}
	}

	return result
}
