package models

import (
	"fmt"
	"strconv"
)

type BranchActionData struct {
	Rank      string   `json:"rank"`
	ActionsID []string `json:"actions"`
}

func GetBranchActionData(a *Action) (*BranchActionData, error) {
	data := &BranchActionData{}
	if a.Metadata["rank"] != nil {
		data.Rank = a.Metadata["rank"].(string)
	}
	if a.Metadata["actions_id"] != nil {
		// Convert map[string]interface{} to []string
		actionsIDInterface, ok := a.Metadata["actions_id"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("actions_id is not a []interface{}")
		}
		data.ActionsID = make([]string, len(actionsIDInterface))
		for i, v := range actionsIDInterface {
			strValue, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("action ID at index %d is not a string", i)
			}
			data.ActionsID[i] = strValue
		}
	}
	return data, nil
}

func BranchActionDataToMetadata(data *BranchActionData) map[string]interface{} {
	return map[string]interface{}{
		"rank":       data.Rank,
		"actions_id": data.ActionsID,
	}
}

func (a *Action) ExecBranch(ctx *ActionChainContext) error {
	b, err := GetBranchActionData(a)
	if err != nil {
		return err
	}
	// Replace placeholders in the body with actual secret values and context values
	body := b.Rank
	body, err = a.ProcessBody(ctx, body)
	if err != nil {
		return err
	}

	rank, err := strconv.Atoi(body)
	if err != nil {
		return fmt.Errorf("error converting rank to int: %v", err)
	}
	a.FollowingActionID = b.ActionsID[rank]

	return nil
}
