//go:build mage

// Package main is the Mage build file for gollm.
// Usage: mage <target>
package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// Build the gollm binary.
func Build() error {
	return run("go", "build", "-o", binaryPath(), "./cmd/glm")
}

// Run tests with coverage.
func Test() error {
	args := []string{"test", "-v", "./..."}
	if os.Getenv("COVERAGE") != "" {
		args = append([]string{"test", "-coverprofile=coverage.out", "-v", "./..."})
	}
	return run("go", args...)
}

// Vet checks for static analysis issues.
func Vet() error {
	return run("go", "vet", "./...")
}

// Lint runs golangci-lint.
func Lint() error {
	return run("golangci-lint", "run", "./...")
}

// Clean removes build artifacts.
func Clean() error {
	os.Remove("glm")
	os.Remove("coverage.out")
	return nil
}

// Format runs gofmt.
func Format() error {
	return run("go", "fmt", "./...")
}

// Tidy runs go mod tidy.
func Tidy() error {
	return run("go", "mod", "tidy")
}

// Generate runs protoc to generate Go gRPC stubs for extensions.
func Generate() error {
	return run("protoc",
		"--go_out=.",
		"--go_opt=paths=source_relative",
		"--go-grpc_out=.",
		"--go-grpc_opt=paths=source_relative",
		"extensions/proto/extension.proto",
	)
}

// All runs build, test, vet, and lint.
func All() error {
	if err := Build(); err != nil {
		return err
	}
	if err := Test(); err != nil {
		return err
	}
	if err := Vet(); err != nil {
		return err
	}
	if err := Lint(); err != nil {
		return err
	}
	fmt.Println("✅ all checks passed")
	return nil
}

// Install builds and copies to GOPATH/bin.
func Install() error {
	return run("go", "install", "./cmd/glm")
}

// Run builds and executes gollm with the given arguments.
func Run(args ...string) error {
	if err := Build(); err != nil {
		return err
	}
	cmd := exec.Command(binaryPath(), args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func binaryPath() string {
	if runtime.GOOS == "windows" {
		return "glm.exe"
	}
	return "glm"
}

func run(name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
