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

//go:generate stringer -type=RuleStatusType -type=RuleStateType rule.go

package main

/* definitions */

type RuleStateType int

const (
	RuleStateUnknown RuleStateType = iota
	RuleStateSuccess
	RuleStateFail
)

type RuleStatusType int

const (
	RuleStatusUnset RuleStatusType = iota
	RuleStatusEnabled
	/* a disabled service leaves the run-time configuration */
	RuleStatusDisabled

	RuleStatusAlwaysFail
	RuleStatusAlwaysSuccess
)

type Rule struct {
	/* name of the grouping for the rule */
	GroupName string

	/* name of the rule in the grouping */
	Name string

	/* what is the status of this rule */
	Status RuleStatusType

	/* what is the period between scheduled runs */
	Interval float64

	/* what is the period between scheduled runs on previously failed rules */
	IntervalFail float64

	/* how long do I delay until starting for the first time */
	StartDelay float64

	/* what is the period this task can run for, before killing it */
	TimeoutInt  float64
	TimeoutKill float64

	/* total number of test runs, 0 for infinite, 1 + AlwaysFail is helpful
	 * for administratively failing over hosts, for example
	 */
	Runs uint16

	/* command to run to initiate test */
	/*  hoping to extend to support go-native tests */
	Test          string
	TestArguments []string

	/* command to run when the state changes to failed */
	ChangeFail          string
	ChangeFailArguments []string
	ChangeFailDebounce  uint16

	/* command to run when the state changes to success */
	ChangeSuccess          string
	ChangeSuccessArguments []string
	ChangeSuccessDebounce  uint16

	/* current state change, debounce status */
	ChangeDebounce uint16

	/* the result of the last rule check */
	LastState RuleStateType
}
