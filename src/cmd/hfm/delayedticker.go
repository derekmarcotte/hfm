/*
 * Copyright (c) 2016, Derek Marcotte
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
	"fmt"
	"time"
)

type DelayedTicker struct {
	// emit a time when the next tick occurs
	C chan time.Time

	// signal the loop to quit
	quit       chan struct{}
	loopStatus chan struct{}

	running bool
	stopped bool

	// used for internal bookeeping
	lastTick time.Time
}

func NewDelayedTicker() *DelayedTicker {
	var t DelayedTicker

	t.C = make(chan time.Time, 1)
	t.quit = make(chan struct{})
	t.loopStatus = make(chan struct{})

	return &t
}

/* Start the ticker emitting messages on C */
func (t *DelayedTicker) Start(delay time.Duration, interval time.Duration) error {
	if t.stopped {
		return fmt.Errorf("Cannot start a stopped DelayedTicker.")
	} else if t.running {
		return fmt.Errorf("DelayedTicker already running.")
	}

	t.lastTick = time.Now()

	go t.loop(delay, interval)
	<-t.loopStatus

	return nil
}

func (t *DelayedTicker) loop(delay time.Duration, interval time.Duration) {
	// we want the most accurate delay we can get, do this before blocking
	start := time.NewTimer(delay)

	t.running = true
	t.loopStatus <- struct{}{}

	// need a valid ticker reference, so we don't get nil reference
	// in the select
	ticker := time.NewTicker(time.Nanosecond)
	ticker.Stop()

	// this gets close to a spin at 1/3ms, be careful
	// on FreeBSD at least this is limited to 1 tick (1000ms by default)
	if interval == 0 {
		interval = time.Microsecond * 333
	}

timerEvents:
	for {
		select {
		case <-t.quit:
			break timerEvents
		case <-start.C:
			// only start the ticker after the initial delay
			ticker = time.NewTicker(interval)
		case <-ticker.C:
		}

		// only emit the tick if someone has consumed it, otherwise discard
		if len(t.C) == 0 {
			t.lastTick = time.Now()
			t.C <- t.lastTick
		}
	}

	start.Stop()
	ticker.Stop()

	t.running = false
	t.loopStatus <- struct{}{}
}

/* Stop emitting messages and close C */
func (t *DelayedTicker) Stop() error {
	if !t.running {
		return fmt.Errorf("DelayedTicker not running")
	}

	t.stopped = true
	t.quit <- struct{}{}
	<-t.loopStatus

	close(t.C)
	close(t.quit)
	close(t.loopStatus)

	return nil
}

/* Kill event loop with old parameters, reconfigure */
func (t *DelayedTicker) ChangeRunningInterval(interval time.Duration) error {
	if !t.running {
		return fmt.Errorf("DelayedTicker not running")
	}

	t.quit <- struct{}{}
	<-t.loopStatus

	go t.loop(interval-time.Since(t.lastTick), interval)
	<-t.loopStatus

	return nil
}
