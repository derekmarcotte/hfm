package main

import "testing"
import "reflect"
import "errors"

func matchesInitial(r Rule) error {
	if r.Interval != 1 {
		return errors.New("interval")
	}

	if r.TimeoutInt != 1 {
		return errors.New("timeoutInt")
	}

	if r.Status != RuleStatusEnabled {
		return errors.New("status")
	}

	if r.Shell != "/bin/sh" {
		return errors.New("shell")
	}

	return nil
}

func matchesDefaults(r Rule) error {
	if r.IntervalFail != r.IntervalFail {
		return errors.New("intervalFail")
	}

	if r.TimeoutKill != r.TimeoutInt+3 {
		return errors.New("timeoutKill")
	}

	return nil
}

func matchesInherited(r Rule, e Rule) error {
	if r.Status != e.Status {
		return errors.New("status")
	}

	if r.Shell != e.Shell {
		return errors.New("shell")
	}

	if r.Interval != e.Interval {
		return errors.New("interval")
	}

	if r.IntervalFail != e.IntervalFail {
		return errors.New("intervalFail")
	}

	if r.TimeoutInt != e.TimeoutInt {
		return errors.New("timeoutInt")
	}

	return nil
}

func matchesExpected(r Rule, e Rule) bool {
	return reflect.DeepEqual(reflect.ValueOf(r), reflect.ValueOf(e))
}

func TestConfigEmpty(t *testing.T) {
	var c Configuration

	if e := c.SetConfiguration(""); e != nil {
		t.Errorf("Received error for empty config: %v", e)
	}

	if len(c.Rules) != 0 {
		t.Errorf("Received non-empty rule set: %+v", c.Rules)
	}
}

func TestConfigBasic(t *testing.T) {
	var c Configuration
	var rule *Rule

	if e := c.SetConfiguration("test=\"true\""); e != nil {
		t.Errorf("Received error for basic config: %v", e)
	}

	if len(c.Rules) != 1 {
		t.Errorf("Received unexpected number of rules: %d", len(c.Rules))
	}

	rule, ok := c.Rules["default"]
	if !ok || rule.Test != "true" {
		t.Errorf("Received unexpected rule: %+v", *rule)
	}
	if e := matchesInitial(*rule); e != nil {
		t.Errorf("Rule didn't match expected value for '%s': %+v", e, rule)
	}
	if e := matchesDefaults(*rule); e != nil {
		t.Errorf("Rule didn't match expected value for '%s': %+v", e, rule)
	}
	//fmt.Printf("%+v\n", *rule)
}

func TestConfigGroup(t *testing.T) {
	var c Configuration
	var rule *Rule

	if e := c.SetConfiguration("t1 { test=\"true\" } "); e != nil {
		t.Errorf("Received error for basic config: %v", e)
	}

	if len(c.Rules) != 1 {
		t.Errorf("Received unexpected number of rules: %d", len(c.Rules))
	}

	rule, ok := c.Rules["t1"]
	if !ok || rule.Test != "true" {
		t.Errorf("Received unexpected rule: %+v", *rule)
	}
	if e := matchesInitial(*rule); e != nil {
		t.Errorf("Rule didn't match expected value for '%s': %+v", e, rule)
	}
	if e := matchesDefaults(*rule); e != nil {
		t.Errorf("Rule didn't match expected value for '%s': %+v", e, rule)
	}
	//fmt.Printf("%+v\n", *rule)
}

func TestConfigInheritedFromDefault(t *testing.T) {
	var c Configuration
	var rule *Rule
	exp := Rule{Status: RuleStatusRunOnce, Shell: "/nonexistent", Interval: 2, IntervalFail: 3, TimeoutInt: 4, TimeoutKill: 7}
	cfg := `
status=run-once
shell=/nonexistent
interval=2
fail_interval=3
timeout_int=4
r1 {
	test="true"
}`

	if e := c.SetConfiguration(cfg); e != nil {
		t.Errorf("Received error for basic config: %v", e)
	}

	if len(c.Rules) != 1 {
		t.Errorf("Received unexpected number of rules: %d", len(c.Rules))
	}

	rule, ok := c.Rules["r1"]
	if !ok || rule.Test != "true" {
		t.Errorf("Received unexpected rule: %+v", *rule)
	}

	if e := matchesInherited(*rule, exp); e != nil {
		t.Errorf("Rule didn't match expected value for '%s': %+v", e, rule)
	}
	//fmt.Printf("%+v\n", *rule)
}

func TestConfigMultipleInherited(t *testing.T) {
	var c Configuration
	var rule *Rule
	exp := Rule{Status: RuleStatusRunOnce, Shell: "/nonexistent", Interval: 5, IntervalFail: 6, TimeoutInt: 7, TimeoutKill: 10}
	cfg := `
status=run-once
shell=/nonexistent
interval=2
fail_interval=3
timeout_int=4
g1 {
	interval=5
	fail_interval=6
	timeout_int=7
	r1 {
		test="true"
	}
}`

	if e := c.SetConfiguration(cfg); e != nil {
		t.Errorf("Received error for basic config: %v", e)
	}

	if len(c.Rules) != 1 {
		t.Errorf("Received unexpected number of rules: %d", len(c.Rules))
	}

	rule, ok := c.Rules["g1/r1"]
	if !ok || rule.Test != "true" {
		t.Errorf("Received unexpected rule: %+v", *rule)
	}

	if e := matchesInherited(*rule, exp); e != nil {
		t.Errorf("Rule didn't match expected value for '%s': %+v", e, rule)
	}
	//fmt.Printf("%+v\n", *rule)
}

func TestConfigGroupMultiple(t *testing.T) {
	var c Configuration
	var rule *Rule
	cfg := `g1 { r1 { test="true" } r2 { test="false" } } `

	if e := c.SetConfiguration(cfg); e != nil {
		t.Errorf("Received error for basic config: %v", e)
	}

	if len(c.Rules) != 2 {
		t.Errorf("Received unexpected number of rules: %d", len(c.Rules))
	}

	rule, ok := c.Rules["g1/r1"]
	if !ok || rule.Test != "true" {
		t.Errorf("Received unexpected rule: %+v", *rule)
	}
	if e := matchesInitial(*rule); e != nil {
		t.Errorf("Rule didn't match expected value for '%s': %+v", e, rule)
	}
	if e := matchesDefaults(*rule); e != nil {
		t.Errorf("Rule didn't match expected value for '%s': %+v", e, rule)
	}
	//fmt.Printf("%+v\n", *rule)

	rule, ok = c.Rules["g1/r2"]
	if !ok || rule.Test != "false" {
		t.Errorf("Received unexpected rule: %+v", *rule)
	}
	if e := matchesInitial(*rule); e != nil {
		t.Errorf("Rule didn't match expected value for '%s': %+v", e, rule)
	}
	if e := matchesDefaults(*rule); e != nil {
		t.Errorf("Rule didn't match expected value for '%s': %+v", e, rule)
	}
	//fmt.Printf("%+v\n", *rule)
}
