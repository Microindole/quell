package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	var mode string
	var gosecSARIF string

	flag.StringVar(&mode, "mode", "all", "check mode: quick|quality|security|all")
	flag.StringVar(&gosecSARIF, "gosec-sarif", "", "output file path for gosec SARIF report")
	flag.Parse()

	if mode == "all" && flag.NArg() > 0 {
		mode = normalizeMode(flag.Arg(0))
	} else {
		mode = normalizeMode(mode)
	}

	root, err := projectRoot()
	if err != nil {
		fail("resolve project root", err)
	}
	if err := os.Chdir(root); err != nil {
		fail("change directory", err)
	}

	switch mode {
	case "quick":
		runQuick()
	case "quality":
		runQuality()
	case "security":
		runSecurity(gosecSARIF)
	case "all":
		runQuality()
		runSecurity(gosecSARIF)
	default:
		failf("unknown mode: %s", mode)
	}

	fmt.Printf("All checks completed for mode: %s\n", mode)
}

func normalizeMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "q":
		return "quick"
	case "qa":
		return "quality"
	case "sec":
		return "security"
	default:
		return strings.ToLower(strings.TrimSpace(mode))
	}
}

func runQuick() {
	step("gofmt", checkGofmt)
	step("go mod tidy consistency", checkGoModTidy)
	step("go vet", func() { runOrFail("go", "vet", "./...") })
	step("go test", func() { runOrFail("go", "test", "-count=1", "./...") })
}

func runQuality() {
	runQuick()
	step("go test -race", runRaceIfSupported)
	step("go build", func() { runOrFail("go", "build", "./...") })
	step("staticcheck", func() {
		runOrFail("go", "run", "honnef.co/go/tools/cmd/staticcheck@latest", "./...")
	})
	step("golangci-lint", func() {
		runOrFail("go", "run", "github.com/golangci/golangci-lint/cmd/golangci-lint@latest", "run", "--timeout", "5m")
	})
}

func runRaceIfSupported() {
	if !isRaceSupported() {
		fmt.Println("Skipping race test because CGO is disabled in current environment.")
		return
	}
	runOrFail("go", "test", "-race", "-count=1", "./...")
}

func isRaceSupported() bool {
	out := strings.TrimSpace(runCaptureOrFail("go", "env", "CGO_ENABLED"))
	return out == "1"
}

func runSecurity(gosecSARIF string) {
	step("govulncheck", func() {
		runOrFail("go", "run", "golang.org/x/vuln/cmd/govulncheck@latest", "./...")
	})
	step("gosec", func() {
		args := []string{
			"run",
			"github.com/securego/gosec/v2/cmd/gosec@latest",
			"-exclude",
			"G104,G204,G301,G302,G304,G306,G401,G501,G703",
		}
		if gosecSARIF != "" {
			if err := os.MkdirAll(filepath.Dir(gosecSARIF), 0o755); err != nil {
				fail("prepare sarif directory", err)
			}
			args = append(args, "-fmt", "sarif", "-out", gosecSARIF)
		}
		args = append(args, "./...")
		runOrFail("go", args...)
	})
}

func checkGofmt() {
	out := runCaptureOrFail("gofmt", "-l", ".")
	trimmed := strings.TrimSpace(out)
	if trimmed != "" {
		failf("unformatted Go files detected:\n%s", trimmed)
	}
}

func checkGoModTidy() {
	beforeMod, err := os.ReadFile("go.mod")
	if err != nil {
		fail("read go.mod before tidy", err)
	}
	beforeSum, err := os.ReadFile("go.sum")
	if err != nil {
		fail("read go.sum before tidy", err)
	}

	runOrFail("go", "mod", "tidy")

	afterMod, err := os.ReadFile("go.mod")
	if err != nil {
		fail("read go.mod after tidy", err)
	}
	afterSum, err := os.ReadFile("go.sum")
	if err != nil {
		fail("read go.sum after tidy", err)
	}
	if !bytes.Equal(beforeMod, afterMod) || !bytes.Equal(beforeSum, afterSum) {
		failf("go.mod/go.sum changed after 'go mod tidy'. Please commit tidy results.")
	}
}

func step(name string, fn func()) {
	fmt.Printf("[check] %s\n", name)
	fn()
}

//nolint:unparam // kept generic for non-go command execution.
func runOrFail(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		failf("command failed: %s %s\n%v", name, strings.Join(args, " "), err)
	}
}

func runCaptureOrFail(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		failf("command failed: %s %s\n%v", name, strings.Join(args, " "), err)
	}
	return stdout.String()
}

func projectRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return wd, nil
}

func fail(action string, err error) {
	failf("%s: %v", action, err)
}

func failf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
