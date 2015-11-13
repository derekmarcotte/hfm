package main

/* stdlib includes */
import (
	"flag"
	"fmt"
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

	fmt.Println("Rules")

	log.Info("Loaded %d rules.", len(config.rules))
	log.Debug("%d goroutines - before main dispatch loop.", runtime.NumGoroutine())
	for _, rule := range config.rules {
		log.Debug("Dispatching rule '%s'", rule.name)
		log.Debug("%s details: %+v", rule.name, rule)

		go func(rule *Rule) {
			driver := RuleDriver{rule: rule, done: ruleDone}

			driver.run()
		}(rule)
	}
	log.Debug("%d goroutines - after dispatch loop.", runtime.NumGoroutine())

	for i := 0; i < len(config.rules); i++ {
		driver := <-ruleDone

		log.Info("'%s' completed execution.  Ran for: %v\n\n", driver.rule.name, driver.execDuration)
	}

	log.Debug("%d goroutines - at the end.", runtime.NumGoroutine())
}
