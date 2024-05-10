// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package shell

import (
	"github.com/sirupsen/logrus"
)

type LogCallback func(line string)

type ExecBuilder struct {
	command              string
	args                 []string
	workingDirectory     string
	environmentVariables []string
	stdinString          string
	stdoutLogLevel       logrus.Level
	stderrLogLevel       logrus.Level
	stdoutCallback       LogCallback
	stderrCallback       LogCallback
	stdoutFilePath       string
	errorStderrLines     int
	warnLogLines         int
}

func NewExecBuilder(command string, args []string) ExecBuilder {
	b := ExecBuilder{
		command:        command,
		args:           args,
		stdoutLogLevel: logrus.DebugLevel,
		stderrLogLevel: logrus.DebugLevel,
	}
	return b
}

func (b ExecBuilder) WorkingDirectory(path string) ExecBuilder {
	b.workingDirectory = path
	return b
}

func (b ExecBuilder) EnvironmentVariables(environmentVariables []string) ExecBuilder {
	b.environmentVariables = environmentVariables
	return b
}

func (b ExecBuilder) Stdin(value string) ExecBuilder {
	b.stdinString = value
	return b
}

func (b ExecBuilder) LogLevel(stdoutLogLevel logrus.Level, stderrLogLevel logrus.Level) ExecBuilder {
	b.stdoutLogLevel = stdoutLogLevel
	b.stderrLogLevel = stderrLogLevel
	return b
}

func (b ExecBuilder) StdoutToFile(path string) ExecBuilder {
	b.stdoutFilePath = path
	return b
}

// ErrorStderrLines sets the number of stderr lines to add to the error object, if the execution fails.
func (b ExecBuilder) ErrorStderrLines(lines int) ExecBuilder {
	b.errorStderrLines = lines
	return b
}

// WarnLogLines sets the number of stdout/stderr lines to log as WARN, if the execution fails.
func (b ExecBuilder) WarnLogLines(lines int) ExecBuilder {
	b.warnLogLines = lines
	return b
}

func (b ExecBuilder) Execute() error {

}

func (b ExecBuilder) ExecuteAndCapture() (string, string, error) {

}
