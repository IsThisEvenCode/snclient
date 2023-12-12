package main

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDaemonRequestsLinux(t *testing.T) {
	bin := getBinary()
	require.FileExistsf(t, bin, "snclient binary must exist")

	writeFile(t, `snclient.ini`, localDaemonINI)

	startBackgroundDaemon(t)

	baseUrl := fmt.Sprintf("http://127.0.0.1:%d", localDaemonPort)

	runCmd(t, &cmd{
		Cmd:  bin,
		Args: []string{"run", "check_nsc_web", "-p", localDaemonPassword, "-r", "-u", baseUrl + "/api/v1/admin/reload"},
		Like: []string{`RESPONSE-ERROR: http request failed: 403 Forbidden`},
		Exit: 3,
	})

	runCmd(t, &cmd{
		Cmd:  "curl",
		Args: []string{"-s", "-u", "user:" + localDaemonAdminPassword, "-k", baseUrl + "/api/v1/admin/reload"},
		Like: []string{`POST method required`},
	})

	runCmd(t, &cmd{
		Cmd:  "curl",
		Args: []string{"-s", "-u", "user:" + localDaemonAdminPassword, "-k", "-X", "POST", baseUrl + "/api/v1/admin/reload"},
		Like: []string{`{"success":true}`},
	})

	postData, _ := json.Marshal(map[string]string{
		"Unknown": "false",
	})
	runCmd(t, &cmd{
		Cmd:  "curl",
		Args: []string{"-s", "-u", "user:" + localDaemonAdminPassword, "-k", "-s", "-d", string(postData), baseUrl + "/api/v1/admin/certs/replace"},
		Like: []string{`unknown field`},
	})

	// test replacing certificates
	os.WriteFile("test.crt", []byte{}, 0o600)
	os.WriteFile("test.key", []byte{}, 0o600)
	postData, _ = json.Marshal(map[string]interface{}{
		"Reload":   true,
		"CertData": "dGVzdGNlcnQ=",
		"KeyData":  "dGVzdGtleQ==",
	})
	runCmd(t, &cmd{
		Cmd:  "curl",
		Args: []string{"-s", "-u", "user:" + localDaemonAdminPassword, "-k", "-s", "-d", string(postData), baseUrl + "/api/v1/admin/certs/replace"},
		Like: []string{`{"success":true}`},
	})
	crt, _ := os.ReadFile("test.crt")
	key, _ := os.ReadFile("test.key")

	assert.Equalf(t, "testcert", string(crt), "test certificate written")
	assert.Equalf(t, "testkey", string(key), "test certificate key written")

	stopBackgroundDaemon(t)
	os.Remove("snclient.ini")
	os.Remove("test.crt")
	os.Remove("test.key")
}