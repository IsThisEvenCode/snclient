package snclient

import (
	"net"
	"strings"

	"pkg/nrpe"
)

func init() {
	RegisterModule(&AvailableListeners, "NRPEServer", "/settings/NRPE/server", NewHandlerNRPE)
}

const NastyCharacters = "|`&><'\"\\[]{}"

type HandlerNRPE struct {
	noCopy   noCopy
	snc      *Agent
	conf     *ConfigSection
	listener *Listener
}

func NewHandlerNRPE() Module {
	return &HandlerNRPE{}
}

func (l *HandlerNRPE) Defaults() ConfigData {
	defaults := ConfigData{
		"allow arguments":        "0",
		"allow nasty characters": "0",
		"port":                   "5666",
	}
	defaults.Merge(DefaultListenTCPConfig)

	return defaults
}

func (l *HandlerNRPE) Init(snc *Agent, conf *ConfigSection, _ *Config) error {
	l.snc = snc
	l.conf = conf
	listener, err := NewListener(snc, conf, l)
	if err != nil {
		return err
	}
	l.listener = listener

	return nil
}

func (l *HandlerNRPE) Type() string {
	return "nrpe"
}

func (l *HandlerNRPE) BindString() string {
	return l.listener.BindString()
}

func (l *HandlerNRPE) Start() error {
	return l.listener.Start()
}

func (l *HandlerNRPE) Stop() {
	l.listener.Stop()
}

func (l *HandlerNRPE) ServeTCP(snc *Agent, con net.Conn) {
	defer con.Close()

	request, err := nrpe.ReadNrpePacket(con)
	if err != nil {
		log.Errorf("nrpe protocol error: %s", err.Error())

		return
	}

	if err := request.Verify(nrpe.NrpeQueryPacket); err != nil {
		log.Errorf("nrpe protocol error: %s", err.Error())

		return
	}

	cmd, args := request.Data()
	log.Tracef("nrpe v%d request: %s %s", request.Version(), cmd, args)

	if request.Version() == nrpe.NrpeV3PacketVersion {
		log.Errorf("nrpe protocol version 3 is deprecated, use v2 or v4")

		return
	}

	var statusResult *CheckResult

	switch {
	case !checkAllowArguments(l.conf, args):
		statusResult = &CheckResult{
			State:  CheckExitUnknown,
			Output: "Exception processing request: Request contained arguments (check the allow arguments option).",
		}
	case !checkNastyCharacters(l.conf, cmd, args):
		statusResult = &CheckResult{
			State:  CheckExitUnknown,
			Output: "Exception processing request: Request contained illegal characters (check the allow nasty characters option).",
		}
	case cmd == "_NRPE_CHECK":
		// version check
		statusResult = snc.RunCheck("check_snclient_version", args)
	default:
		statusResult = snc.RunCheck(cmd, args)
	}

	output := statusResult.BuildPluginOutput()
	response := nrpe.BuildPacket(request.Version(), nrpe.NrpeResponsePacket, uint16(statusResult.State), output)

	if err := response.Write(con); err != nil {
		log.Errorf("nrpe write response error: %s", err.Error())

		return
	}
}

func checkAllowArguments(conf *ConfigSection, args []string) bool {
	allowed, _, err := conf.GetBool("allow arguments")
	if err != nil {
		log.Errorf("config error: %s", err.Error())

		return false
	}

	if allowed {
		return true
	}

	return len(args) == 0
}

func checkNastyCharacters(conf *ConfigSection, cmd string, args []string) bool {
	allowed, _, err := conf.GetBool("allow nasty characters")
	if err != nil {
		log.Errorf("config error: %s", err.Error())

		return false
	}

	if allowed {
		return true
	}

	if strings.ContainsAny(cmd, NastyCharacters) {
		log.Debugf("command string contained nasty character", cmd)

		return false
	}

	for i, arg := range args {
		if strings.ContainsAny(arg, NastyCharacters) {
			log.Debugf("cmd arg (#%d) contained nasty character", i)

			return false
		}
	}

	return true
}