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

/* stdlib includes */
import (
	"flag"
	"os"
	"runtime"
)

/* external includes */
import "github.com/op/go-logging"

/* definitions */

/* meat */
/* dependancy injection is for another day */
var log = logging.MustGetLogger(os.Args[0])

func main() {
	var configPath string
	var config Configuration

	flag.StringVar(&configPath, "config", "etc/hfm.conf", "Configuration file path")
	flag.Parse()

	if e := config.LoadConfiguration(configPath); e != nil {
		log.Error("Could not load configuration file %v: %+v", configPath, e)
		panic(e)
	}

	ruleDone := make(chan *RuleDriver)

	log.Info("Loaded %d rules.", len(config.Rules))
	log.Debug("%d goroutines - before main dispatch loop.", runtime.NumGoroutine())
	for _, rule := range config.Rules {
		log.Debug("Dispatching rule '%s'", rule.Name)
		log.Debug("%s details: %+v", rule.Name, rule)

		// driver gets its own copy of the rule, safe from
		// side effects later
		driver := RuleDriver{Rule: *rule, Done: ruleDone}
		go driver.Run()
	}
	log.Debug("%d goroutines - after dispatch loop.", runtime.NumGoroutine())

	for i := 0; i < len(config.Rules); i++ {
		driver := <-ruleDone
		log.Info("'%s' completed execution.  Ran for: %v\n\n", driver.Rule.Name, driver.Last.ExecDuration)
	}

	log.Debug("%d goroutines - at the end.", runtime.NumGoroutine())
}
