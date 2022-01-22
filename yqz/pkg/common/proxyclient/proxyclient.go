//
package proxyclient

import (
	gmclient "git.code.oa.com/gongyi/gomore/clients/grpc_client"
)

type ClientType int32

const (
	Type_Unknown ClientType = iota
	Type_Sidecar
	Type_Polaris
	Type_Edproxy
)

func (p ClientType) String() string {
	switch p {
	case Type_Polaris:
		return "Polaris"
	case Type_Edproxy:
		return "Edproxy"
	case Type_Sidecar:
		return "Sidecar"
	default:
		return "UNKNOWN"
	}
}

type ServiceConfig struct {
	PolarisEnv      string `yaml:"polaris_env"`
	FullPolarisAddr string `yaml:"full_polaris_addr"`
	PolarisAddr     string `yaml:"polaris_addr"`
}

type ProxyConfig struct {
	Flag      bool   `yaml:"flag"`
	ProxyType string `yaml:"type"`
	Addr      string `yaml:"addr"`
	PollInit  int    `yaml:"poll_init"`
	PollIdle  int    `yaml:"poll_idle"`
	PollPeak  int    `yaml:"poll_peak"`
}

// InitSidecarPorxy init sidecar
func InitSidecarPorxy(conf *ProxyConfig) error {
	if err := gmclient.InitGRPCClient(int32(conf.PollInit), int32(conf.PollIdle), int32(conf.PollPeak)); err != nil {
		return err
	}
	gmclient.SetSidecarAddr(conf.Addr)
	return nil
}

// InitEdProxy init edproxy
func InitEdProxy(conf *ProxyConfig) error {
	if err := gmclient.InitGRPCClient(int32(conf.PollInit), int32(conf.PollIdle), int32(conf.PollPeak)); err != nil {
		return err
	}
	gmclient.SetEdProxyAddr(conf.Addr)
	return nil
}
