package main

import (
	"os/exec"
	"testing"
)

func TestMainCompileAndRun(t *testing.T) {
	// Build the binary
	buildCmd := exec.Command("go", "build", "-o", "rest-api.testbin")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("failed to build rest-api: %v", err)
	}
	defer exec.Command("rm", "rest-api.testbin").Run()

	// Run with bad config to ensure it exits with an error code (DB connection fail)
	runCmd := exec.Command("./rest-api.testbin")
	runCmd.Env = append(runCmd.Environ(), "DATABASE_URL=postgres://invalid:invalid@localhost:0/invalid")
	
	err := runCmd.Run()
	if err == nil {
		t.Error("expected process to fail with invalid DB URL, but it succeeded")
	}
}
