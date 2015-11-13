package main

/* stdlib includes */
import (
	"bytes"
	"os/exec"
	"reflect"
	"syscall"
	"time"
)

type RuleDriver struct {
	rule         *Rule
	done         chan *RuleDriver
	execDuration time.Duration
}

func (rd *RuleDriver) run() {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.Command(rd.rule.shell, "-c", rd.rule.test)

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	cmdDone := make(chan error)

	start := time.Now()
	log.Debug("'%s' starting...", rd.rule.name)
	if err := cmd.Start(); err != nil {
		log.Error("'%s' failed to start: %v", rd.rule.name, err)
		rd.done <- rd
		return
	}

	go func() {
		cmdDone <- cmd.Wait()
	}()

	// the code isnt't very readale otherwise
	timeoutInt := time.Duration(rd.rule.timeoutInt * float64(time.Second))
	timeoutKill := time.Duration(rd.rule.timeoutKill * float64(time.Second))

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
				log.Error("'%s' completed with error: %v", rd.rule.name, err)
			}
		case 1:
			log.Warning("'%s' kill timeout exceeded, issuing kill.", rd.rule.name)
			if err := cmd.Process.Kill(); err != nil {
				log.Error("'%s' failed to kill test process: %v", rd.rule.name, err)
			}
		case 2:
			log.Info("'%s' interrupt timeout exceeded, issuing interrupt.", rd.rule.name)
			if err := cmd.Process.Signal(syscall.SIGINT); err != nil {
				log.Error("'%s' failed to interrupt test process: %v", rd.rule.name, err)
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
	rd.execDuration = time.Since(start)

	if stdout.Len() > 0 {
		log.Info("'%s' produced output: %v", rd.rule.name, stdout.String())
	}

	if stderr.Len() > 0 {
		log.Error("'%s' produced error output: %v", rd.rule.name, stderr.String())
	}

	rd.done <- rd
}
