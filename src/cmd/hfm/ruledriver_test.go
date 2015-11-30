package main

import "testing"

//import "reflect"
//import "errors"

func TestStatusRunOnce(t *testing.T) {
	var c Configuration

	cfg := `status=run-once; test="true"`

	c.SetConfiguration(cfg)

	ruleDone := make(chan *RuleDriver)

	driver := RuleDriver{Rule: *c.Rules["default"], Done: ruleDone}
	go driver.Run()

	<-ruleDone
}

func TestInterrupt(t *testing.T) {
	var c Configuration

	cfg := `status=run-once; timeout_int=10ms; test="sleep 10"`

	c.SetConfiguration(cfg)

	ruleDone := make(chan *RuleDriver)

	driver := RuleDriver{Rule: *c.Rules["default"], Done: ruleDone}
	go driver.Run()

	<-ruleDone
}

func TestKill(t *testing.T) {
	var c Configuration

	cfg := `status=run-once; timeout_int=10ms; test="trap '' SIGINT; sleep 10"`

	c.SetConfiguration(cfg)

	ruleDone := make(chan *RuleDriver)

	driver := RuleDriver{Rule: *c.Rules["default"], Done: ruleDone}
	go driver.Run()

	<-ruleDone
}
