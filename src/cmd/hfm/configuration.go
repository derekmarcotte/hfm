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
		case "status":
			if c.Type() != libucl.ObjectTypeString {
				return errors.New(fmt.Sprintf("%s: '%s' must be a string type, got type %v", name, field, c.Type()))
			}

			switch strings.ToLower(c.ToString()) {
			case "enabled":
				rule.Status = RuleStatusEnabled
			case "disabled":
				rule.Status = RuleStatusDisabled
			case "always-fail":
				rule.Status = RuleStatusAlwaysFail
			case "always-success":
				rule.Status = RuleStatusAlwaysSuccess
			default:
				return errors.New(fmt.Sprintf("%s: '%s' does not contain a valid string", name, field))
			}
		case "start_delay", "interval", "fail_interval", "timeout_int", "timeout_kill":
			tmp := 0.0
			/* interval/duration fields */
			switch c.Type() {
			case libucl.ObjectTypeInt, libucl.ObjectTypeFloat, libucl.ObjectTypeTime:
				tmp = c.ToFloat()
			default:
				return errors.New(fmt.Sprintf("%s: '%s' must be a valid numeric type, got type %v", name, field, c.Type()))
			}

			switch field {
			case "start_delay":
				rule.StartDelay = tmp
			case "interval":
				rule.Interval = tmp
			case "fail_interval":
				rule.IntervalFail = tmp
			case "timeout_int":
				rule.TimeoutInt = tmp
			case "timeout_kill":
				rule.TimeoutKill = tmp
			}
		case "test", "change_fail", "change_success":
			/* command fields */
			if c.Type() != libucl.ObjectTypeString {
				return errors.New(fmt.Sprintf("%s: '%s' must be a string type, got type %v", name, field, c.Type()))
			}

			tmp := c.ToString()
			switch field {
			case "test":
				rule.Test = tmp
			case "change_fail":
				rule.ChangeFail = tmp
			case "change_success":
				rule.ChangeSuccess = tmp
			}
		case "test_arguments", "change_fail_arguments", "change_success_arguments":
			tmp := []string{}
			if c.Type() == libucl.ObjectTypeString {
				tmp = append(tmp, c.ToString())
			} else if c.Type() == libucl.ObjectTypeArray {

				j := c.Iterate(true)
				defer j.Close()

				for arg := j.Next(); arg != nil; arg = j.Next() {
					defer arg.Close()

					if arg.Type() != libucl.ObjectTypeString {
						return errors.New(fmt.Sprintf("%s: '%s' must contain only string elements, got type %v", name, field, arg.Type()))
					}

					tmp = append(tmp, arg.ToString())
				}

			} else {
				return errors.New(fmt.Sprintf("%s: '%s' must be a string or an array of strings, got type %v", name, field, c.Type()))
			}

			switch field {
			case "test_arguments":
				rule.TestArguments = tmp
			case "change_fail_arguments":
				rule.ChangeFailArguments = tmp
			case "change_success_arguments":
				rule.ChangeSuccessArguments = tmp
			}
		case "runs":
			if c.Type() != libucl.ObjectTypeInt {
				return errors.New(fmt.Sprintf("%s: '%s' must be an integer type, got type %v", name, field, c.Type()))
			}

			tmp := c.ToInt()
			if tmp < 0 || tmp > 65535 {
				return errors.New(fmt.Sprintf("%s: '%s' must be in 0..65535", name, field))
			}

			rule.Runs = uint16(tmp)
		case "change_fail_debounce", "change_success_debounce":
			if c.Type() != libucl.ObjectTypeInt {
				return errors.New(fmt.Sprintf("%s: '%s' must be an integer type, got type %v", name, field, c.Type()))
			}

			tmp := c.ToInt()

			if tmp < 1 || tmp > 65535 {
				return errors.New(fmt.Sprintf("%s: '%s' must be in 1..65535", name, field))
			}

			switch field {
			case "change_fail_debounce":
				rule.ChangeFailDebounce = uint16(tmp)
			case "change_success_debounce":
				rule.ChangeSuccessDebounce = uint16(tmp)
			}

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
			c.inheritValues(rule, *g)

			if d, ok := c.ruleDefaults[g.GroupName]; ok {
				// map the root/default rules before applying the
				// group rules
				c.inheritValues(rule, *d)
			}
		}

		if rule.IntervalFail == 0 {
			rule.IntervalFail = rule.Interval
		}

		if rule.TimeoutKill == 0 {
			// 3 is totally arbitrary
			rule.TimeoutKill = rule.TimeoutInt + 3
		}

		if rule.ChangeFailDebounce == 0 {
			rule.ChangeFailDebounce = 1
		}

		if rule.ChangeSuccessDebounce == 0 {
			rule.ChangeSuccessDebounce = 1
		}
	}
	c.ruleDefaults = nil
}

func (c *Configuration) inheritValues(dst *Rule, src Rule) {
	if dst.Status == RuleStatusUnset {
		dst.Status = src.Status
	}

	if dst.Runs == 0 {
		dst.Runs = src.Runs
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

	if dst.ChangeFailDebounce == 0 {
		dst.ChangeFailDebounce = src.ChangeFailDebounce
	}

	if dst.ChangeSuccessDebounce == 0 {
		dst.ChangeSuccessDebounce = src.ChangeSuccessDebounce
	}
}
