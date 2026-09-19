package cli

import (
	"errors"
	"reflect"
	"testing"
)

func TestSelectBrowserCommandUsesConfiguredBrowser(t *testing.T) {
	t.Parallel()
	command, err := selectBrowserCommand(
		"linux",
		func(key string) string {
			if key == "BROWSER" {
				return "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
			}
			return ""
		},
		func(string) (string, error) { return "", errors.New("not found") },
		"https://example.test",
	)
	if err != nil {
		t.Fatal(err)
	}
	if command.name != "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" || !reflect.DeepEqual(command.args, []string{"https://example.test"}) {
		t.Fatalf("command = %#v", command)
	}
}

func TestSelectBrowserCommandUsesWSLView(t *testing.T) {
	t.Parallel()
	command, err := selectBrowserCommand(
		"linux",
		func(key string) string {
			if key == "WSL_DISTRO_NAME" {
				return "Ubuntu"
			}
			return ""
		},
		func(name string) (string, error) {
			if name == "wslview" {
				return "/usr/bin/wslview", nil
			}
			return "", errors.New("not found")
		},
		"https://example.test",
	)
	if err != nil {
		t.Fatal(err)
	}
	if command.name != "/usr/bin/wslview" {
		t.Fatalf("command = %#v", command)
	}
}

func TestSelectBrowserCommandUsesMountedWindowsLauncher(t *testing.T) {
	t.Parallel()
	command, err := selectBrowserCommand(
		"linux",
		func(key string) string {
			if key == "WSL_INTEROP" {
				return "/run/WSL/interop"
			}
			return ""
		},
		func(name string) (string, error) {
			if name == "/mnt/c/Windows/System32/rundll32.exe" {
				return name, nil
			}
			return "", errors.New("not found")
		},
		"https://example.test",
	)
	if err != nil {
		t.Fatal(err)
	}
	if command.name != "/mnt/c/Windows/System32/rundll32.exe" {
		t.Fatalf("command = %#v", command)
	}
}

func TestSelectBrowserCommandFailsWithoutLinuxLauncher(t *testing.T) {
	t.Parallel()
	_, err := selectBrowserCommand(
		"linux",
		func(string) string { return "" },
		func(string) (string, error) { return "", errors.New("not found") },
		"https://example.test",
	)
	if err == nil {
		t.Fatal("selectBrowserCommand succeeded without a browser launcher")
	}
}
