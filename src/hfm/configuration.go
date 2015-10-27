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

	c.ruleDefaults = make(map[string]*Rule)
	c.rules = make(map[string]*Rule)

	return c.walkConfiguration(uclConfig, "", ConfigLevelRoot)

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
		rule.failInterval = rule.interval
	}

	if !isRule {
		config.ruleDefaults[name] = &rule
	}
	config.rules[name] = &rule

	i := uclConfig.Iterate(true)
	defer i.Close()

	for c := i.Next(); c != nil; c = i.Next() {
		defer c.Close()

		if c.Type() == libucl.ObjectTypeObject {
			/* if we are a rule, we stop parsing children */
			if depth != ConfigLevelRule || !isRule {
				//fmt.Printf("%s%v: \n", strings.Repeat("\t", tabs), c.Key())
				config.walkConfiguration(c, name, nextDepth)
			} else {
				return errors.New(fmt.Sprintf("%s: rules cannot contain child rules", name))
			}

			continue
		}

		//fmt.Printf("%s%+v\t%v\n", strings.Repeat("\t", tabs), c.Key(), c.Type())

		switch strings.ToLower(c.Key()) {
		case "status":
			if c.Type() != libucl.ObjectTypeString {
				return errors.New(fmt.Sprintf("%s: 'status' must be a string type", name))
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
				return errors.New(fmt.Sprintf("%s: 'status' does not contain a valid string", name))
			}
		case "interval":
			switch c.Type() {
			case libucl.ObjectTypeInt, libucl.ObjectTypeFloat, libucl.ObjectTypeTime:
				rule.interval = c.ToFloat()
			default:
				return errors.New(fmt.Sprintf("%s: 'interval' must be a valid numeric type", name))
			}
		case "fail_interval":
			switch c.Type() {
			case libucl.ObjectTypeInt, libucl.ObjectTypeFloat, libucl.ObjectTypeTime:
				rule.interval = c.ToFloat()
			default:
				return errors.New(fmt.Sprintf("%s: 'fail_interval' must be a valid numeric type", name))
			}
		case "test":
			if c.Type() != libucl.ObjectTypeString {
				return errors.New(fmt.Sprintf("%s: '%s' must be a string type", name, c.Key()))
			}
			rule.test = c.ToString()
		case "change_fail":
			if c.Type() != libucl.ObjectTypeString {
				return errors.New(fmt.Sprintf("%s: '%s' must be a string type", name, c.Key()))
			}
			rule.changeFail = c.ToString()
		case "change_success":
			if c.Type() != libucl.ObjectTypeString {
				return errors.New(fmt.Sprintf("%s: '%s' must be a string type", name, c.Key()))
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
