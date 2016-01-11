/*
 * Copyright (c) 2015, Derek Marcotte
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are
 * met:
 *
 * 1. Redistributions of source code must retain the above copyright
 * notice, this list of conditions and the following disclaimer.
 *
 * 2. Redistributions in binary form must reproduce the above copyright
 * notice, this list of conditions and the following disclaimer in the
 * documentation and/or other materials provided with the distribution.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
 * "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
 * LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
 * A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
 * HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
 * SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
 * LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
 * DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
 * THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 * (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
 * OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

package main

import "testing"
import "time"
import "io/ioutil"
import "os"

//import "fmt"

/* tightly coupled to the the logging interface ! */
import "github.com/op/go-logging"

func init() {
	log.SetBackend(logging.AddModuleLevel(logging.InitForTesting(logging.NOTICE)))
}

func TestDriverStatusRunOnce(t *testing.T) {
	var c Configuration

	cfg := `status=run-once; test="true"`

	c.SetConfiguration(cfg)

	ruleDone := make(chan *RuleDriver)

	driver := RuleDriver{Rule: *c.Rules["default"], Done: ruleDone}
	go driver.Run()

	<-ruleDone

	// fmt.Printf("error: %v, status: %v\n", driver.Last.Error, driver.Last.ExitStatus)
	// if we get here, the test passes
}

func TestDriverStatusDisabled(t *testing.T) {
	var c Configuration

	cfg := `status=disabled; test="exit 1"`

	c.SetConfiguration(cfg)

	ruleDone := make(chan *RuleDriver)

	driver := RuleDriver{Rule: *c.Rules["default"], Done: ruleDone}
	go driver.Run()

	driver = *(<-ruleDone)

	if driver.Last.ExitStatus != 0 {
		t.Errorf("Expected exit status 0, recevied: %+v\n", driver.Last.ExitStatus)
	}

	// fmt.Printf("error: %+v", driver.Last.Error)
}

func TestDriverInterrupt(t *testing.T) {
	var c Configuration

	cfg := `status=run-once; timeout_int=10ms; test="sleep 2"`

	c.SetConfiguration(cfg)

	ruleDone := make(chan *RuleDriver)

	driver := RuleDriver{Rule: *c.Rules["default"], Done: ruleDone}

	s := time.Now()
	go driver.Run()
	<-ruleDone

	e := time.Since(s)
	if e > 12*time.Millisecond {
		t.Errorf("took %v to finish", e)
	}
}

func TestDriverKill(t *testing.T) {
	var c Configuration

	cfg := `status=run-once; timeout_kill=10ms; test="trap '' SIGINT; sleep 2"`

	c.SetConfiguration(cfg)

	ruleDone := make(chan *RuleDriver)

	driver := RuleDriver{Rule: *c.Rules["default"], Done: ruleDone}

	s := time.Now()
	go driver.Run()
	<-ruleDone

	e := time.Since(s)
	if e > 12*time.Millisecond {
		t.Errorf("took %v to finish", e)
	}

	// fmt.Printf("error: %v, status: %v\n", driver.Last.Error, driver.Last.ExitStatus)
}

func TestDriverExit1(t *testing.T) {
	var c Configuration

	cfg := `status=run-once; test="false"`

	c.SetConfiguration(cfg)

	ruleDone := make(chan *RuleDriver)

	driver := RuleDriver{Rule: *c.Rules["default"], Done: ruleDone}
	go driver.Run()

	driver = *(<-ruleDone)

	// fmt.Printf("error: %+v", driver.Last.Error)

	if driver.Last.ExitStatus != 1 {
		t.Errorf("Expected exit status 1, recevied: %+v\n", driver.Last.ExitStatus)
	}

}

func TestDriverChangeFail(t *testing.T) {
	var c Configuration

	f, err := ioutil.TempFile("", "hfm-test-suite-")
	if f == nil || err != nil {
		t.Errorf("Could not create temp file: %v", err)
	}

	exists := true

	cfg := `status=run-once; test="false"; change_fail="rm"; change_success_arguments = "` + f.Name() + `"`

	c.SetConfiguration(cfg)

	ruleDone := make(chan *RuleDriver)

	driver := RuleDriver{Rule: *c.Rules["default"], Done: ruleDone}
	go driver.Run()

	driver = *(<-ruleDone)

	// fmt.Printf("error: %+v", driver.Last.Error)

	if driver.Last.ExitStatus != 1 {
		t.Errorf("Expected exit status 1, recevied: %+v\n", driver.Last.ExitStatus)
	}

	/* give enough time for the async call to complete, 50ms is totally arbitrary */
	d, _ := time.ParseDuration("50ms")
	time.Sleep(d)

	if _, err := os.Stat(f.Name()); os.IsNotExist(err) {
		exists = false
	}

	if exists {
		t.Errorf("Tempfile %s exists, expected to have been removed.  Please remove manually.\n", f.Name())
	}

}

func TestDriverChangeSuccess(t *testing.T) {
	var c Configuration

	f, err := ioutil.TempFile("", "hfm-test-suite-")
	if f == nil || err != nil {
		t.Errorf("Could not create temp file: %v", err)
	}

	exists := true

	cfg := `status=run-once; test="true"; change_success="rm"; change_success_arguments = "` + f.Name() + `"`

	c.SetConfiguration(cfg)

	ruleDone := make(chan *RuleDriver)

	driver := RuleDriver{Rule: *c.Rules["default"], Done: ruleDone}
	go driver.Run()

	driver = *(<-ruleDone)

	// fmt.Printf("error: %+v", driver.Last.Error)

	if driver.Last.ExitStatus != 0 {
		t.Errorf("Expected exit status 0, recevied: %+v\n", driver.Last.ExitStatus)
	}

	/* give enough time for the async call to complete, 50ms is totally arbitrary */
	d, _ := time.ParseDuration("50ms")
	time.Sleep(d)

	if _, err := os.Stat(f.Name()); os.IsNotExist(err) {
		exists = false
	}

	if exists {
		t.Errorf("Tempfile %s exists, expected to have been removed.  Please remove manually.\n", f.Name())
	}

}
