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

func TestMainScheduleEmpty(t *testing.T) {
	var c Configuration

	delays, schedule := scheduleRules(c.RulesOrder, &c.Rules)

	if len(delays) != 0 {
		t.Errorf("Received non-empty delay set: %+v", delays)
	}

	if len(schedule) != 0 {
		t.Errorf("Received non-empty schedule set: %+v", schedule)
	}
}

func TestMainScheduleDefault(t *testing.T) {
	var c Configuration
	c.SetConfiguration(`test = "true"`)

	delays, schedule := scheduleRules(c.RulesOrder, &c.Rules)

	if len(delays) != 1 {
		t.Errorf("Received larger than expected delay set: %+v", delays)
	}

	if len(schedule) != 1 {
		t.Errorf("Received larger than expected schedule set: %+v", delays)
	}

	if delays[0] != 0 {
		t.Errorf("Received unexpected delay bucket: %+v", delays[0])
	}

	if schedule[delays[0]][0].Test != "true" {
		t.Errorf("Received unexpected schedule for bucket %+v: %+v", delays[0], *schedule[delays[0]][0])
	}

}

func TestMainScheduleDefault500ms(t *testing.T) {
	var c Configuration
	c.SetConfiguration(`start_delay = 500ms; test = "true"`)

	delays, schedule := scheduleRules(c.RulesOrder, &c.Rules)

	if len(delays) != 1 {
		t.Errorf("Received larger than expected delay set: %+v", delays)
	}

	if len(schedule) != 1 {
		t.Errorf("Received larger than expected schedule set: %+v", delays)
	}

	if delays[0] != 0.5 {
		t.Errorf("Received unexpected delay bucket: %+v", delays[0])
	}

	if schedule[delays[0]][0].Test != "true" {
		t.Errorf("Received unexpected schedule for bucket %+v: %+v", delays[0], *schedule[delays[0]][0])
	}

}

func TestMainScheduleInheritDefault(t *testing.T) {
	var c Configuration
	c.SetConfiguration(`start_delay = 500ms; g1 { t1 { test = "true" } }`)

	delays, schedule := scheduleRules(c.RulesOrder, &c.Rules)

	if len(delays) != 1 {
		t.Errorf("Received larger than expected delay set: %+v", delays)
	}

	if len(schedule) != 1 {
		t.Errorf("Received larger than expected schedule set: %+v", delays)
	}

	if delays[0] != 0.5 {
		t.Errorf("Received unexpected delay bucket: %+v", delays[0])
	}

	if schedule[delays[0]][0].Test != "true" || schedule[delays[0]][0].Name != "g1/t1" {
		t.Errorf("Received unexpected schedule for bucket %+v: %+v", delays[0], *schedule[delays[0]][0])
	}

}

func TestMainScheduleDelayOrder(t *testing.T) {
	var c Configuration
	c.SetConfiguration(`
g1 { 
	t50	{ start_delay = 50ms; test = "true" }
	t500	{ start_delay = 500ms; test = "true" }
	t0	{ test = "true" } 
}`)

	/* we should receive these lowest delay first */
	exDelays := []float64{0, 0.05, 0.5}
	exRuleNames := []string{"g1/t0", "g1/t50", "g1/t500"}

	delays, schedule := scheduleRules(c.RulesOrder, &c.Rules)

	if len(delays) != 3 {
		t.Errorf("Received unexpected delay set: %+v", delays)
	}

	if len(schedule) != 3 {
		t.Errorf("Received unexpected schedule set: %+v", schedule)
	}

	i := 0
	for _, d := range delays {
		if d != exDelays[i] {
			t.Errorf("Received unexpected delay bucket: %+v", delays[0])
		}

		if len(schedule[d]) != 1 {
			t.Errorf("Received unexpected schedule set: %+v", schedule[d])
		}

		if schedule[d][0].Name != exRuleNames[i] {
			t.Errorf("Received unexpected rule for bucket %+v: %+v", d, schedule[d][0])
		}

		i++
	}
}

func TestMainScheduleDelayScanOrdering(t *testing.T) {
	var c Configuration
	c.SetConfiguration(`
g1 { 
	t0	{ test = "true" }
	t50	{ start_delay = 50ms; test = "true" }
	t500	{ start_delay = 500ms; test = "true" }
}
g2 { 
	t50	{ start_delay = 50ms; test = "true" }
	t500	{ start_delay = 500ms; test = "true" }
	t0	{ test = "true" }
}
g3 { 
	t500	{ start_delay = 500ms; test = "true" }
	t0	{ test = "true" }
	t50	{ start_delay = 50ms; test = "true" }
}
`)

	exDelays := []float64{0, 0.05, 0.5}
	exRuleNames := [][]string{
		{"g1/t0", "g2/t0", "g3/t0"},
		{"g1/t50", "g2/t50", "g3/t50"},
		{"g1/t500", "g2/t500", "g3/t500"},
	}

	delays, schedule := scheduleRules(c.RulesOrder, &c.Rules)

	if len(delays) != 3 {
		t.Errorf("Received unexpected delay set: %+v", delays)
	}

	if len(schedule) != 3 {
		t.Errorf("Received unexpected schedule set: %+v", schedule)
	}

	i := 0
	for _, d := range delays {
		if d != exDelays[i] {
			t.Errorf("Received unexpected delay bucket: %+v", delays[0])
		}

		if len(schedule[d]) != 3 {
			t.Errorf("Received unexpected schedule set: %+v", schedule[d])
		}

		j := 0
		for _, rule := range schedule[d] {
			if rule.Name != exRuleNames[i][j] {
				t.Errorf("Received unexpected rule for bucket %+v: %+v", d, rule)
			}

			j++
		}

		i++
	}

}
