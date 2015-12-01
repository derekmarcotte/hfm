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
		log.Debug("Dispatching rule '%s'", rule.name)
		log.Debug("%s details: %+v", rule.name, rule)

		// driver gets its own copy of the rule, safe from
		// side effects later
		driver := RuleDriver{Rule: *rule, Done: ruleDone}
		go driver.Run()
	}
	log.Debug("%d goroutines - after dispatch loop.", runtime.NumGoroutine())

	for i := 0; i < len(config.Rules); i++ {
		driver := <-ruleDone
		log.Info("'%s' completed execution.  Ran for: %v\n\n", driver.Rule.name, driver.LastExecDuration)
	}

	log.Debug("%d goroutines - at the end.", runtime.NumGoroutine())
}
