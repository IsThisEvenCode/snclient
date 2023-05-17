package snclient

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func init() {
	setLogLevel("error")
}

// Starts a full Agent from given config
func StartTestAgent(t *testing.T, config string, args []string) *Agent {
	t.Helper()
	tmpConfig, err := os.CreateTemp("", "testconfig")
	assert.NoErrorf(t, err, "tmp config created")
	_, err = tmpConfig.WriteString(config)
	assert.NoErrorf(t, err, "tmp config written")
	err = tmpConfig.Close()
	assert.NoErrorf(t, err, "tmp config created")
	defer os.Remove(tmpConfig.Name())

	tmpPidfile, err := os.CreateTemp("", "testpid")
	assert.NoErrorf(t, err, "tmp pidfile created")
	tmpPidfile.Close()
	os.Remove(tmpPidfile.Name())

	osArgs := []string{
		fmt.Sprintf("--config=%s", tmpConfig.Name()),
		fmt.Sprintf("--pidfile=%s", tmpPidfile.Name()),
	}
	osArgs = append(osArgs, args...)
	snc := NewAgent("test", "0", osArgs)
	started := snc.StartWait(10 * time.Second)
	assert.Truef(t, started, "agent is started successfully")
	if !started {
		t.Fatalf("agent did not start")
	}

	return snc
}

// Stops the agent started by StartTestAgent
func StopTestAgent(t *testing.T, snc *Agent) {
	t.Helper()
	stopped := snc.StopWait(10 * time.Second)
	assert.Truef(t, stopped, "agent stopped successfully")
	if !stopped {
		t.Fatalf("agent did not stop")
	}
}