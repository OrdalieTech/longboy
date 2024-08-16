package models

import (
	"fmt"
	"os/exec"
)

type CodeAction struct {
	BaseAction
	Language   string `json:"language"`
	SourceCode string `json:"source_code"`
}

func (c *CodeAction) GetID() string {
	return c.ID
}

func (c *CodeAction) SetID(id string) {
	c.ID = id
	c.ResultID = id
}

func (c *CodeAction) GetDescription() string {
	return c.Description
}

func (c *CodeAction) GetType() string {
	return "code"
}

/*
POC —
Need refactoring to avoid using exec.Commands inside the Longboy container
-> Maybe using a pakcage like Goja (github.com/dop251/goja)
*/
func (c *CodeAction) Exec(ctx *Context) error {
	var cmd *exec.Cmd
	switch c.Language {
	case "python":
		cmd = exec.Command("python", "-c", c.SourceCode)
	case "bash":
		cmd = exec.Command("bash", "-c", c.SourceCode)
	default:
		return fmt.Errorf("unsupported language: %s", c.Language)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("execution failed: %s, output: %s", err, output)
	}

	if c.ResultID != "" {
		ctx.Results[c.ResultID] = string(output)
	}

	return nil
}

func (c *CodeAction) GetResultID() string {
	return c.ResultID
}

func (c *CodeAction) GetFollowingActionID() string {
	return c.FollowingActionID
}
