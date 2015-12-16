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

	/* run the rule once at startup, then disable the rule
	 * helpful for failing over hosts, or services
	 */
	RuleStatusRunOnce
	RuleStatusRunOnceFail
	RuleStatusRunOnceSuccess

	/* are these helpful, at all? */
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

	/* what is the period this task can run for, before killing it */
	TimeoutInt  float64
	TimeoutKill float64

	/* shell to execute commands in */
	Shell string

	/* shell command to run to initiate test */
	/*  hoping to extend to support go-native tests */
	Test string

	/* shell command to run when the state changes to failed */
	ChangeFail string

	/* shell command to run when the state changes to success */
	ChangeSuccess string

	/* the result of the last rule check */
	LastState RuleStateType
}
