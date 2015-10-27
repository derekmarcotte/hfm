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
	RuleStatusEnabled RuleStatusType = iota
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
	groupName string

	/* name of the rule in the grouping */
	name string

	/* what is the status of this rule */
	status RuleStatusType

	/* what is the period between scheduled runs */
	interval float64

	/* what is the period between scheduled runs on previously failed rules */
	failInterval float64

	/* shell command to run to initiate test */
	/*  hoping to extend to support go-native tests */
	test string

	/* shell command to run when the state changes to failed */
	changeFail string

	/* shell command to run when the state changes to success */
	changeSuccess string

	/* the result of the last rule check */
	lastState RuleStateType
}
