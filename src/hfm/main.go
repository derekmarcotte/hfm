package main

/* stdlib includes */
import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"syscall"
	"time"
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

	ruleDone := make(chan string)

	fmt.Println("Rules")

	var end time.Time

	startTimes := make(map[string]time.Time)

	log.Info("Loaded %d rules.", len(config.rules))
	log.Debug("%d goroutines - before main dispatch loop.", runtime.NumGoroutine())
	for _, rule := range config.rules {
		log.Debug("Dispatching rule '%s'", rule.name)
		log.Debug("%s details: %+v", rule.name, rule)

		go func(rule *Rule) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			cmd := exec.Command(rule.shell, "-c", rule.test)

			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			cmdDone := make(chan error)

			startTimes[rule.name] = time.Now()
			log.Debug("'%s' starting...", rule.name)
			if err := cmd.Start(); err != nil {
				log.Error("'%s' failed to start: %v", rule.name, err)
				ruleDone <- rule.name
				return
			}

			go func() {
				cmdDone <- cmd.Wait()
			}()

			// the code isnt't very readale otherwise
			timeoutInt := time.Duration(rule.timeoutInt * float64(time.Second))
			timeoutKill := time.Duration(rule.timeoutKill * float64(time.Second))

			cases := make([]reflect.SelectCase, 3)
			cases[0] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(cmdDone)}
			cases[1] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(time.After(timeoutKill))}
			cases[2] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(time.After(timeoutInt))}

			for len(cases) > 0 {
				i, value, _ := reflect.Select(cases)

				switch i {
				case 0:
					err := value.Interface()
					if err != nil {
						log.Error("'%s' completed with error: %v", rule.name, err)
					}
				case 1:
					log.Warning("'%s' kill timeout exceeded, issuing kill.", rule.name)
					if err := cmd.Process.Kill(); err != nil {
						log.Error("'%s' failed to kill test process: %v", rule.name, err)
					}
				case 2:
					log.Info("'%s' interrupt timeout exceeded, issuing interrupt.", rule.name)
					if err := cmd.Process.Signal(syscall.SIGINT); err != nil {
						log.Error("'%s' failed to interrupt test process: %v", rule.name, err)
					}
				}

				switch i {
				case 0:
					cases = nil
				case 1, 2:
					cases = cases[:i]
				}
			}

			if stdout.Len() > 0 {
				log.Info("'%s' produced output: %v", rule.name, stdout.String())
			}

			if stderr.Len() > 0 {
				log.Error("'%s' produced error output: %v", rule.name, stderr.String())
			}

			ruleDone <- rule.name
		}(rule)
	}
	log.Debug("%d goroutines - after dispatch loop.", runtime.NumGoroutine())

	for i := 0; i < len(config.rules); i++ {
		ruleName := <-ruleDone
		end = time.Now()

		log.Info("'%s' completed execution.  Ran for: %v\n\n", ruleName, end.Sub(startTimes[ruleName]))
	}

	log.Debug("%d goroutines - at the end.", runtime.NumGoroutine())
}
