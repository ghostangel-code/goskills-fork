package tool

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ShellConfig 配置
 type ShellConfig struct {
	AllowedScripts []string
}

// RunShellScriptInput 输入
 type RunShellScriptInput struct {
	ScriptPath string   `json:"scriptPath" jsonschema:"description=Path to the shell script to execute"`
	Args       []string `json:"args,omitempty" jsonschema:"description=Arguments to pass to the script"`
}

// RunShellScriptOutput 输出
 type RunShellScriptOutput struct {
	Stdout string `json:"stdout" jsonschema:"description=Standard output from the script"`
	Stderr string `json:"stderr" jsonschema:"description=Standard error from the script"`
}

// RunShellScript 执行 shell 脚本或命令
 func RunShellScript(ctx context.Context, config RunnerConfig, input RunShellScriptInput) (*RunShellScriptOutput, error) {
	// 【修复】检查 scriptPath 是否是解释器（如 /bin/bash, /bin/sh）
	// 如果是解释器，且 args 包含 -c，则直接执行命令
	if isShellInterpreter(input.ScriptPath) && len(input.Args) >= 2 && input.Args[0] == "-c" {
		// 这是解释器 + 命令的形式，直接执行
		cmd := exec.CommandContext(ctx, input.ScriptPath, input.Args...)
		output, err := cmd.CombinedOutput()
		
		return &RunShellScriptOutput{
			Stdout: string(output),
			Stderr: "",
		}, err
	}
	
	// 原有的脚本文件执行逻辑
	if !isAllowedScript(config.AllowedScripts, input.ScriptPath) {
		return nil, fmt.Errorf("script path '%s' is not in the allowed list. allowed scripts: %v", 
			input.ScriptPath, config.AllowedScripts)
	}

	// Check if script exists
	if _, err := os.Stat(input.ScriptPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("script file does not exist: %s", input.ScriptPath)
	}

	// Determine shell interpreter
	shell := "/bin/sh"
	if runtime.GOOS == "windows" {
		shell = "cmd"
	} else {
		// Try to use bash if available
		if _, err := exec.LookPath("bash"); err == nil {
			shell = "bash"
		}
	}

	// Prepare command
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, shell, "/c", input.ScriptPath)
	} else {
		cmd = exec.CommandContext(ctx, shell, input.ScriptPath)
	}

	// Add arguments
	if len(input.Args) > 0 {
		cmd.Args = append(cmd.Args, input.Args...)
	}

	// Execute
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run shell script '%s': %v\nStdout: %s\nStderr: %s", 
			input.ScriptPath, err, string(output), "")
	}

	return &RunShellScriptOutput{
		Stdout: string(output),
		Stderr: "",
	}, nil
}

// isShellInterpreter 检查路径是否是 shell 解释器
 func isShellInterpreter(path string) bool {
	interpreters := []string{"/bin/bash", "/bin/sh", "/usr/bin/bash", "/usr/bin/sh", "/bin/zsh"}
	for _, interp := range interpreters {
		if path == interp {
			return true
		}
	}
	return false
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