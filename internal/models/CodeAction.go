package models

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/dop251/goja"
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
	var output strings.Builder
	var err error

	switch c.Language {
	case "python":
		cmd := exec.Command("python", "-c", c.SourceCode)
		var byteOutput []byte
		byteOutput, err = cmd.CombinedOutput()
		output.Write(byteOutput)
	case "bash":
		cmd := exec.Command("bash", "-c", c.SourceCode)
		var byteOutput []byte
		byteOutput, err = cmd.CombinedOutput()
		output.Write(byteOutput)
	case "javascript":
		vm := goja.New()

		// Provide a custom console.log implementation
		console := vm.NewObject()
		err = console.Set("log", func(call goja.FunctionCall) goja.Value {
			for _, arg := range call.Arguments {
				fmt.Fprintf(&output, "%v ", arg)
			}
			fmt.Fprintln(&output)
			return goja.Undefined()
		})
		if err != nil {
			return fmt.Errorf("failed to set console.log: %v", err)
		}
		err = vm.Set("console", console)
		if err != nil {
			return fmt.Errorf("failed to set console object: %v", err)
		}

		_, err = vm.RunString(c.SourceCode)
		if err != nil {
			return fmt.Errorf("JavaScript execution failed: %v", err)
		}
	default:
		return fmt.Errorf("unsupported language: %s", c.Language)
	}

	if err != nil {
		return fmt.Errorf("execution failed: %s, output: %s", err, output.String())
	}

	if c.ResultID != "" {
		ctx.Results[c.ResultID] = output.String()
	}

	return nil
}

func (c *CodeAction) GetResultID() string {
	return c.ResultID
}

func (c *CodeAction) GetFollowingActionID() string {
	return c.FollowingActionID
}
