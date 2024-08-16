package models

import "fmt"

type BranchAction struct {
	BaseAction
	Condition     string `json:"condition"`
	TrueActionID  string `json:"true_action_id"`
	FalseActionID string `json:"false_action_id"`
}

func (b *BranchAction) GetID() string {
	return b.ID
}

func (b *BranchAction) SetID(id string) {
	b.ID = id
}

func (b *BranchAction) GetDescription() string {
	return b.Description
}

func (b *BranchAction) GetType() string {
	return "branch"
}

func (b *BranchAction) Exec(ctx *Context) error {
	// Evaluate the condition
	result, err := evaluateCondition(b.Condition, ctx)
	if err != nil {
		return fmt.Errorf("error evaluating condition: %v", err)
	}

	// Set the FollowingActionID based on the condition result
	if result {
		b.FollowingActionID = b.TrueActionID
	} else {
		b.FollowingActionID = b.FalseActionID
	}

	return nil
}

func (b *BranchAction) GetResultID() string {
	return ""
}

func (b *BranchAction) GetFollowingActionID() string {
	return b.FollowingActionID
}
