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

/* tightly coupled to the the logging interface ! */
import "github.com/op/go-logging"

func init() {
	log.SetBackend(logging.AddModuleLevel(logging.InitForTesting(logging.NOTICE)))
}

func TestDriverRunsOne(t *testing.T) {
	var c Configuration

	cfg := `runs=1; test="true"`

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

	cfg := `status=disabled; test="false"`

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

	cfg := `runs=1; timeout_int=10ms; test="sleep"; test_arguments="2"`

	c.SetConfiguration(cfg)

	ruleDone := make(chan *RuleDriver)

	driver := RuleDriver{Rule: *c.Rules["default"], Done: ruleDone}

	s := time.Now()
	go driver.Run()
	<-ruleDone

	// was getting intermittant failures at 11, and 12
	e := time.Since(s)
	if e > 13*time.Millisecond || e < 9*time.Millisecond {
		t.Errorf("took %v to finish", e)
	}
}

func TestDriverKill(t *testing.T) {
	var c Configuration

	cfg := `runs=1; timeout_kill=10ms; test="/bin/sh"; test_arguments=[ "-c", "trap '' SIGINT; sleep 2" ]`

	c.SetConfiguration(cfg)

	ruleDone := make(chan *RuleDriver)

	driver := RuleDriver{Rule: *c.Rules["default"], Done: ruleDone}

	s := time.Now()
	go driver.Run()
	<-ruleDone

	// was getting intermittant failures at 11, and 12
	e := time.Since(s)
	if e > 13*time.Millisecond || e < 9*time.Millisecond {
		t.Errorf("took %v to finish", e)
	}

	// fmt.Printf("error: %v, status: %v\n", driver.Last.Error, driver.Last.ExitStatus)
}

func TestDriverExit1(t *testing.T) {
	var c Configuration

	cfg := `runs=1; test="false"`

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

	cfg := `runs=1; test="false"; change_fail="rm"; change_fail_arguments = "` + f.Name() + `"`

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

	cfg := `
runs=1;
test="true";
change_success="rm";
change_success_arguments = "` + f.Name() + `"`

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

func TestDriverDebounceFail(t *testing.T) {
	var c Configuration

	cf, err := ioutil.TempFile("", "hfm-test-suite-debounce-fail-runs-")
	if cf == nil || err != nil {
		t.Errorf("Could not create temp file: %v", err)
	}
	defer os.Remove(cf.Name())

	sf, err := ioutil.TempFile("", "hfm-test-suite-debounce-fail-successes-")
	if cf == nil || err != nil {
		t.Errorf("Could not create temp file: %v", err)
	}
	defer os.Remove(sf.Name())

	ff, err := ioutil.TempFile("", "hfm-test-suite-debounce-fail-fails-")
	if cf == nil || err != nil {
		t.Errorf("Could not create temp file: %v", err)
	}
	defer os.Remove(ff.Name())

	cfg := `
interval=15ms
runs=10
test="/bin/sh"
test_arguments = [ "-c", <<EOD
CF="` + cf.Name() + `"
if [ -f "$CF" ]; then
	C=$(cat "$CF")
	if [ -z "$C" ]; then
		C=0
	fi
	C=$(echo "$C+1" | bc)
else
	C=1
fi
echo "$C" > "$CF"

if [ $C -eq 1 ]; then
	# change to success
	exit 0
elif [ $C -eq 2 ]; then
	# blip
	exit 1
elif [ $C -eq 3 ]; then
	# still good
	exit 0
elif [ $C -eq 4 ]; then
	# blip
	exit 1
elif [ $C -eq 5 ]; then
	# longer blip
	exit 1
elif [ $C -eq 6 ]; then
	# still good
	exit 0
elif [ $C -eq 7 ]; then
	exit 1
elif [ $C -eq 8 ]; then
	exit 1
elif [ $C -eq 9 ]; then
	# three in a row should do it
	exit 1
elif [ $C -eq 10 ]; then
	# back to success
	# last run, good juju
	sync; sync; sync;
	exit 0
fi
EOD
]
change_success="/bin/sh"
change_success_arguments = [ "-c", <<EOD
CF="` + cf.Name() + `"
SF="` + sf.Name() + `"
cat "$CF" >> "$SF"
EOD
]
change_fail="/bin/sh"
change_fail_debounce=3
change_fail_arguments = [ "-c", <<EOD
CF="` + cf.Name() + `"
FF="` + ff.Name() + `"
cat "$CF" >> "$FF"
EOD
]
`

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

	buf, err := ioutil.ReadFile(cf.Name())
	if err != nil {
		t.Errorf("Could not read count file: %+v\n", err)
	} else if tmp := string(buf[:]); "10\n" != tmp {
		t.Errorf("Expected count file contents of '10\\n', recieved: %+v\n", tmp)
	}

	buf, err = ioutil.ReadFile(sf.Name())
	if err != nil {
		t.Errorf("Could not read success file: %+v\n", err)
	} else if tmp := string(buf[:]); "1\n10\n" != tmp {
		t.Errorf("Expected success file contents of '1\\n10\\n', recieved: %+v\n", tmp)
	}

	buf, err = ioutil.ReadFile(ff.Name())
	if err != nil {
		t.Errorf("Could not read fail file: %+v\n", err)
	} else if tmp := string(buf[:]); "9\n" != tmp {
		t.Errorf("Expected fail file contents of '9\\n', recieved: %+v\n", tmp)
	}

}

func TestDriverDebounceSuccess(t *testing.T) {
	var c Configuration

	cf, err := ioutil.TempFile("", "hfm-test-suite-debounce-success-runs-")
	if cf == nil || err != nil {
		t.Errorf("Could not create temp file: %v", err)
	}
	defer os.Remove(cf.Name())

	sf, err := ioutil.TempFile("", "hfm-test-suite-debounce-success-successes-")
	if cf == nil || err != nil {
		t.Errorf("Could not create temp file: %v", err)
	}
	defer os.Remove(sf.Name())

	ff, err := ioutil.TempFile("", "hfm-test-suite-debounce-success-fails-")
	if cf == nil || err != nil {
		t.Errorf("Could not create temp file: %v", err)
	}
	defer os.Remove(ff.Name())

	cfg := `
interval=15ms
runs=10
test="/bin/sh"
test_arguments = [ "-c", <<EOD
CF="` + cf.Name() + `"
if [ -f "$CF" ]; then
	C=$(cat "$CF")
	if [ -z "$C" ]; then
		C=0
	fi
	C=$(echo "$C+1" | bc)
else
	C=1
fi
echo "$C" > "$CF"

if [ $C -eq 1 ]; then
	# change to fail
	exit 1
elif [ $C -eq 2 ]; then
	# blip
	exit 0
elif [ $C -eq 3 ]; then
	# still good
	exit 1
elif [ $C -eq 4 ]; then
	# blip
	exit 0
elif [ $C -eq 5 ]; then
	# longer blip
	exit 0
elif [ $C -eq 6 ]; then
	# still good
	exit 1
elif [ $C -eq 7 ]; then
	exit 0
elif [ $C -eq 8 ]; then
	exit 0
elif [ $C -eq 9 ]; then
	# three in a row should do it
	exit 0
elif [ $C -eq 10 ]; then
	# back to fail
	# last run, good juju
	sync; sync; sync;
	exit 1
fi
EOD
]
change_success_debounce=3
change_success="/bin/sh"
change_success_arguments = [ "-c", <<EOD
CF="` + cf.Name() + `"
SF="` + sf.Name() + `"
cat "$CF" >> "$SF"
EOD
]
change_fail="/bin/sh"
change_fail_arguments = [ "-c", <<EOD
CF="` + cf.Name() + `"
FF="` + ff.Name() + `"
cat "$CF" >> "$FF"
EOD
]
`

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

	buf, err := ioutil.ReadFile(cf.Name())
	if err != nil {
		t.Errorf("Could not read count file: %+v\n", err)
	} else if tmp := string(buf[:]); "10\n" != tmp {
		t.Errorf("Expected count file contents of '10\\n', recieved: %+v\n", tmp)
	}

	buf, err = ioutil.ReadFile(ff.Name())
	if err != nil {
		t.Errorf("Could not read fail file: %+v\n", err)
	} else if tmp := string(buf[:]); "1\n10\n" != tmp {
		t.Errorf("Expected fail file contents of '1\\n10\\n', recieved: %+v\n", tmp)
	}

	buf, err = ioutil.ReadFile(sf.Name())
	if err != nil {
		t.Errorf("Could not read success file: %+v\n", err)
	} else if tmp := string(buf[:]); "9\n" != tmp {
		t.Errorf("Expected success file contents of '9\\n', recieved: %+v\n", tmp)
	}

}
