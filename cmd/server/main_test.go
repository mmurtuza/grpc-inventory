package main

import (
	"os/exec"
	"testing"
)

func TestMainCompileAndRun(t *testing.T) {
	buildCmd := exec.Command("go", "build", "-o", "server.testbin")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("failed to build server: %v", err)
	}
	defer exec.Command("rm", "server.testbin").Run()

	runCmd := exec.Command("./server.testbin")
	// Try binding to an invalid port to trigger early exit
	runCmd.Env = append(runCmd.Environ(), "GRPC_PORT=-1")
	
	err := runCmd.Run()
	if err == nil {
		t.Error("expected process to fail with invalid port, but it succeeded")
	}
}
