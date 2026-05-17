package main

import (
	"os/exec"
	"testing"
)

func TestMainCompileAndRun(t *testing.T) {
	buildCmd := exec.Command("go", "build", "-o", "client.testbin")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("failed to build client: %v", err)
	}
	defer exec.Command("rm", "client.testbin").Run()

	runCmd := exec.Command("./client.testbin")
	runCmd.Env = append(runCmd.Environ(), "SERVER_ADDR=localhost:0")
	
	err := runCmd.Run()
	if err == nil {
		t.Error("expected process to fail connecting to invalid port, but it succeeded")
	}
}
