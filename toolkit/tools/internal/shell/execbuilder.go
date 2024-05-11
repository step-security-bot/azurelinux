// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package shell

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os/exec"
	"strings"
	"sync"

	"github.com/microsoft/azurelinux/toolkit/tools/internal/logger"
	"github.com/sirupsen/logrus"
)

const (
	LogDisabledLevel    logrus.Level = math.MaxUint32
	DefaultWarnLogLines int          = 1500
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
	errorStderrLines     int
	warnLogLines         int
}

func NewExecBuilder(command string, args ...string) ExecBuilder {
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

func (b ExecBuilder) StdoutLogLevel(stdoutLogLevel logrus.Level) ExecBuilder {
	b.stdoutLogLevel = stdoutLogLevel
	return b
}

func (b ExecBuilder) StderrLogLevel(stderrLogLevel logrus.Level) ExecBuilder {
	b.stderrLogLevel = stderrLogLevel
	return b
}

func (b ExecBuilder) LogLevel(stdoutLogLevel logrus.Level, stderrLogLevel logrus.Level) ExecBuilder {
	b.stdoutLogLevel = stdoutLogLevel
	b.stderrLogLevel = stderrLogLevel
	return b
}

// ErrorStderrLines sets the number of stderr lines to add to the error object, if the execution fails.
func (b ExecBuilder) ErrorStderrLines(lines int) ExecBuilder {
	b.errorStderrLines = lines
	return b
}

func (b ExecBuilder) WarnLogLines(lines int) ExecBuilder {
	b.warnLogLines = lines
	return b
}

func (b ExecBuilder) StdoutCallback(stdoutCallback LogCallback) ExecBuilder {
	b.stdoutCallback = stdoutCallback
	return b
}

func (b ExecBuilder) StderrCallback(stderrCallback LogCallback) ExecBuilder {
	b.stderrCallback = stderrCallback
	return b
}

func (b ExecBuilder) Callbacks(stdoutCallback LogCallback, stderrCallback LogCallback) ExecBuilder {
	b.stdoutCallback = stdoutCallback
	b.stderrCallback = stderrCallback
	return b
}

func (b ExecBuilder) Execute() error {
	_, _, err := b.executeHelper(false /*captureOutput*/)
	return err
}

func (b ExecBuilder) ExecuteCaptureOuput() (string, string, error) {
	return b.executeHelper(true /*captureOutput*/)
}

func (b ExecBuilder) executeHelper(captureOutput bool) (string, string, error) {
	stdoutLinesChans := []chan string(nil)
	stdErrLinesChans := []chan string(nil)

	var warnLogChan chan string
	if b.warnLogLines > 0 {
		warnLogChan = make(chan string, b.warnLogLines)
		stdoutLinesChans = append(stdoutLinesChans, warnLogChan)
		stdErrLinesChans = append(stdErrLinesChans, warnLogChan)
	}

	var errorChan chan string
	if b.errorStderrLines > 0 {
		errorChan = make(chan string, b.errorStderrLines)
		stdErrLinesChans = append(stdErrLinesChans, errorChan)
	}

	stdoutResultChan := chan string(nil)
	stderrResultChan := chan string(nil)
	if captureOutput {
		stdoutResultChan = make(chan string, 1)
		stderrResultChan = make(chan string, 1)
	}

	// Setup process.
	cmd := exec.Command(b.command, b.args...)
	cmd.Dir = b.workingDirectory
	cmd.Env = b.environmentVariables

	if b.stdinString != "" {
		cmd.Stdin = strings.NewReader(b.stdinString)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		err = fmt.Errorf("failed to open stdout pipe:\n%w", err)
		return "", "", err
	}
	defer stdoutPipe.Close()

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		err = fmt.Errorf("failed to open stderr pipe:\n%w", err)
		return "", "", err
	}
	defer stderrPipe.Close()

	// Start process.
	err = trackAndStartProcess(cmd)
	if err != nil {
		err = fmt.Errorf("failed to start process:\n%w", err)
		return "", "", err
	}

	defer untrackProcess(cmd)

	wg := new(sync.WaitGroup)
	wg.Add(2)

	// Read stdout and stderr.
	go readExecPipe(stdoutPipe, wg, b.stdoutCallback, b.stdoutLogLevel, stdoutLinesChans, stdoutResultChan)
	go readExecPipe(stderrPipe, wg, b.stderrCallback, b.stderrLogLevel, stdErrLinesChans, stderrResultChan)

	// Wait for process to exit.
	wg.Wait()
	err = cmd.Wait()

	// Cleanup the lines channels.
	// Note: While technically senders are suppose to close channels, it is to do it here because of the use of the
	// waitgroup (wg).
	if warnLogChan != nil {
		close(warnLogChan)
	}

	if errorChan != nil {
		close(errorChan)
	}

	stdout := ""
	stderr := ""
	if captureOutput {
		stdout = <-stdoutResultChan
		stderr = <-stderrResultChan
	}

	if err != nil {
		if warnLogChan != nil {
			// Report last x lines of process's output (stderr and stdout) as warning logs.
			logger.Log.Errorf("Call to %s returned error, last %d lines of output:", b.command, b.warnLogLines)
			for line := range warnLogChan {
				logger.Log.Warn(line)
			}
		}

		if errorChan != nil {
			// Add last x line from stderr to the error message.
			builder := strings.Builder{}
			for errLine := range errorChan {
				if builder.Len() > 0 {
					builder.WriteString("\n")
				}
				builder.WriteString(errLine)
			}

			errLines := builder.String()
			if errLines != "" {
				err = fmt.Errorf("%s\n%w", errLines, err)
			}
		}
	}

	return stdout, stderr, err
}

func readExecPipe(pipe io.Reader, wg *sync.WaitGroup, logCallback LogCallback, logLevel logrus.Level,
	linesOutputChans []chan string, outputResultChan chan string,
) {
	defer wg.Done()

	outputBuilder := strings.Builder{}

	reader := bufio.NewReader(pipe)
	for {
		// Read up to the next line.
		bytes, err := reader.ReadBytes('\n')

		// Drop \n or \r\n from line.
		omitBytes := 0
		if len(bytes) >= 1 && bytes[len(bytes)-1] == '\n' {
			omitBytes = 1
			if len(bytes) >= 2 && bytes[len(bytes)-2] == '\r' {
				omitBytes = 2
			}
		}

		line := string(bytes[:len(bytes)-omitBytes])

		if logCallback != nil {
			// Call user callback.
			logCallback(line)
		}

		if logLevel <= logrus.TraceLevel {
			// Log the line.
			logger.Log.Log(logLevel, line)
		}

		for _, linesOutputChan := range linesOutputChans {
			channelDropAndPush(line, linesOutputChan)
		}

		if outputResultChan != nil {
			// Collect the entire stream into a single string.
			outputBuilder.Write(bytes)
		}

		if err != nil {
			break
		}
	}

	if outputResultChan != nil {
		// Return the full stream as a string.
		output := outputBuilder.String()
		outputResultChan <- output
		close(outputResultChan)
	}
}

func channelDropAndPush(line string, outputChan chan string) {
	const maxRetries = 8

	for i := 0; i < maxRetries; i++ {
		if len(outputChan) == cap(outputChan) {
			// The buffer is full, discard the oldest value.
			select {
			case <-outputChan:
			default:
			}
		}

		select {
		case outputChan <- line:
			// Line was pushed.
			return

		default:
			// The event buffer is full, presumably from another goroutine pushing an entry.
			// So, loop back around and try again.
		}
	}
}
