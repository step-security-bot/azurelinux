// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerlib

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/microsoft/azurelinux/toolkit/tools/imagecustomizerapi"
	"github.com/microsoft/azurelinux/toolkit/tools/internal/logger"
	"github.com/microsoft/azurelinux/toolkit/tools/internal/safechroot"
	"github.com/microsoft/azurelinux/toolkit/tools/internal/safemount"
	"github.com/microsoft/azurelinux/toolkit/tools/internal/shell"
	"golang.org/x/sys/unix"
)

func runUserScripts(baseConfigPath string, scripts []imagecustomizerapi.Script, scriptsName string,
	imageChroot *safechroot.Chroot,
) error {
	if len(scripts) <= 0 {
		return nil
	}

	logger.Log.Infof("Running %s scripts", scriptsName)

	configDirMountPath := filepath.Join(imageChroot.RootDir(), configDirMountPathInChroot)

	// Bind mount the config directory so that the scripts can access any required resources.
	mount, err := safemount.NewMount(baseConfigPath, configDirMountPath, "", unix.MS_BIND|unix.MS_RDONLY, "", true)
	if err != nil {
		return err
	}
	defer mount.Close()

	// Runs scripts.
	for i, script := range scripts {
		err := runUserScript(i, script, scriptsName, imageChroot)
		if err != nil {
			return err
		}
	}

	err = mount.CleanClose()
	if err != nil {
		return err
	}

	return nil
}

func runUserScript(scriptIndex int, script imagecustomizerapi.Script, scriptsName string,
	imageChroot *safechroot.Chroot,
) error {
	var err error

	scriptLogName := createScriptLogName(scriptIndex, script, scriptsName)

	logger.Log.Infof("Running script (%s)", scriptLogName)

	// Collect the process name and args.
	tempScriptFullPath := ""
	process := ""
	args := []string(nil)
	if script.Path != "" {
		scriptPath := filepath.Join(configDirMountPathInChroot, script.Path)

		if script.Interpreter != "" {
			process = script.Interpreter

			args = []string{scriptPath}
			args = append(args, script.Arguments...)
		} else {
			process = scriptPath

			args = script.Arguments
		}
	} else {
		process = script.Interpreter
		if process == "" {
			process = "/bin/sh"
		}

		// Write the script to file.
		tempScriptFullPath, err = createTempScriptFile(script, scriptsName, imageChroot)
		if err != nil {
			return err
		}
		defer os.Remove(tempScriptFullPath)

		// Get the path of the script file in the chroot.
		tempScriptPath, err := filepath.Rel(imageChroot.RootDir(), tempScriptFullPath)
		if err != nil {
			return fmt.Errorf("failed to get relative path for temp script file:\n%w", err)
		}

		// Ensure path is rooted.
		tempScriptPath = filepath.Join("/", tempScriptPath)

		args = []string{tempScriptPath}
		args = append(args, script.Arguments...)
	}

	// Run the script.
	err = imageChroot.UnsafeRun(func() error {
		return shell.ExecuteLiveWithErr(1, process, args...)
	})
	if err != nil {
		return fmt.Errorf("script (%s) failed:\n%w", scriptLogName, err)
	}

	if tempScriptFullPath != "" {
		// Remove the script file and error out if the delete fails.
		err = os.Remove(tempScriptFullPath)
		if err != nil {
			return fmt.Errorf("failed to remove temp script file:\n%w", err)
		}
	}

	return nil
}

func createScriptLogName(scriptIndex int, script imagecustomizerapi.Script, scriptsName string) string {
	switch {
	case script.Name != "" && script.Path != "":
		return fmt.Sprintf("%s(%s)", script.Name, script.Path)
	case script.Name != "":
		return script.Name
	case script.Path != "":
		return script.Path
	default:
		return fmt.Sprintf("%s[%d]", scriptsName, scriptIndex)
	}
}

func createTempScriptFile(script imagecustomizerapi.Script, scriptsName string, imageChroot *safechroot.Chroot,
) (string, error) {
	chrootTempDir := filepath.Join(imageChroot.RootDir(), "tmp")

	// Create a temporary file for the script.
	tempFile, err := os.CreateTemp(chrootTempDir, scriptsName)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file for script:\n%w", err)
	}
	defer tempFile.Close()

	tempFilePath := tempFile.Name()

	// Write the script's content.
	_, err = tempFile.WriteString(script.Content)
	if err != nil {
		return "", fmt.Errorf("failed to write temp file for script:\n%w", err)
	}

	// Ensure the file is written correctly.
	err = tempFile.Close()
	if err != nil {
		return "", fmt.Errorf("failed to close temp file for script:\n%w", err)
	}

	return tempFilePath, nil
}
