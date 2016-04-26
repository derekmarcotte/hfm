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
	"time"
)

/* external includes */
import "github.com/mitchellh/go-libucl"

/* definitions */

// whether the following Rule fields were found when parsing the configuration
type RuleFound struct {
	Interval              bool
	IntervalFail          bool
	StartDelay            bool
	TimeoutInt            bool
	TimeoutKill           bool
	Runs                  bool
	ChangeFailDebounce    bool
	ChangeSuccessDebounce bool
}

/* How far we are nested into the config */
type ConfigLevelType int

const (
	ConfigLevelRoot ConfigLevelType = iota
	ConfigLevelGroup
	ConfigLevelRule
)

/* hfm configuration */
type Configuration struct {
	/* path to configuration file, may be empty */
	path string

	/* set of the rules parsed from the config, string maps to rule name */
	Rules map[string]*Rule

	/* the order that the rules were parsed in, so we can schedule in parse
	 * order
	 */
	RulesOrder []string

	/* our group rules before resolving inheritance, string maps to group
	 * name
	 */
	ruleDefaults map[string]*Rule

	/* whether a value was explicitly set for a parsed rule, string maps to
	 * rule name
	 */
	ruleFinds map[string]*RuleFound
}

/* meat */

/* set the configuration by a static string */
func (c *Configuration) SetConfiguration(config string) error {
	return c.startConfiguration(config, "string")
}

/* load and set the configuration from a file */
func (c *Configuration) LoadConfiguration(path string) error {
	return c.startConfiguration(path, "file")
}

func (c *Configuration) startConfiguration(config string, configType string) error {
	var e error

	/* get the ucl object */
	p := libucl.NewParser(0)
	defer p.Close()

	switch configType {
	case "string":
		e = p.AddString(config)
	default:
		e = p.AddFile(config)
	}

	if e != nil {
		return e
	}

	uclConfig := p.Object()
	defer uclConfig.Close()

	/* use it to populate myself as a valid hfm object */
	return c.walkConfiguration(uclConfig, "", ConfigLevelRoot)
}

/* figure out what this element's name should be */
func (config *Configuration) buildName(uclConfig *libucl.Object, parentRule string, depth ConfigLevelType) (string, error) {
	if depth == ConfigLevelRoot {
		return "default", nil
	}

	var err error
	var name string

	if parentRule == "default" {
		name = uclConfig.Key()
	} else {
		name = parentRule + "/" + uclConfig.Key()
	}

	err = nil
	if name == "" {
		err = errors.New("Rule is missing a name.")
	} else if _, ok := config.Rules[name]; ok {
		err = fmt.Errorf("%s: name has been used already", name)
	} else if _, ok := config.ruleDefaults[name]; ok {
		err = fmt.Errorf("%s: name has been used by a group already", name)
	}

	return name, err
}

/* recursively walk the ucl configuration, populating an hfm Configuration
 * instance
 */
func (config *Configuration) walkConfiguration(uclConfig *libucl.Object, parentRule string, depth ConfigLevelType) error {

	if depth == ConfigLevelRoot {
		config.ruleFinds = make(map[string]*RuleFound)
		config.ruleDefaults = make(map[string]*Rule)
		config.Rules = make(map[string]*Rule)
	}

	name, err := config.buildName(uclConfig, parentRule, depth)
	if err != nil {
		return err
	}

	var nextDepth ConfigLevelType

	/* all actual rules have a test, defaults do not */
	isRule := (uclConfig.Get("test") != nil)

	switch depth {
	case ConfigLevelRoot:
		nextDepth = ConfigLevelGroup
	case ConfigLevelGroup:
		nextDepth = ConfigLevelRule
	case ConfigLevelRule:
		if !isRule {
			return fmt.Errorf("%s: a 'test' value must exist for rules", name)
		}
	}

	var ruleFound RuleFound
	rule := Rule{Name: name, GroupName: parentRule}

	if !isRule {
		config.ruleDefaults[name] = &rule
	} else {
		config.Rules[name] = &rule
		config.RulesOrder = append(config.RulesOrder, name)
		config.ruleFinds[name] = &ruleFound
	}

	i := uclConfig.Iterate(true)
	defer i.Close()

	for c := i.Next(); c != nil; c = i.Next() {
		defer c.Close()
		field := strings.ToLower(c.Key())

		if c.Type() == libucl.ObjectTypeObject {
			/* if we are a rule, we stop parsing children */
			if depth != ConfigLevelRule || !isRule {
				config.walkConfiguration(c, name, nextDepth)
			} else {
				return fmt.Errorf("%s: '%s' rules cannot contain child rules", name, field)
			}

			continue
		}

		switch field {
		case "status":
			if c.Type() != libucl.ObjectTypeString {
				return fmt.Errorf("%s: '%s' must be a string type, got type %v", name, field, c.Type())
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
				return fmt.Errorf("%s: '%s' does not contain a valid string", name, field)
			}
		case "start_delay", "interval", "interval_fail", "timeout_int", "timeout_kill":
			tmp := 0.0
			/* interval/duration fields */
			switch c.Type() {
			case libucl.ObjectTypeInt, libucl.ObjectTypeFloat, libucl.ObjectTypeTime:
				tmp = c.ToFloat()
			default:
				return fmt.Errorf("%s: '%s' must be a valid numeric type, got type %v", name, field, c.Type())
			}

			switch field {
			case "start_delay":
				rule.StartDelay = intervalToDuration(tmp)
				ruleFound.StartDelay = true
			case "interval":
				rule.Interval = intervalToDuration(tmp)
				ruleFound.Interval = true
			case "interval_fail":
				rule.IntervalFail = intervalToDuration(tmp)
				ruleFound.IntervalFail = true
			case "timeout_int":
				rule.TimeoutInt = intervalToDuration(tmp)
				ruleFound.TimeoutInt = true
			case "timeout_kill":
				rule.TimeoutKill = intervalToDuration(tmp)
				ruleFound.TimeoutKill = true
			}
		case "test", "change_fail", "change_success":
			/* command fields */
			if c.Type() != libucl.ObjectTypeString {
				return fmt.Errorf("%s: '%s' must be a string type, got type %v", name, field, c.Type())
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
						return fmt.Errorf("%s: '%s' must contain only string elements, got type %v", name, field, arg.Type())
					}

					tmp = append(tmp, arg.ToString())
				}

			} else {
				return fmt.Errorf("%s: '%s' must be a string or an array of strings, got type %v", name, field, c.Type())
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
				return fmt.Errorf("%s: '%s' must be an integer type, got type %v", name, field, c.Type())
			}

			tmp := c.ToInt()
			if tmp < 0 || tmp > 65535 {
				return fmt.Errorf("%s: '%s' must be in 0..65535", name, field)
			}

			rule.Runs = uint16(tmp)
			ruleFound.Runs = true
		case "change_fail_debounce", "change_success_debounce":
			if c.Type() != libucl.ObjectTypeInt {
				return fmt.Errorf("%s: '%s' must be an integer type, got type %v", name, field, c.Type())
			}

			tmp := c.ToInt()

			if tmp < 1 || tmp > 65535 {
				return fmt.Errorf("%s: '%s' must be in 1..65535", name, field)
			}

			switch field {
			case "change_fail_debounce":
				rule.ChangeFailDebounce = uint16(tmp)
				ruleFound.ChangeFailDebounce = true
			case "change_success_debounce":
				rule.ChangeSuccessDebounce = uint16(tmp)
				ruleFound.ChangeSuccessDebounce = true
			}

		default:
			return fmt.Errorf("%s: '%s' unrecognized property", name, c.Key())
		}
	}

	if depth == ConfigLevelRoot {
		config.resolveDefaults()
	}

	return nil
}

/* take our set of raw parsed rules, and apply the group and root inherited,
 * and initial values
 */
func (c *Configuration) resolveDefaults() {
	var f RuleFound

	for _, rule := range c.Rules {
		if tmp, ok := c.ruleFinds[rule.Name]; ok {
			f = *tmp
		} else {
			f = RuleFound{}
		}

		/* inherit group first */
		if g, ok := c.ruleDefaults[rule.GroupName]; ok {
			c.inheritValues(rule, *g, &f)

			/* inherit root next */
			if d, ok := c.ruleDefaults[g.GroupName]; ok {
				c.inheritValues(rule, *d, &f)
			}
		}

		if rule.Status == RuleStatusUnset {
			rule.Status = RuleStatusEnabled
		}

		/* properties that likely should be non-zero after defaults
		 * applied
		 */
		if !f.IntervalFail && rule.IntervalFail == 0 {
			rule.IntervalFail = rule.Interval
		}

		/* these must be greater than zero */
		if !f.ChangeFailDebounce && rule.ChangeFailDebounce == 0 {
			rule.ChangeFailDebounce = 1
		}

		if !f.ChangeSuccessDebounce && rule.ChangeSuccessDebounce == 0 {
			rule.ChangeSuccessDebounce = 1
		}
	}

	/* we don't need this book keeping around after this step */
	c.ruleDefaults = nil
	c.ruleFinds = nil
}

/* apply inherited values to fields that haven't been explicitly set */
func (c *Configuration) inheritValues(dst *Rule, src Rule, f *RuleFound) {
	if dst.Status == RuleStatusUnset {
		dst.Status = src.Status
	}

	/* we need to check for 0s here, as they may have been set by the
	 * group, and now we are in the root context
	 */

	//fmt.Printf("%s: %+v\n", dst.Name, *f)

	if !f.Runs && dst.Runs == 0 {
		dst.Runs = src.Runs
	}

	if !f.Interval && dst.Interval == 0 {
		dst.Interval = src.Interval
	}

	if !f.IntervalFail && dst.IntervalFail == 0 {
		dst.IntervalFail = src.IntervalFail
	}

	if !f.StartDelay && dst.StartDelay == 0 {
		dst.StartDelay = src.StartDelay
	}

	if !f.TimeoutInt && dst.TimeoutInt == 0 {
		dst.TimeoutInt = src.TimeoutInt
	}

	if !f.TimeoutKill && dst.TimeoutKill == 0 {
		dst.TimeoutKill = src.TimeoutKill
	}

	if !f.ChangeFailDebounce && dst.ChangeFailDebounce == 0 {
		dst.ChangeFailDebounce = src.ChangeFailDebounce
	}

	if !f.ChangeSuccessDebounce && dst.ChangeSuccessDebounce == 0 {
		dst.ChangeSuccessDebounce = src.ChangeSuccessDebounce
	}
}

// coverts a rule interval to a time.Duration
func intervalToDuration(i float64) time.Duration {
	return time.Duration(i * float64(time.Second))
}
