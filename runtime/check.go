package runtime

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/manelmontilla/vulcan-sdk/check/report"
)

const (
	StateCreated      = "CREATED"
	StateInit         = "INIT"
	StateRunning      = "RUNNING"
	StateTimeout      = "TIMEOUT"
	StateAborted      = "ABORTED"
	StateKilled       = "KILLED"
	StateFailed       = "FAILED"
	StateFinished     = "FINISHED"
	StateMalformed    = "MALFORMED"
	StateInconclusive = "INCONCLUSIVE"
)

// State represents the possible states that a check can take.
type State string

// NewState creates a State from its name, returns an error if the name is
// invalid.
func NewState(s string) (State, error) {
	for _, states := range CheckStates {
		_, found := slices.BinarySearch(states, State(s))
		if found {
			return State(s), nil
		}
	}
	return "", fmt.Errorf("invalid state %s", s)
}

// UnmarshalJSON unmarshals a [State] from its JSON representation.
func (s *State) UnmarshalJSON(data []byte) error {
	v, err := NewState(string(data))
	if err != nil {
		return err
	}
	*s = v
	return nil
}

// Check defines a check that can be run by the [Runtime].
type Check struct {
	Image    string
	Target   string
	Timeout  *int
	Options  string
	Metadata map[string]string
}

var (
	// CheckStates defines the valid state changes of a check over time.
	CheckStates = States{
		[]State{StateCreated},
		[]State{StateInit},
		[]State{StateRunning},
		[]State{
			StateMalformed,
			StateAborted,
			StateKilled,
			StateFailed,
			StateFinished,
			StateTimeout,
			StateInconclusive,
		},
	}
)

func init() {
	CheckStates.init()
}

// States holds all the States in a finite state machine in a way that is easy
// to determine if a check is less or equal than any other given state. This
// implementation supposes that there are only few States so the cost of walking
// through all the States is close to constant.
type States [][]State

func (c States) init() {
	for _, s := range c {
		slices.Sort(s)
	}
}

// LessOrEqual returns the states from state machine that are preceding s. If s
// is not an existent state in the state machine, all states are returned.
func (c States) LessOrEqual(s State) []State {
	res := []State{}
	for i := 0; i < len(c); i++ {
		res = append(res, c[i]...)
		_, found := slices.BinarySearch(c[i], s)
		if found {
			break
		}
	}
	return res
}

// HigherThan returns the states that a check can take after a given state.
// If is not an existent state in the state machine, all states are returned.
func (c States) HigherThan(s State) []State {
	res := []State{}
	for i := len(c) - 1; i >= 0; i-- {
		_, found := slices.BinarySearch(c[i], s)
		if found {
			break
		}
		res = append(res, c[i]...)
	}
	return res
}

// IsHigher returns true if a given state comes after the provided base state in
// a check execution flow.
func (c States) IsHigher(s, base State) bool {
	for _, v := range c.HigherThan(base) {
		if s == v {
			return true
		}
	}
	return false
}

// IsLessOrEqual returns true if a given state comes before the provided base
// state in a check execution flow.
func (c States) IsLessOrEqual(s, base State) bool {
	for _, v := range c.LessOrEqual(base) {
		if s == v {
			return true
		}
	}
	return false
}

// Terminal returns the terminal states in a check execution flow.
func (c States) Terminal() []State {
	return c[len(c)-1]
}

// NonTerminal returns all the states that are non terminal.
func (c States) NonTerminal() []State {
	return []State{StateCreated, StateInit, StateRunning}
}

// IsTerminal returns true if the given state is terminal.
func (c States) IsTerminal(s State) bool {
	t := c.Terminal()
	_, found := slices.BinarySearch(t, s)
	return found
}

// runningCheck contains the information about a check being run by a [Runtime].
type runningCheck struct {
	ID         string
	Check      Check
	Cancel     context.CancelFunc
	Started    time.Time
	FinalState *State
	Report     *report.Report
	progress   chan RunState
}

// running stores the information related to the checks run by a [Runtime].
type running struct {
	checks map[string]Check
}
