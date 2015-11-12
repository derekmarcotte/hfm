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
	rules        map[string]*Rule
	ruleDefaults map[string]*Rule

}

/* meat */
func (c *Configuration) LoadConfiguration(path string) error {
	p := libucl.NewParser(0)
	defer p.Close()

	e := p.AddFile(path)
	if e != nil {
		return e
	}

	uclConfig := p.Object()
	defer uclConfig.Close()

	//fmt.Println(config.Emit(libucl.EmitConfig))

	e = c.walkConfiguration(uclConfig, "", ConfigLevelRoot)
	if e != nil {
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
		config.rules = make(map[string]*Rule)
	} else {
		if parentRule == "default" {
			name = uclConfig.Key()
		} else {
			name = parentRule + "/" + uclConfig.Key()
		}
	}

	if name == "" {
		return errors.New("Rule is missing a name.")
	} else if _, ok := config.rules[name]; ok {
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

	rule := Rule{name: name, groupName: parentRule}
	if rule.name == "default" {
		rule.interval = 1 
		rule.timeoutInt = 1 
		rule.status = RuleStatusEnabled
		rule.shell = "/bin/sh"
	}

	if !isRule {
		config.ruleDefaults[name] = &rule
	} else {
		config.rules[name] = &rule
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

			rule.shell = c.ToString()
		case "status":
			if c.Type() != libucl.ObjectTypeString {
				return errors.New(fmt.Sprintf("%s: '%s' must be a string type", name, field))
			}

			switch strings.ToLower(c.ToString()) {
			case "enabled":
				rule.status = RuleStatusEnabled
			case "disabled":
				rule.status = RuleStatusDisabled
			case "run-once":
				rule.status = RuleStatusRunOnce
			case "run-once-fail":
				rule.status = RuleStatusRunOnceFail
			case "run-once-sucess":
				rule.status = RuleStatusRunOnceSuccess
			case "always-fail":
				rule.status = RuleStatusAlwaysFail
			case "always-success":
				rule.status = RuleStatusAlwaysSuccess
			default:
				return errors.New(fmt.Sprintf("%s: '%s' does not contain a valid string", name, field))
			}
		case "interval":
			switch c.Type() {
			case libucl.ObjectTypeInt, libucl.ObjectTypeFloat, libucl.ObjectTypeTime:
				rule.interval = c.ToFloat()
			default:
				return errors.New(fmt.Sprintf("%s: '%s' must be a valid numeric type", name, field))
			}
		case "fail_interval":
			switch c.Type() {
			case libucl.ObjectTypeInt, libucl.ObjectTypeFloat, libucl.ObjectTypeTime:
				rule.intervalFail = c.ToFloat()
			default:
				return errors.New(fmt.Sprintf("%s: '%s' must be a valid numeric type", name, field))
			}
		case "test":
			if c.Type() != libucl.ObjectTypeString {
				return errors.New(fmt.Sprintf("%s: '%s' must be a string type", name, field))
			}
			rule.test = c.ToString()
		case "change_fail":
			if c.Type() != libucl.ObjectTypeString {
				return errors.New(fmt.Sprintf("%s: '%s' must be a string type", name, field))
			}
			rule.changeFail = c.ToString()
		case "change_success":
			if c.Type() != libucl.ObjectTypeString {
				return errors.New(fmt.Sprintf("%s: '%s' must be a string type", name, field))
			}
			rule.changeSuccess = c.ToString()
		default:
			//fmt.Printf("%s%+v\n", strings.Repeat("\t", tabs), c)
			return errors.New(fmt.Sprintf("%s: '%s' unrecognized property", name, c.Key()))
		}
	}

	//fmt.Printf("%s%+v\n", strings.Repeat("\t", tabs), rule)
	return nil
}

func (c *Configuration) resolveDefaults() {
	for _, rule := range c.rules {
		if g, ok := c.ruleDefaults[rule.name]; ok {
			if r, ok := c.ruleDefaults[g.groupName]; ok {
				// map the root/default rules before applying the
				// group rules
				c.mapDefaults(rule, *r)
			}
			c.mapDefaults(rule, *g)
		}

		if rule.intervalFail == 0 {
			rule.intervalFail = rule.interval
		}

		if rule.timeoutKill == 0 {
			// 3 is totally arbitrary
			rule.timeoutKill = rule.timeoutInt + 3
		}
	}
	c.ruleDefaults = nil
}

func (c *Configuration) mapDefaults(dst *Rule, src Rule) {
	if dst.status == RuleStatusUnset {
		dst.status = src.status
	}

	if dst.interval == 0 {
		dst.interval = src.interval
	}

	if dst.intervalFail == 0 {
		dst.intervalFail = src.intervalFail
	}
}
