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

const delayedTickerTestEps = time.Millisecond * 75

func TestDelayedTickerS1000T100(t *testing.T) {
	if testing.Short() {
		t.Skip("Tests relying on timing should be done in a more controlled environment.")
	}

	var l [6]time.Time

	sv := time.Second * 1
	tv := time.Millisecond * 100

	dt := NewDelayedTicker()
	start := time.Now()

	// run the ticker
	dt.Start(sv, tv)
	for i := 0; i < len(l); i++ {
		l[i] = <-dt.C
	}
	dt.Stop()

	// test the values
	d := l[0].Sub(start)
	if d > (sv + delayedTickerTestEps) {
		t.Errorf("Start time out of expected range, d: %v, sv: %v, delayedTickerTestEps: %v", d, sv, delayedTickerTestEps)
	}

	for i := 1; i < len(l); i++ {
		d := l[i].Sub(l[i-1])
		if d > (tv + delayedTickerTestEps) {
			t.Errorf("Tick interval out of expected range, i: %v, d: %v, tv: %v, delayedTickerTestEps: %v", i, d, tv, delayedTickerTestEps)
		}
	}
}

func TestDelayedTickerStartStop(t *testing.T) {
	if testing.Short() {
		t.Skip("Tests relying on timing should be done in a more controlled environment.")
	}

	dt := NewDelayedTicker()

	dt.Start(0, 0)
	time.Sleep(2 * delayedTickerTestEps)
	dt.Stop()

	select {
	case <-dt.C:
	default:
		t.Errorf("No value ready")
	}

	select {
	case <-dt.C:
		t.Errorf("Still producing values")
	default:
	}

}

func TestDelayedTickerStartStopStop(t *testing.T) {
	dt := NewDelayedTicker()

	dt.Start(0, time.Second*10)
	if err := dt.Stop(); err != nil {
		t.Errorf("First Stop produced an error %v:", err)
	}

	if err := dt.Stop(); err == nil {
		t.Errorf("Second Stop produced no error %v:", err)
	}

	if err := dt.ChangeRunningInterval(time.Second * 1000); err == nil {
		t.Errorf("ChangeRunningInterval produced no error %v:", err)
	}
}

func TestDelayedTickerS0T0(t *testing.T) {
	if testing.Short() {
		t.Skip("Tests relying on timing should be done in a more controlled environment.")
	}

	var l [6]time.Time

	dt := NewDelayedTicker()

	// run the ticker
	start := time.Now()
	dt.Start(0, 0)
	for i := 0; i < len(l); i++ {
		l[i] = <-dt.C
	}
	dt.Stop()

	// test the values
	d := l[0].Sub(start)
	if d > delayedTickerTestEps {
		t.Errorf("Start time out of expected range, d: %v, sv: %v, delayedTickerTestEps: %v", d, 0, delayedTickerTestEps)
	}

	for i := 1; i < len(l); i++ {
		d := l[i].Sub(l[i-1])
		if d > (delayedTickerTestEps) {
			t.Errorf("Tick interval out of expected range, i: %v, d: %v, delayedTickerTestEps: %v", i, d, delayedTickerTestEps)
		}
	}
}

func TestDelayedTickerOneChangeSlower(t *testing.T) {
	if testing.Short() {
		t.Skip("Tests relying on timing should be done in a more controlled environment.")
	}

	var l [10]time.Time
	exp := [...]time.Duration{0,
		time.Millisecond * 100, time.Millisecond * 100,
		time.Millisecond * 100, time.Millisecond * 100,
		time.Millisecond * 100,
		time.Millisecond * 150, time.Millisecond * 150,
		time.Millisecond * 150, time.Millisecond * 150}

	//fmt.Println(len(exp))

	dt := NewDelayedTicker()

	// run the ticker
	start := time.Now()
	dt.Start(0, time.Millisecond*100)

	for i := 0; i < len(l); i++ {
		l[i] = <-dt.C
		if i == 5 {
			dt.ChangeRunningInterval(time.Millisecond * 150)
		}

	}
	dt.Stop()

	d := l[0].Sub(start)
	//fmt.Printf("d[0]: %v\n", d)
	if d > delayedTickerTestEps {
		t.Errorf("Tick interval out of expected range, d: %v, tv: %v, delayedTickerTestEps: %v", d, 0, delayedTickerTestEps)
	}

	for i := 1; i < len(l); i++ {
		d := l[i].Sub(l[i-1])
		//fmt.Printf("d[%d]: %v l: %v\n", i, d, l[i])

		if d > (exp[i] + delayedTickerTestEps) {
			t.Errorf("Tick interval out of expected range, i: %v, d: %v, tv: %v, delayedTickerTestEps: %v", i, d, exp[i], delayedTickerTestEps)
		}
	}
}

func TestDelayedTickerAFewChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("Tests relying on timing should be done in a more controlled environment.")
	}

	var l [10]time.Time
	exp := [...]time.Duration{0,
		time.Millisecond * 100, time.Millisecond * 100,
		time.Millisecond * 150, time.Millisecond * 150,
		0, 0,
		time.Millisecond * 200, time.Millisecond * 0,
		time.Millisecond * 50, 0}

	//fmt.Println(len(exp))

	dt := NewDelayedTicker()

	// run the ticker
	start := time.Now()
	dt.Start(exp[0], exp[1])

	for i := 0; i < len(l); i++ {
		l[i] = <-dt.C
		dt.ChangeRunningInterval(exp[i+1])

	}
	dt.Stop()

	d := l[0].Sub(start)
	//fmt.Printf("d[0]: %v\n", d)
	if d > delayedTickerTestEps {
		t.Errorf("Tick interval out of expected range, d: %v, tv: %v, delayedTickerTestEps: %v", d, 0, delayedTickerTestEps)
	}

	for i := 1; i < len(l); i++ {
		d := l[i].Sub(l[i-1])
		//fmt.Printf("d[%d]: %v l: %v\n", i, d, l[i])

		if d > (exp[i] + delayedTickerTestEps) {
			t.Errorf("Tick interval out of expected range, i: %v, d: %v, tv: %v, delayedTickerTestEps: %v", i, d, exp[i], delayedTickerTestEps)
		}
	}
}
