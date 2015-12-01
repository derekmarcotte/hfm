package main

import "testing"

/* tightly coupled to the the logging interface ! */
import "github.com/op/go-logging"

func init() {
	log.SetBackend(logging.AddModuleLevel(logging.InitForTesting(logging.NOTICE)))
}

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
