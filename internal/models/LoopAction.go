package models

import (
	"fmt"
)

type LoopActionData struct {
	Action    Action `json:"action"`
	Condition string `json:"condition"`
}

func GetLoopActionData(a *Action) (*LoopActionData, error) {
	data := &LoopActionData{}
	if a.Metadata["action"] != nil {
		data.Action = a.Metadata["action"].(Action)
	}
	if a.Metadata["condition"] != nil {
		data.Condition = a.Metadata["condition"].(string)
	}
	return data, nil
}

func LoopActionDataToMetadata(data *LoopActionData) map[string]interface{} {
	return map[string]interface{}{
		"action":    data.Action,
		"condition": data.Condition,
	}
}

func (a *Action) ExecLoop(ctx *Context) error {
	l, err := GetLoopActionData(a)
	if err != nil {
		return err
	}
	for {
		// Execute the action
		if err := l.Action.Exec(ctx); err != nil {
			return fmt.Errorf("error executing action: %v", err)
		}
		cond, err := a.ProcessBody(ctx, l.Condition)
		if err != nil {
			return err
		}
		// Evaluate the condition
		conditionMet, err := evaluateCondition(cond, ctx)
		if err != nil {
			return fmt.Errorf("error evaluating condition: %v", err)
		}
		if conditionMet {
			break
		}
	}

	return nil
}
