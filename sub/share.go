package sub

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"strconv"
	"strings"
	"vc/vc"
)

type Endpoint interface {
	Tag() string
	Share() string
	CheckPort() int
	SetCheckPort(p int)
	Outbound() *vc.Outbound
}

type SsEndpoint struct {
	tag       string
	share     string
	checkPort int
	method    string
	password  string
	address   string
	port      int64
}

func (e *SsEndpoint) Tag() string {
	return e.tag
}

func (e *SsEndpoint) Share() string {
	return e.share
}

func (e *SsEndpoint) CheckPort() int {
	return e.checkPort
}

func (e *SsEndpoint) SetCheckPort(p int) {
	e.checkPort = p
}

func (e *SsEndpoint) Outbound() *vc.Outbound {
	return &vc.Outbound{
		SendThrough: "0.0.0.0",
		Protocol:    "shadowsocks",
		Settings: &vc.OutboundCommonSettings{
			Servers: []*vc.SsServer{
				{
					Address:  e.address,
					Port:     e.port,
					Method:   e.method,
					Password: e.password,
				},
			},
		},
		Tag:            e.tag,
		StreamSettings: &vc.StreamSettings{},
		Mux:            &vc.Mux{},
	}
}

type VMessEndpoint struct {
	tag       string
	share     string
	checkPort int
	address   string
	port      json.Number
	id        string
	alterId   json.Number
	security  string
	net       string
	fakeType  string
	fakeHost  string
	path      string
	tls       string
	sni       string
}

func (e *VMessEndpoint) Tag() string {
	return e.tag
}

func (e *VMessEndpoint) Share() string {
	return e.share
}

func (e *VMessEndpoint) CheckPort() int {
	return e.checkPort
}

func (e *VMessEndpoint) SetCheckPort(p int) {
	e.checkPort = p
}

func (e *VMessEndpoint) Outbound() *vc.Outbound {
	security := e.security
	if security == "" {
		security = "auto"
	}
	ss := &vc.StreamSettings{
		Network:  "",
		Security: "none",
		TlsSettings: &vc.TlsSettings{
			ServerName:    "server.cc",
			Alpn:          []string{"http/1.1"},
			AllowInsecure: false,
		},
		TcpSettings: &vc.TcpSettings{
			Header: &vc.Headers{Type: "none"},
		},
		KcpSettings: &vc.KcpSettings{
			Mtu:              1350,
			Tti:              20,
			UplinkCapacity:   5,
			DownlinkCapacity: 20,
			Congestion:       false,
			ReadBufferSize:   1,
			WriteBufferSize:  1,
			Header: &vc.Headers{
				Type: "none",
			},
			Seed: "",
		},
		WsSettings: &vc.WsSettings{
			Path:    "",
			Headers: map[string]any{},
		},
		HttpSettings: &vc.HttpSettings{
			Host: []string{""},
			Path: "",
		},
		QUICSettings: &vc.QUICSettings{
			Security: "none",
			Key:      "",
			Header: &vc.Headers{
				Type: "none",
			},
		},
	}
	switch e.net {
	case "tcp":
		if e.fakeType != "none" && e.fakeType != "http" {
			break
		}
		ss.TcpSettings.Header.Type = e.fakeType
		if e.fakeType != "http" {
			break
		}
		if e.fakeHost == "" {
			break
		}
		ss.TcpSettings.Header.Request = &vc.Request{
			Headers: map[string]any{
				"host": strings.Split(e.fakeHost, ","),
			},
		}
		break
	case "kcp":
		if e.fakeType != "none" &&
			e.fakeType != "srtp" &&
			e.fakeType != "utp" &&
			e.fakeType != "wechat-video" &&
			e.fakeType != "dtls" &&
			e.fakeType != "wireguard" {
			break
		}
		ss.KcpSettings.Header.Type = e.fakeType
		break
	case "ws":
		parts := strings.SplitN(e.fakeHost, ";", 2)
		if len(parts) == 2 {
			ss.WsSettings.Path = parts[0]
			ss.WsSettings.Headers["Host"] = parts[1]
		} else {
			ss.WsSettings.Path = e.path
			ss.WsSettings.Headers["Host"] = e.fakeHost
		}
		break
	case "http":
		pathHosts := strings.SplitN(e.fakeHost, ";", 2)
		if len(pathHosts) == 2 {
			ss.WsSettings.Path = pathHosts[0]
			ss.WsSettings.Headers["Host"] = strings.Split(pathHosts[1], ",")
			break
		}
		ss.HttpSettings.Path = e.path
		if e.fakeHost == "" {
			break
		}
		ss.HttpSettings.Host = strings.Split(e.fakeHost, ",")
		break
	}
	if e.tls == "tls" {
		ss.Security = "tls"
		ss.TlsSettings.ServerName = e.address
	}
	return &vc.Outbound{
		SendThrough: "0.0.0.0",
		Mux: &vc.Mux{
			Enabled:     false,
			Concurrency: 8,
		},
		Protocol: "vmess",
		Settings: &vc.OutboundCommonSettings{
			VNext: []*vc.VNext{
				{
					Address: e.address,
					Users: []*vc.User{
						{
							Id:       e.id,
							AlterId:  e.alterId,
							Security: security,
							Level:    0,
						},
					},
					Port: e.port,
				},
			},
		},
		Tag:            e.tag,
		StreamSettings: ss,
	}
}

func FromSsShareUrl(shareUrl string) (Endpoint, error) {
	encStr, tag := divideStr(shareUrl[5:], "#")
	decData, err := base64.RawURLEncoding.DecodeString(encStr)
	if err != nil {
		return nil, errors.Wrap(err, "decoding share url base64 failed")
	}
	matches := legacySsShareUrlPattern.FindStringSubmatch(string(decData))
	if len(matches) == 0 {
		return nil, errors.Errorf("invalid legacy ss share url: %s , SIP002 format is unsuppoted", shareUrl)
	}
	port, _ := strconv.ParseInt(matches[4], 10, 64)
	if tag == "" {
		tag = fmt.Sprintf("%s-%d", matches[3], port)
	}
	return &SsEndpoint{
		tag:      tag,
		share:    shareUrl,
		method:   matches[1],
		password: matches[2],
		address:  matches[3],
		port:     port,
	}, nil
}

func FromVMessShareUrl(shareUrl string) (Endpoint, error) {
	decData, err := base64.RawURLEncoding.DecodeString(shareUrl[8:])
	if err != nil {
		return nil, errors.Wrap(err, "decoding vmess share base64 failed")
	}
	type Share struct {
		Version  string      `json:"v"`
		Ps       string      `json:"ps"`
		Address  string      `json:"add"`
		Port     json.Number `json:"port"`
		Id       string      `json:"id"`
		AlterId  json.Number `json:"aid"`
		Security string      `json:"scy"`
		Net      string      `json:"net"`
		FakeType string      `json:"type"`
		FakeHost string      `json:"host"`
		Path     string      `json:"path"`
		Tls      string      `json:"tls"`
		Sni      string      `json:"sni"`
	}
	share := Share{}
	err = json.Unmarshal(decData, &share)
	if err != nil {
		return nil, errors.Wrapf(err, "decoding vmess share json failed")
	}
	tag := share.Ps
	if tag == "" {
		tag = fmt.Sprintf("%s-%d", share.Address, share.Port)
	}
	return &VMessEndpoint{
		tag:      tag,
		share:    shareUrl,
		address:  share.Address,
		port:     share.Port,
		id:       share.Id,
		alterId:  share.AlterId,
		security: share.Security,
		net:      share.Net,
		fakeType: share.FakeType,
		fakeHost: share.FakeHost,
		path:     share.Path,
		tls:      share.Tls,
		sni:      share.Sni,
	}, nil
}

func FromShareUrl(shareUrl string) (Endpoint, error) {
	parts := strings.Split(shareUrl, "://")
	switch parts[0] {
	case "ss":
		return FromSsShareUrl(shareUrl)
	case "vmess":
		return FromVMessShareUrl(shareUrl)
	default:
		return nil, errors.Errorf("unsupported share url: %s", shareUrl)
	}
}
