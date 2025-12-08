package health

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

type ScriptChecker struct {
	name    string
	script  string
	args    []string
	timeout time.Duration
}

func NewScriptChecker(name string, config map[string]interface{}) (*ScriptChecker, error) {
	script, ok := config["script"].(string)
	if !ok {
		return nil, fmt.Errorf("script checker: missing or invalid 'script' field")
	}

	var args []string
	if argsInterface, ok := config["args"].([]interface{}); ok {
		args = make([]string, len(argsInterface))
		for i, arg := range argsInterface {
			if argStr, ok := arg.(string); ok {
				args[i] = argStr
			}
		}
	}

	timeout := 10 * time.Second
	if t, ok := config["timeout"].(string); ok {
		if d, err := time.ParseDuration(t); err == nil {
			timeout = d
		}
	}

	return &ScriptChecker{
		name:    name,
		script:  script,
		args:    args,
		timeout: timeout,
	}, nil
}

func (c *ScriptChecker) Name() string {
	return c.name
}

func (c *ScriptChecker) Type() string {
	return "script"
}

func (c *ScriptChecker) Check(ctx context.Context) CheckResult {
	cmdCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, c.script, c.args...)

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() != 0 {
				return CheckResultFailure
			}
		}
		return CheckResultFailure
	}

	return CheckResultSuccess
}
