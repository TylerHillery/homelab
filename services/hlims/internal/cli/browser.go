package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type browserCommand struct {
	name string
	args []string
}

func openBrowser(ctx context.Context, target string) error {
	command, err := selectBrowserCommand(runtime.GOOS, os.Getenv, exec.LookPath, target)
	if err != nil {
		return err
	}
	output, err := exec.CommandContext(ctx, command.name, command.args...).CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message != "" {
			return fmt.Errorf("open browser: %w: %s", err, message)
		}
		return fmt.Errorf("open browser: %w", err)
	}
	return nil
}

func selectBrowserCommand(
	goos string,
	getenv func(string) string,
	lookPath func(string) (string, error),
	target string,
) (browserCommand, error) {
	if configured := getenv("BROWSER"); configured != "" {
		return browserCommand{name: configured, args: []string{target}}, nil
	}

	switch goos {
	case "darwin":
		return browserCommand{name: "open", args: []string{target}}, nil
	case "windows":
		return browserCommand{name: "rundll32", args: []string{"url.dll,FileProtocolHandler", target}}, nil
	case "linux":
		if getenv("WSL_DISTRO_NAME") != "" || getenv("WSL_INTEROP") != "" {
			if executable, err := lookPath("wslview"); err == nil {
				return browserCommand{name: executable, args: []string{target}}, nil
			}
			for _, candidate := range []string{"rundll32.exe", "/mnt/c/Windows/System32/rundll32.exe"} {
				if executable, err := lookPath(candidate); err == nil {
					return browserCommand{name: executable, args: []string{"url.dll,FileProtocolHandler", target}}, nil
				}
			}
		}
		if executable, err := lookPath("xdg-open"); err == nil {
			return browserCommand{name: executable, args: []string{target}}, nil
		}
		return browserCommand{}, errors.New("no browser launcher found; install xdg-open or set BROWSER")
	default:
		return browserCommand{}, fmt.Errorf("opening a browser is unsupported on %s; use hlims open --print", goos)
	}
}
