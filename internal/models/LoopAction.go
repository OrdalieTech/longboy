package models

import (
	"fmt"
)

type LoopAction struct {
	ID                string `json:"id"`
	Type              string `json:"type"`
	Description       string `json:"description"`
	Action            Action `json:"action"`
	Condition         string `json:"condition"`
	FollowingActionID string `json:"following_action_id"`
}

func (l *LoopAction) GetID() string {
	return l.ID
}

func (l *LoopAction) SetID(id string) {
	l.ID = id
}

func (l *LoopAction) GetDescription() string {
	return l.Description
}

func (l *LoopAction) GetType() string {
	return "loop"
}

func (l *LoopAction) Exec(ctx *Context) error {
	for {
		// Execute the action
		if err := l.Action.Exec(ctx); err != nil {
			return fmt.Errorf("error executing action: %v", err)
		}

		// Evaluate the condition
		conditionMet, err := evaluateCondition(l.Condition, ctx)
		if err != nil {
			return fmt.Errorf("error evaluating condition: %v", err)
		}
		if conditionMet {
			break
		}
	}

	return nil
}

func (l *LoopAction) GetResultID() string {
	return ""
}

func (l *LoopAction) GetFollowingActionID() string {
	return l.FollowingActionID
}
