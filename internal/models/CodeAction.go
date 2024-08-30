package models

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/dop251/goja"
)

type CodeActionData struct {
	Language   string `json:"language"`
	SourceCode string `json:"source_code"`
}

func GetCodeActionData(a *Action) (*CodeActionData, error) {
	data := &CodeActionData{}
	if a.Metadata["language"] != nil {
		data.Language = a.Metadata["language"].(string)
	}
	if a.Metadata["source_code"] != nil {
		data.SourceCode = a.Metadata["source_code"].(string)
	}
	return data, nil
}

func CodeActionDataToMetadata(data *CodeActionData) map[string]interface{} {
	return map[string]interface{}{
		"language":    data.Language,
		"source_code": data.SourceCode,
	}
}

/*
POC —
Need refactoring to avoid using exec.Commands inside the Longboy container
-> Maybe using a pakcage like Goja (github.com/dop251/goja)
*/
func (a *Action) ExecCode(ctx *Context) error {
	c, err := GetCodeActionData(a)
	if err != nil {
		return err
	}
	sc := c.SourceCode
	sc, err = a.ProcessBody(ctx, sc)
	if err != nil {
		return err
	}
	fmt.Printf("Source Code: %s\n", sc)
	var output strings.Builder
	switch c.Language {
	case "python":
		cmd := exec.Command("python", "-c", sc)
		var byteOutput []byte
		byteOutput, err = cmd.CombinedOutput()
		output.Write(byteOutput)
	case "bash":
		cmd := exec.Command("bash", "-c", sc)
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

		_, err = vm.RunString(sc)
		if err != nil {
			return fmt.Errorf("JavaScript execution failed: %v", err)
		}
	default:
		return fmt.Errorf("unsupported language: %s", c.Language)
	}

	if err != nil {
		return fmt.Errorf("execution failed: %s, output: %s", err, output.String())
	}

	if a.ResultID != "" {
		ctx.Results[a.ResultID] = output.String()
	}

	return nil
}
