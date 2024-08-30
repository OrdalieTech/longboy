package models

import "fmt"

type IfThenActionData struct {
	Condition     string `json:"condition"`
	TrueActionID  string `json:"true_action_id"`
	FalseActionID string `json:"false_action_id"`
}

func GetIfThenActionData(a *Action) (*IfThenActionData, error) {
	data := &IfThenActionData{}
	if a.Metadata["condition"] != nil {
		data.Condition = a.Metadata["condition"].(string)
	}
	if a.Metadata["true_action_id"] != nil {
		data.TrueActionID = a.Metadata["true_action_id"].(string)
	}
	if a.Metadata["false_action_id"] != nil {
		data.FalseActionID = a.Metadata["false_action_id"].(string)
	}
	return data, nil
}

func IfThenActionDataToMetadata(data *IfThenActionData) map[string]interface{} {
	return map[string]interface{}{
		"condition":       data.Condition,
		"true_action_id":  data.TrueActionID,
		"false_action_id": data.FalseActionID,
	}
}

func (a *Action) ExecIfThen(ctx *Context) error {
	i, err := GetIfThenActionData(a)
	if err != nil {
		return err
	}
	condition := i.Condition
	condition, err = a.ProcessBody(ctx, condition)
	if err != nil {
		return err
	}
	// Evaluate the condition
	result, err := evaluateCondition(condition)
	if err != nil {
		return fmt.Errorf("error evaluating condition: %v", err)
	}

	// Set the FollowingActionID based on the condition result
	if result {
		a.FollowingActionID = i.TrueActionID
	} else {
		a.FollowingActionID = i.FalseActionID
	}

	return nil
}
