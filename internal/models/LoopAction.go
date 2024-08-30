package models

import (
	"fmt"
)

type LoopActionData struct {
	Action    Action `json:"action"`
	Condition string `json:"condition"`
}

func mapToPlaceholder(placeholderMap map[string]interface{}) (*Placeholder, error) {
	placeholder := &Placeholder{}
	if name, ok := placeholderMap["name"].(string); ok {
		placeholder.Name = name
	}
	if value, ok := placeholderMap["next"].(map[string]interface{}); ok {
		if value == nil {
			placeholder.Next = nil
		} else {
			p, err := mapToPlaceholder(value)
			if err != nil {
				return nil, err
			}
			placeholder.Next = p
		}
	}
	return placeholder, nil
}

func GetLoopActionData(a *Action) (*LoopActionData, error) {
	data := &LoopActionData{}
	if a.Metadata["action"] != nil {
		actionMap, ok := a.Metadata["action"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("action metadata is not in the expected format")
		}
		// Convert the map to an Action struct
		action := Action{}
		// Populate the Action struct fields from the map
		action.ID = actionMap["id"].(string)
		action.Type = actionMap["type"].(string)
		action.Description = actionMap["description"].(string)
		action.ResultID = actionMap["result_id"].(string)
		placeholders := actionMap["placeholders"].(map[string]interface{})
		action.Placeholders = make(map[string]*Placeholder)
		for key, value := range placeholders {
			if placeholderMap, ok := value.(map[string]interface{}); ok {
				placeholder, err := mapToPlaceholder(placeholderMap)
				if err != nil {
					return nil, fmt.Errorf("error mapping placeholder: %v", err)
				}
				action.Placeholders[key] = placeholder
			} else {
				return nil, fmt.Errorf("placeholder value is not in the expected format")
			}
		}
		// Add more fields as necessary
		action.Metadata = actionMap["metadata"].(map[string]interface{})
		data.Action = action
	}
	if a.Metadata["condition"] != nil {
		condition, ok := a.Metadata["condition"].(string)
		if !ok {
			return nil, fmt.Errorf("condition metadata is not a string")
		}
		data.Condition = condition
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
		fmt.Printf("Action: %+v\n", l.Action)
		// Execute the action
		if err := l.Action.Exec(ctx); err != nil {
			return fmt.Errorf("error executing action: %v", err)
		}
		cond, err := a.ProcessBody(ctx, l.Condition)
		if err != nil {
			return err
		}
		// Evaluate the condition
		conditionMet, err := evaluateCondition(cond)
		if err != nil {
			return fmt.Errorf("error evaluating condition: %v", err)
		}
		if conditionMet {
			break
		}
	}

	return nil
}
