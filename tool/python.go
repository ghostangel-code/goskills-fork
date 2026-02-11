package tool

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PythonConfig 配置
type PythonConfig struct {
	AllowedScripts []string
}

// RunPythonScriptInput 输入
type RunPythonScriptInput struct {
	ScriptPath string   `json:"scriptPath" jsonschema:"description=Path to the Python script to execute, or 'inline' for inline code"`
	Args       []string `json:"args,omitempty" jsonschema:"description=Arguments to pass to the script"`
	Code       string   `json:"code,omitempty" jsonschema:"description=Inline Python code to execute (when ScriptPath is 'inline')"`
}

// RunPythonScriptOutput 输出
type RunPythonScriptOutput struct {
	Stdout string `json:"stdout" jsonschema:"description=Standard output from the script"`
	Stderr string `json:"stderr" jsonschema:"description=Standard error from the script"`
}

// RunPythonScript 执行 Python 脚本或内联代码
func RunPythonScript(ctx context.Context, config RunnerConfig, input RunPythonScriptInput) (*RunPythonScriptOutput, error) {
	// 【改造】支持内联代码执行
	if input.ScriptPath == "inline" && input.Code != "" {
		return runInlinePython(ctx, config, input.Code, input.Args)
	}

	// 原有的脚本文件执行逻辑
	if !isAllowedScript(config.AllowedScripts, input.ScriptPath) {
		return nil, fmt.Errorf("script path '%s' is not in the allowed list. allowed scripts: %v",
			input.ScriptPath, config.AllowedScripts)
	}

	// Check if script exists
	if _, err := os.Stat(input.ScriptPath); os.IsNotExist(err) {
		// 【改造】如果脚本不存在，尝试让 LLM 生成内联代码执行
		return nil, fmt.Errorf("script file does not exist: %s. Consider using inline code execution by setting ScriptPath to 'inline' and providing the code in the 'code' field", input.ScriptPath)
	}

	// Find Python interpreter
	python := "python3"
	if config.PythonPath != "" {
		python = config.PythonPath
	}

	// Prepare command
	cmd := exec.CommandContext(ctx, python, input.ScriptPath)
	cmd.Args = append(cmd.Args, input.Args...)

	// Execute
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run python script '%s' with '%s': %v\nStdout: %s\nStderr: %s",
			input.ScriptPath, python, err, string(output), "")
	}

	return &RunPythonScriptOutput{
		Stdout: string(output),
		Stderr: "",
	}, nil
}

// runInlinePython 执行内联 Python 代码
func runInlinePython(ctx context.Context, config RunnerConfig, code string, args []string) (*RunPythonScriptOutput, error) {
	// Find Python interpreter
	python := "python3"
	if config.PythonPath != "" {
		python = config.PythonPath
	}

	// Prepare command: python3 -c "code" args...
	cmdArgs := []string{"-c", code}
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.CommandContext(ctx, python, cmdArgs...)

	// Execute
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &RunPythonScriptOutput{
			Stdout: string(output),
			Stderr: err.Error(),
		}, fmt.Errorf("failed to run inline python code: %v", err)
	}

	return &RunPythonScriptOutput{
		Stdout: string(output),
		Stderr: "",
	}, nil
}

// isAllowedScript 检查脚本是否在允许列表中
func isAllowedScript(allowed []string, script string) bool {
	if len(allowed) == 0 {
		return true
	}

	// Get absolute path
	absScript, err := filepath.Abs(script)
	if err != nil {
		absScript = script
	}

	for _, allowedScript := range allowed {
		absAllowed, err := filepath.Abs(allowedScript)
		if err != nil {
			absAllowed = allowedScript
		}

		// Check for exact match or if script is in allowed directory
		if absScript == absAllowed || strings.HasPrefix(absScript, absAllowed+string(filepath.Separator)) {
			return true
		}
	}
	return false
}