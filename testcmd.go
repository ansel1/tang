package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
)

var valueTangFlags = map[string]bool{
	"f": true, "outfile": true, "jsonfile": true, "junitfile": true,
	"slow-threshold": true, "rate": true,
}

func parseFlagArg(arg string) (name, value string, isFlag bool) {
	if len(arg) == 0 || arg[0] != '-' {
		return "", "", false
	}
	s := arg[1:]
	if len(s) > 0 && s[0] == '-' {
		s = s[1:]
	}
	if idx := bytes.IndexByte([]byte(s), '='); idx != -1 {
		return s[:idx], s[idx+1:], true
	}
	return s, "", true
}

func validatePreTestArgs(args []string) error {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			break
		}
		flagName, flagValue, isFlag := parseFlagArg(arg)
		if !isFlag {
			continue
		}

		if flagName == "f" {
			return errors.New("Error: -f is not compatible with 'test' subcommand")
		}
		if flagName == "replay" {
			return errors.New("Error: -replay is not compatible with 'test' subcommand")
		}
		if flagName == "rate" {
			if flagValue == "" && i+1 < len(args) {
				flagValue = args[i+1]
				i++
			}
			if flagValue != "" {
				rate, err := strconv.ParseFloat(flagValue, 64)
				if err == nil && rate != 1.0 {
					return errors.New("Error: -rate is not compatible with 'test' subcommand")
				}
			}
		}

		if valueTangFlags[flagName] && flagValue == "" {
			i++
		}
	}

	return nil
}

func scanForTestSubcommand() int {
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg == "--" {
			break
		}
		flagName, _, isFlag := parseFlagArg(arg)
		if isFlag {
			if valueTangFlags[flagName] {
				i++
			}
			continue
		}
		if arg == "test" {
			return i
		}
	}
	return -1
}

func splitTestArgs(allArgs []string) (tangArgs []string, goTestArgs []string, hasVerbose bool) {
	foundTest := false
	for _, arg := range allArgs {
		if !foundTest {
			if arg == "test" {
				foundTest = true
				continue
			}
			tangArgs = append(tangArgs, arg)
		} else {
			if arg == "-v" {
				hasVerbose = true
			}
			goTestArgs = append(goTestArgs, arg)
		}
	}
	return tangArgs, goTestArgs, hasVerbose
}

type goTestProcess struct {
	cmd    *exec.Cmd
	stdout io.ReadCloser
}

func startGoTest(goTestArgs []string) (*goTestProcess, error) {
	args := []string{"test"}

	hasJSON := false
	for _, arg := range goTestArgs {
		if arg == "-json" {
			hasJSON = true
		}
	}
	if !hasJSON {
		args = append(args, "-json")
	}
	args = append(args, goTestArgs...)

	cmd := exec.Command("go", args...)
	configureProcessGroup(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("Error creating stdout pipe: %w", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("Error starting go test: %w", err)
	}

	return &goTestProcess{cmd: cmd, stdout: stdout}, nil
}

func (p *goTestProcess) wait() int {
	if err := p.cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		return 1
	}
	return 0
}

func (p *goTestProcess) signal(sig os.Signal) error {
	return signalProcessGroup(p.cmd, sig)
}

func (p *goTestProcess) cleanup() {
	killProcessGroup(p.cmd)
}
