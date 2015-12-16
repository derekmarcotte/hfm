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
	"bytes"
	"os/exec"
	"reflect"
	"syscall"
	"time"
)

type ExitRecord struct {
	ExecDuration time.Duration
	Error        error
	ExitStatus   int
}

type RuleDriver struct {
	Rule Rule
	Done chan *RuleDriver
	Last ExitRecord
}

func (rd *RuleDriver) resetLast() {
	rd.Last.ExecDuration = 0
	rd.Last.Error = nil
	rd.Last.ExitStatus = 0
}

func (rd *RuleDriver) handleCmdDone(value reflect.Value) {
	err := value.Interface()
	if err != nil {
		log.Error("'%s' completed with error: %v", rd.Rule.Name, err)

		ee := err.(*exec.ExitError)
		rd.Last.Error = ee

		if ws, ok := ee.Sys().(syscall.WaitStatus); ok {
			rd.Last.ExitStatus = ws.ExitStatus()
		}
	}
}

func (rd *RuleDriver) handleCmdIntTimeout(cmd *exec.Cmd) {
	log.Info("'%s' interrupt timeout exceeded, issuing interrupt.", rd.Rule.Name)
	if err := cmd.Process.Signal(syscall.SIGINT); err != nil {
		log.Error("'%s' failed to interrupt test process: %v, disabling further checks", rd.Rule.Name, err)
		rd.Rule.Status = RuleStatusDisabled
	}
}

func (rd *RuleDriver) handleCmdKillTimeout(cmd *exec.Cmd) {
	log.Warning("'%s' kill timeout exceeded, issuing kill.", rd.Rule.Name)
	if err := cmd.Process.Kill(); err != nil {
		log.Error("'%s' failed to kill test process: %v, disabling further checks", rd.Rule.Name, err)
		rd.Rule.Status = RuleStatusDisabled
	}
}

/* process any output produced by the command, get buffers ready for next run */
func (rd *RuleDriver) handleCmdBuffers(stdout *bytes.Buffer, stderr *bytes.Buffer) {
	if stdout.Len() > 0 {
		log.Info("'%s' produced output: %v", rd.Rule.Name, stdout.String())
	}
	stdout.Reset()

	if stderr.Len() > 0 {
		log.Error("'%s' produced error output: %v", rd.Rule.Name, stderr.String())
	}
	stderr.Reset()
}

func (rd *RuleDriver) handleStateChange(newState RuleStateType) {
	log.Warning("'%s' changed state to: %v", rd.Rule.Name, newState)
	rd.Rule.LastState = newState

	var changeCmd string
	if newState == RuleStateSuccess {
		changeCmd = rd.Rule.ChangeSuccess
	} else {
		changeCmd = rd.Rule.ChangeFail
	}

	if changeCmd == "" {
		return
	}

	/* XXX: may never return, oooooooo */
	cmd := exec.Command(rd.Rule.Shell, "-c", changeCmd)
	go cmd.Run()
}

/* update the state of the rule if required, take action if state or status
 * requires it
 */
func (rd *RuleDriver) updateRuleState() {
	newState := RuleStateSuccess
	switch {
	case rd.Last.Error != nil, rd.Last.ExitStatus != 0, rd.Rule.Status == RuleStatusRunOnceFail, rd.Rule.Status == RuleStatusAlwaysFail:
		if rd.Rule.Status != RuleStatusAlwaysSuccess {
			newState = RuleStateFail
		}
	}

	/* if the state has changed, or is an Always */
	switch {
	case rd.Rule.LastState != newState, rd.Rule.Status == RuleStatusAlwaysFail, rd.Rule.Status == RuleStatusAlwaysSuccess:
		rd.handleStateChange(newState)
	}
}

func (rd *RuleDriver) updateRuleStatus() {
	switch rd.Rule.Status {
	case RuleStatusRunOnce, RuleStatusRunOnceFail, RuleStatusRunOnceSuccess:
		rd.Rule.Status = RuleStatusDisabled
	}
}

func (rd *RuleDriver) Run() {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmdDone := make(chan error)

	// the code isnt't very readale otherwise
	timeoutInt := time.Duration(rd.Rule.TimeoutInt * float64(time.Second))
	timeoutKill := time.Duration(rd.Rule.TimeoutKill * float64(time.Second))

	for rd.Rule.Status != RuleStatusDisabled {
		start := time.Now()
		log.Debug("'%s' starting at %v...", rd.Rule.Name, start)

		rd.resetLast()

		cmd := exec.Command(rd.Rule.Shell, "-c", rd.Rule.Test)

		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		cases := make([]reflect.SelectCase, 3)
		cases[0] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(cmdDone)}
		cases[1] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(time.After(timeoutKill))}
		cases[2] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(time.After(timeoutInt))}

		if err := cmd.Start(); err != nil {
			log.Error("'%s' failed to start: %v", rd.Rule.Name, err)
			rd.Done <- rd
			return
		}

		go func() {
			cmdDone <- cmd.Wait()
		}()

		/* while we still are expecting events to listen to */
		for len(cases) > 0 {
			i, value, _ := reflect.Select(cases)

			switch i {
			case 0:
				rd.handleCmdDone(value)
			case 1:
				rd.handleCmdKillTimeout(cmd)
			case 2:
				rd.handleCmdIntTimeout(cmd)
			}

			/* each of these happens once, and are ordered
			 * by index, with done being the first in the set
			 */
			switch i {
			case 0:
				cases = nil
			case 1, 2:
				cases = cases[:i]
			}
		}
		rd.Last.ExecDuration = time.Since(start)

		rd.handleCmdBuffers(&stdout, &stderr)

		rd.updateRuleState()

		rd.updateRuleStatus()

		/* I don't think we should allow back-log
		 *   if the test takes longer than the interval
		 *   we'll just run it in a tight loop
		 * Maybe there's a more graceful way to do this, but
		 *   this is fairly cheap to implement
		 *   although tests will not execute on exactly interval
		 */
		next := time.Duration(rd.Rule.Interval*float64(time.Second)) - rd.Last.ExecDuration
		if rd.Rule.Status != RuleStatusDisabled && next > 0 {
			time.Sleep(next)
		}
	}

	rd.Done <- rd
}
