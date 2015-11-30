package main

/* stdlib includes */
import (
	"bytes"
	"fmt"
	"os/exec"
	"reflect"
	"syscall"
	"time"
)

type RuleDriver struct {
	Rule             Rule
	Done             chan *RuleDriver
	Log              chan LogMessage
	LastExecDuration time.Duration
	LastError        error
}

func (rd *RuleDriver) Run() {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmdDone := make(chan error)

	// the code isnt't very readale otherwise
	timeoutInt := time.Duration(rd.Rule.timeoutInt * float64(time.Second))
	timeoutKill := time.Duration(rd.Rule.timeoutKill * float64(time.Second))

	for rd.Rule.status != RuleStatusDisabled {
		start := time.Now()

		rd.Log <- LogMessage{logging.DEBUG, fmt.Sprintf("'%s' starting at %v...", rd.Rule.name, start)}

		stdout.Reset()
		stderr.Reset()

		cmd := exec.Command(rd.Rule.shell, "-c", rd.Rule.test)

		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		cases := make([]reflect.SelectCase, 3)
		cases[0] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(cmdDone)}
		cases[1] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(time.After(timeoutKill))}
		cases[2] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(time.After(timeoutInt))}

		if err := cmd.Start(); err != nil {
			log.Error("'%s' failed to start: %v", rd.Rule.name, err)
			rd.Done <- rd
			return
		}

		go func() {
			cmdDone <- cmd.Wait()
		}()

		for len(cases) > 0 {
			i, value, _ := reflect.Select(cases)

			switch i {
			case 0:
				err := value.Interface()
				if err != nil {
					log.Error("'%s' completed with error: %v", rd.Rule.name, err)
					if _, ok := err.(error); !ok {
						rd.LastError = err.(error)
					}
				}
			case 1:
				log.Warning("'%s' kill timeout exceeded, issuing kill.", rd.Rule.name)
				if err := cmd.Process.Kill(); err != nil {
					log.Error("'%s' failed to kill test process: %v, disabling further checks", rd.Rule.name, err)
					rd.Rule.status = RuleStatusDisabled
				}
			case 2:
				log.Info("'%s' interrupt timeout exceeded, issuing interrupt.", rd.Rule.name)
				if err := cmd.Process.Signal(syscall.SIGINT); err != nil {
					log.Error("'%s' failed to interrupt test process: %v, disabling further checks", rd.Rule.name, err)
					rd.Rule.status = RuleStatusDisabled
				}
			}

			/* each of these happens once, and are ordered
			 * by index
			 */
			switch i {
			case 0:
				cases = nil
			case 1, 2:
				cases = cases[:i]
			}
		}

		if stdout.Len() > 0 {
			log.Info("'%s' produced output: %v", rd.Rule.name, stdout.String())
		}

		if stderr.Len() > 0 {
			log.Error("'%s' produced error output: %v", rd.Rule.name, stderr.String())
		}

		switch rd.Rule.status {
		case RuleStatusRunOnce, RuleStatusRunOnceFail, RuleStatusRunOnceSuccess:
			rd.Rule.status = RuleStatusDisabled
		}

		/* I don't think we should allow back-log
		 *   if the test takes longer than the interval
		 *   we'll just run it in a tight loop
		 * Maybe there's a more graceful way to do this, but
		 *   this is fairly cheap to implement
		 *   although tests will not execute on exactly interval
		 */
		rd.LastExecDuration = time.Since(start)
		next := time.Duration(rd.Rule.interval*float64(time.Second)) - rd.LastExecDuration
		if rd.Rule.status != RuleStatusDisabled && next > 0 {
			time.Sleep(next)
		}
	}

	rd.Done <- rd
}
