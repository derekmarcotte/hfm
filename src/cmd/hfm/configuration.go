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
	"errors"
	"fmt"
	"strings"
)

/* external includes */
import "github.com/mitchellh/go-libucl"

/* definitions */

/* ConfigLevelType is how far we are nested into the config */
type ConfigLevelType int

const (
	ConfigLevelRoot ConfigLevelType = iota
	ConfigLevelGroup
	ConfigLevelRule
)

type Configuration struct {
	path         string
	shell        string
	Rules        map[string]*Rule
	RulesOrder   []string
	ruleDefaults map[string]*Rule
}

/* meat */
func (c *Configuration) SetConfiguration(config string) error {
	p := libucl.NewParser(0)
	defer p.Close()

	if e := p.AddString(config); e != nil {
		return e
	}

	uclConfig := p.Object()
	defer uclConfig.Close()

	//fmt.Println(config.Emit(libucl.EmitConfig))
	if e := c.walkConfiguration(uclConfig, "", ConfigLevelRoot); e != nil {
		return e
	}

	c.resolveDefaults()

	return nil
}

func (c *Configuration) LoadConfiguration(path string) error {
	p := libucl.NewParser(0)
	defer p.Close()

	if e := p.AddFile(path); e != nil {
		return e
	}

	uclConfig := p.Object()
	defer uclConfig.Close()

	//fmt.Println(config.Emit(libucl.EmitConfig))
	if e := c.walkConfiguration(uclConfig, "", ConfigLevelRoot); e != nil {
		return e
	}

	c.resolveDefaults()

	return nil
}

/* config format:
 *  default
 *  group
 *    rule
 *  group
 *    rule
 *    rule
 *  rule
 *  default
 */
func (config *Configuration) walkConfiguration(uclConfig *libucl.Object, parentRule string, depth ConfigLevelType) error {
	var name string
	if depth == ConfigLevelRoot {
		name = "default"
		config.ruleDefaults = make(map[string]*Rule)
		config.Rules = make(map[string]*Rule)
	} else {
		if parentRule == "default" {
			name = uclConfig.Key()
		} else {
			name = parentRule + "/" + uclConfig.Key()
		}
	}

	if name == "" {
		return errors.New("Rule is missing a name.")
	} else if _, ok := config.Rules[name]; ok {
		return errors.New(fmt.Sprintf("%s: name has been used already", name))
	} else if _, ok := config.ruleDefaults[name]; ok {
		return errors.New(fmt.Sprintf("%s: name has been used by a group already", name))
	}

	var nextDepth ConfigLevelType
	tabs := 0
	var _ = tabs

	/* all actual rules have a test, defaults do not */
	isRule := (uclConfig.Get("test") != nil)

	switch depth {
	case ConfigLevelRoot:
		nextDepth = ConfigLevelGroup
	case ConfigLevelGroup:
		nextDepth = ConfigLevelRule
		tabs = 1
	case ConfigLevelRule:
		if !isRule {
			return errors.New(fmt.Sprintf("%s: a 'test' value must exist for rules", name))
		}
		tabs = 2
	}

	rule := Rule{Name: name, GroupName: parentRule}
	if rule.Name == "default" {
		rule.Interval = 1
		rule.TimeoutInt = 1
		rule.Status = RuleStatusEnabled
		rule.Shell = "/bin/sh"
	}

	if !isRule {
		config.ruleDefaults[name] = &rule
	} else {
		config.Rules[name] = &rule
		config.RulesOrder = append(config.RulesOrder, name)
	}

	i := uclConfig.Iterate(true)
	defer i.Close()

	for c := i.Next(); c != nil; c = i.Next() {
		defer c.Close()
		field := strings.ToLower(c.Key())

		if c.Type() == libucl.ObjectTypeObject {
			/* if we are a rule, we stop parsing children */
			if depth != ConfigLevelRule || !isRule {
				//fmt.Printf("%s%v: \n", strings.Repeat("\t", tabs), c.Key())
				config.walkConfiguration(c, name, nextDepth)
			} else {
				return errors.New(fmt.Sprintf("%s: '%s' rules cannot contain child rules", name, field))
			}

			continue
		}

		//fmt.Printf("%s%+v\t%v\n", strings.Repeat("\t", tabs), c.Key(), c.Type())

		switch field {
		case "shell":
			if c.Type() != libucl.ObjectTypeString {
				return errors.New(fmt.Sprintf("%s: '%s' must be a string type", name, field))
			}

			rule.Shell = c.ToString()
		case "status":
			if c.Type() != libucl.ObjectTypeString {
				return errors.New(fmt.Sprintf("%s: '%s' must be a string type", name, field))
			}

			switch strings.ToLower(c.ToString()) {
			case "enabled":
				rule.Status = RuleStatusEnabled
			case "disabled":
				rule.Status = RuleStatusDisabled
			case "run-once":
				rule.Status = RuleStatusRunOnce
			case "run-once-fail":
				rule.Status = RuleStatusRunOnceFail
			case "run-once-sucess":
				rule.Status = RuleStatusRunOnceSuccess
			case "always-fail":
				rule.Status = RuleStatusAlwaysFail
			case "always-success":
				rule.Status = RuleStatusAlwaysSuccess
			default:
				return errors.New(fmt.Sprintf("%s: '%s' does not contain a valid string", name, field))
			}
		case "interval":
			switch c.Type() {
			case libucl.ObjectTypeInt, libucl.ObjectTypeFloat, libucl.ObjectTypeTime:
				rule.Interval = c.ToFloat()
			default:
				return errors.New(fmt.Sprintf("%s: '%s' must be a valid numeric type", name, field))
			}
		case "fail_interval":
			switch c.Type() {
			case libucl.ObjectTypeInt, libucl.ObjectTypeFloat, libucl.ObjectTypeTime:
				rule.IntervalFail = c.ToFloat()
			default:
				return errors.New(fmt.Sprintf("%s: '%s' must be a valid numeric type", name, field))
			}
		case "start_delay":
			switch c.Type() {
			case libucl.ObjectTypeInt, libucl.ObjectTypeFloat, libucl.ObjectTypeTime:
				rule.StartDelay = c.ToFloat()
			default:
				return errors.New(fmt.Sprintf("%s: '%s' must be a valid numeric type", name, field))
			}
		case "timeout_int":
			switch c.Type() {
			case libucl.ObjectTypeInt, libucl.ObjectTypeFloat, libucl.ObjectTypeTime:
				rule.TimeoutInt = c.ToFloat()
			default:
				return errors.New(fmt.Sprintf("%s: '%s' must be a valid numeric type", name, field))
			}
		case "timeout_kill":
			switch c.Type() {
			case libucl.ObjectTypeInt, libucl.ObjectTypeFloat, libucl.ObjectTypeTime:
				rule.TimeoutKill = c.ToFloat()
			default:
				return errors.New(fmt.Sprintf("%s: '%s' must be a valid numeric type", name, field))
			}
		case "test":
			if c.Type() != libucl.ObjectTypeString {
				return errors.New(fmt.Sprintf("%s: '%s' must be a string type", name, field))
			}
			rule.Test = c.ToString()
		case "change_fail":
			if c.Type() != libucl.ObjectTypeString {
				return errors.New(fmt.Sprintf("%s: '%s' must be a string type", name, field))
			}
			rule.ChangeFail = c.ToString()
		case "change_success":
			if c.Type() != libucl.ObjectTypeString {
				return errors.New(fmt.Sprintf("%s: '%s' must be a string type", name, field))
			}
			rule.ChangeSuccess = c.ToString()
		default:
			//fmt.Printf("%s%+v\n", strings.Repeat("\t", tabs), c)
			return errors.New(fmt.Sprintf("%s: '%s' unrecognized property", name, c.Key()))
		}
	}

	//fmt.Printf("%s%+v\n", strings.Repeat("\t", tabs), rule)
	return nil
}

func (c *Configuration) resolveDefaults() {
	for _, rule := range c.Rules {
		if g, ok := c.ruleDefaults[rule.GroupName]; ok {
			c.mapDefaults(rule, *g)

			if d, ok := c.ruleDefaults[g.GroupName]; ok {
				// map the root/default rules before applying the
				// group rules
				c.mapDefaults(rule, *d)
			}
		}

		if rule.IntervalFail == 0 {
			rule.IntervalFail = rule.Interval
		}

		if rule.TimeoutKill == 0 {
			// 3 is totally arbitrary
			rule.TimeoutKill = rule.TimeoutInt + 3
		}
	}
	c.ruleDefaults = nil
}

func (c *Configuration) mapDefaults(dst *Rule, src Rule) {
	if dst.Status == RuleStatusUnset {
		dst.Status = src.Status
	}

	if dst.Shell == "" {
		dst.Shell = src.Shell
	}

	if dst.Interval == 0 {
		dst.Interval = src.Interval
	}

	if dst.IntervalFail == 0 {
		dst.IntervalFail = src.IntervalFail
	}

	if dst.StartDelay == 0 {
		dst.StartDelay = src.StartDelay
	}

	if dst.TimeoutInt == 0 {
		dst.TimeoutInt = src.TimeoutInt
	}
}
