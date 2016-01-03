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
	"fmt"
	"log/syslog"
	"os"
	"path"
	"runtime"
	"strings"
	"time"
)

/* external includes */
import "github.com/op/go-logging"

/* definitions */

type LogConfiguration struct {
	Where    string
	Facility string
}

/* meat */
/* dependancy injection is for another day */
var log = logging.MustGetLogger(path.Base(os.Args[0]))

func configureLogging(conf LogConfiguration) error {
	conf.Where = strings.ToLower(conf.Where)
	switch conf.Where {
	case "syslog":
	case "stderr":
		return nil
	default:
		return fmt.Errorf("Invalid log location, must be one of {stderr, syslog}\n")
	}

	facilityList := map[string]syslog.Priority{
		"kern":     syslog.LOG_KERN,
		"user":     syslog.LOG_USER,
		"mail":     syslog.LOG_MAIL,
		"daemon":   syslog.LOG_DAEMON,
		"auth":     syslog.LOG_AUTH,
		"syslog":   syslog.LOG_SYSLOG,
		"lpr":      syslog.LOG_LPR,
		"news":     syslog.LOG_NEWS,
		"uucp":     syslog.LOG_UUCP,
		"cron":     syslog.LOG_CRON,
		"authpriv": syslog.LOG_AUTHPRIV,
		"ftp":      syslog.LOG_FTP,
		"local0":   syslog.LOG_LOCAL0,
		"local1":   syslog.LOG_LOCAL1,
		"local2":   syslog.LOG_LOCAL2,
		"local3":   syslog.LOG_LOCAL3,
		"local4":   syslog.LOG_LOCAL4,
		"local5":   syslog.LOG_LOCAL5,
		"local6":   syslog.LOG_LOCAL6,
		"local7":   syslog.LOG_LOCAL7,
	}

	conf.Facility = strings.ToLower(conf.Facility)

	f, ok := facilityList[conf.Facility]
	if !ok {
		return fmt.Errorf("Invalid syslog facility")
	}

	be, _ := logging.NewSyslogBackendPriority(path.Base(os.Args[0]), f)
	log.SetBackend(logging.AddModuleLevel(be))

	return nil
}

func main() {
	var configPath string
	var config Configuration

	var lc LogConfiguration

	flag.StringVar(&configPath, "config", "etc/hfm.conf", "Configuration file path")
	flag.StringVar(&lc.Where, "log", "stderr", "Where to log {stderr, syslog}")
	flag.StringVar(&lc.Facility, "facility", "local0", "Log facility (when -log set to syslog) {local0-9, user, etc}")
	flag.Parse()

	if e := configureLogging(lc); e != nil {
		fmt.Printf("Could not configure logging: %v", e)
		panic(e)
	}

	if e := config.LoadConfiguration(configPath); e != nil {
		fmt.Printf("Could not load configuration file %v: %+v", configPath, e)
		panic(e)
	}

	ruleDone := make(chan *RuleDriver)

	/* close enough for most applications */
	appInstance := time.Now().UnixNano()

	log.Info("Loaded %d rules.", len(config.Rules))
	log.Debug("%d goroutines - before main dispatch loop.", runtime.NumGoroutine())
	for _, rule := range config.Rules {
		log.Debug("Dispatching rule '%s'", rule.Name)
		log.Debug("%s details: %+v", rule.Name, rule)

		// driver gets its own copy of the rule, safe from
		// side effects later
		driver := RuleDriver{Rule: *rule, Done: ruleDone, AppInstance: appInstance}
		go driver.Run()
	}
	log.Debug("%d goroutines - after dispatch loop.", runtime.NumGoroutine())

	for i := 0; i < len(config.Rules); i++ {
		driver := <-ruleDone
		log.Info("'%s' completed execution.  Ran for: %v\n\n", driver.Rule.Name, driver.Last.ExecDuration)
	}

	log.Debug("%d goroutines - at the end.", runtime.NumGoroutine())
}
