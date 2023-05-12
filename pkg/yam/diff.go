package yam

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

type DiffHandler func(want, got []byte) error

func ExecDiff(want, got []byte) error {
	handlerOutput := os.Stderr

	tempWant, err := os.CreateTemp("", "want-*")
	if err != nil {
		return err
	}
	defer os.Remove(tempWant.Name())

	tempGot, err := os.CreateTemp("", "got-*")
	if err != nil {
		return err
	}
	defer os.Remove(tempGot.Name())

	_, err = tempWant.Write(want)
	if err != nil {
		return err
	}

	_, err = tempGot.Write(got)
	if err != nil {
		return err
	}

	const command = "diff"
	_, err = exec.LookPath(command)
	if err != nil {
		return fmt.Errorf("unable to execute diff command: %w", err)
	}

	cmd := exec.Command(
		command,
		"-U",
		"5",
		"--label",
		"want",
		"--label",
		"got",
		tempWant.Name(),
		tempGot.Name(),
	)
	cmd.Stdout = handlerOutput
	cmd.Stderr = handlerOutput

	err = cmd.Run()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			if exitError.ExitCode() == 1 {
				// that's expected for `diff` when there's a diff!

				fmt.Fprint(handlerOutput, "\n") // for gap between diffs; useful if diffing multiple files.

				return nil
			}
		}

		return fmt.Errorf("error executing diff command %q: %w", cmd, err)
	}

	return nil
}
