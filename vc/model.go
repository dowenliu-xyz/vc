package vc

import (
	"encoding/json"
	"fmt"
)

type Config struct {
	Log       *Log        `json:"log,omitempty"`
	Dns       *Dns        `json:"dns,omitempty"`
	Routing   *Routing    `json:"routing,omitempty"`
	Inbounds  []*Inbound  `json:"inbounds,omitempty"`
	Outbounds []*Outbound `json:"outbounds,omitempty"`
}

type Log struct {
	Access   string `json:"access,omitempty"`
	Error    string `json:"error,omitempty"`
	LogLevel string `json:"loglevel,omitempty"`
}

type Dns struct {
	Hosts   map[string]any `json:"hosts,omitempty"`
	Servers []*Server      `json:"servers,omitempty"`
}

type Server struct {
	string
	*ComplexServer
}

func (s *Server) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("null"), nil
	}
	if s.ComplexServer == nil {
		return json.Marshal(s.string)
	}
	return json.Marshal(s.ComplexServer)
}

func (s *Server) UnmarshalJSON(data []byte) error {
	if s == nil {
		return fmt.Errorf("cannot unmarshal to a nil *Server")
	}
	s.ComplexServer = &ComplexServer{}
	if err := json.Unmarshal(data, s.ComplexServer); err == nil {
		s.string = ""
		return nil
	}
	err := json.Unmarshal(data, &s.string)
	if err != nil {
		return fmt.Errorf("cannot unmarshal to *Server")
	}
	s.ComplexServer = nil
	return nil
}

type ComplexServer struct {
	Address      string   `json:"address,omitempty"`
	Port         int64    `json:"port,omitempty"`
	ClientIp     string   `json:"clientIp,omitempty"`
	SkipFallback bool     `json:"skipFallback,omitempty"`
	Domains      []string `json:"domains,omitempty"`
	ExpectIPs    []string `json:"expectIPs,omitempty"`
}

type Routing struct {
	DomainStrategy string      `json:"domainStrategy,omitempty"`
	DomainMatcher  string      `json:"domainMatcher,omitempty"`
	Rules          []*Rule     `json:"rules,omitempty"`
	Balancers      []*Balancer `json:"balancers,omitempty"`
}

type Rule struct {
	DomainMatcher string   `json:"domainMatcher,omitempty"`
	Type          string   `json:"type,omitempty"`
	Domains       []string `json:"domains,omitempty"`
	IP            []string `json:"ip,omitempty"`
	Port          any      `json:"port,omitempty"`
	SourcePort    any      `json:"sourcePort,omitempty"`
	Network       string   `json:"network,omitempty"`
	Source        []string `json:"source,omitempty"`
	User          []string `json:"user,omitempty"`
	InboundTag    []string `json:"inboundTag,omitempty"`
	Protocol      []string `json:"protocol,omitempty"`
	Attrs         string   `json:"attrs,omitempty"`
	OutboundTag   string   `json:"outboundTag,omitempty"`
	BalancerTag   string   `json:"balancerTag,omitempty"`
}

type Balancer struct {
	Tag      string    `json:"tag,omitempty"`
	Selector []string  `json:"selector,omitempty"`
	Strategy *Strategy `json:"strategy,omitempty"`
}

type Strategy struct {
	Type string `json:"type,omitempty"`
}

type Inbound struct {
	Listen         string                 `json:"listen,omitempty"`
	Port           any                    `json:"port,omitempty"`
	Protocol       string                 `json:"protocol,omitempty"`
	Settings       *InboundCommonSettings `json:"settings,omitempty"`
	StreamSettings any                    `json:"streamSettings,omitempty"`
	Tag            string                 `json:"tag,omitempty"`
	Sniffing       *Sniffing              `json:"sniffing,omitempty"`
	Allocate       any                    `json:"allocate,omitempty"`
}

type InboundCommonSettings struct {
	Timeout          json.Number `json:"timeout,omitempty"`
	Auth             string      `json:"auth,omitempty"`
	Accounts         []*Account  `json:"accounts,omitempty"`
	AllowTransparent bool        `json:"allowTransparent,omitempty"`
	Udp              bool        `json:"udp,omitempty"`
	IP               string      `json:"ip,omitempty"`
	UserLevel        json.Number `json:"userLevel,omitempty"`
}

type Account struct {
	User string `json:"user,omitempty"`
	Pass string `json:"pass,omitempty"`
}

type Sniffing struct {
	Enabled      bool     `json:"enabled,omitempty"`
	DestOverride []string `json:"destOverride,omitempty"`
	MetadataOnly bool     `json:"metadataOnly,omitempty"`
}

type Outbound struct {
	SendThrough    string                  `json:"sendThrough,omitempty"`
	Protocol       string                  `json:"protocol,omitempty"`
	Settings       *OutboundCommonSettings `json:"settings,omitempty"`
	Tag            string                  `json:"tag,omitempty"`
	StreamSettings *StreamSettings         `json:"streamSettings,omitempty"`
	ProxySettings  *ProxySettings          `json:"proxySettings,omitempty"`
	Mux            *Mux                    `json:"mux,omitempty"`
}

type OutboundCommonSettings struct {
	Servers []*SsServer `json:"servers,omitempty"`
	VNext   []*VNext    `json:"vnext,omitempty"`
}

type SsServer struct {
	Email    string `json:"email,omitempty"`
	Address  string `json:"address,omitempty"`
	Port     int64  `json:"port,omitempty"`
	Method   string `json:"method,omitempty"`
	Password string `json:"password,omitempty"`
	Level    int64  `json:"level,omitempty"`
	IVCheck  bool   `json:"ivCheck,omitempty"`
}

type VNext struct {
	Address string      `json:"address,omitempty"`
	Port    json.Number `json:"port,omitempty"`
	Users   []*User     `json:"users,omitempty"`
}

type User struct {
	Id       string      `json:"id,omitempty"`
	AlterId  json.Number `json:"alterId"`
	Security string      `json:"security,omitempty"`
	Level    int64       `json:"level"`
}

type StreamSettings struct {
	Network      string        `json:"network,omitempty"`
	Security     string        `json:"security,omitempty"`
	TlsSettings  *TlsSettings  `json:"tlsSettings,omitempty"`
	TcpSettings  *TcpSettings  `json:"tcpSettings,omitempty"`
	KcpSettings  *KcpSettings  `json:"kcpSettings,omitempty"`
	WsSettings   *WsSettings   `json:"wsSettings,omitempty"`
	HttpSettings *HttpSettings `json:"httpSettings,omitempty"`
	QUICSettings *QUICSettings `json:"quicSettings,omitempty"`
}

type TlsSettings struct {
	ServerName                       string         `json:"serverName,omitempty"`
	Alpn                             []string       `json:"alpn,omitempty"`
	AllowInsecure                    bool           `json:"allowInsecure"`
	DisableSystemRoot                bool           `json:"disableSystemRoot,omitempty"`
	Certificates                     []*Certificate `json:"certificates,omitempty"`
	VerifyClientCertificate          bool           `json:"verifyClientCertificate,omitempty"`
	PinnedPeerCertificateChainSha256 string         `json:"pinnedPeerCertificateChainSha256,omitempty"`
}

type Certificate struct {
	Usage           string   `json:"usage,omitempty"`
	CertificateFile string   `json:"certificateFile,omitempty"`
	KeyFile         string   `json:"keyFile,omitempty"`
	Certificate     []string `json:"certificate,omitempty"`
	Key             []string `json:"key,omitempty"`
}

type TcpSettings struct {
	AcceptProxyProtocol bool     `json:"acceptProxyProtocol,omitempty"`
	Header              *Headers `json:"header,omitempty"`
}

type Headers struct {
	Type     string    `json:"type,omitempty"`
	Request  *Request  `json:"request,omitempty"`
	Response *Response `json:"response,omitempty"`
}

type Request struct {
	Version string         `json:"version,omitempty"`
	Method  string         `json:"method,omitempty"`
	Path    []string       `json:"path,omitempty"`
	Headers map[string]any `json:"headers,omitempty"`
}

type Response struct {
	Version string         `json:"version,omitempty"`
	Status  string         `json:"status,omitempty"`
	Reason  string         `json:"reason,omitempty"`
	Headers map[string]any `json:"headers,omitempty"`
}

type KcpSettings struct {
	Mtu              int      `json:"mtu,omitempty"`
	Tti              int      `json:"tti,omitempty"`
	UplinkCapacity   int      `json:"uplinkCapacity,omitempty"`
	DownlinkCapacity int      `json:"downlinkCapacity,omitempty"`
	Congestion       bool     `json:"congestion"`
	ReadBufferSize   int      `json:"readBufferSize,omitempty"`
	WriteBufferSize  int      `json:"writeBufferSize,omitempty"`
	Header           *Headers `json:"header,omitempty"`
	Seed             string   `json:"seed,omitempty"`
}

type WsSettings struct {
	AcceptProxyProtocol  bool           `json:"acceptProxyProtocol,omitempty"`
	Path                 string         `json:"path"`
	Headers              map[string]any `json:"headers"`
	MaxEarlyData         int            `json:"maxEarlyData,omitempty"`
	UseBrowserForwarding bool           `json:"useBrowserForwarding,omitempty"`
	EarlyDataHeaderName  string         `json:"earlyDataHeaderName,omitempty"`
}

type HttpSettings struct {
	Host    []string       `json:"host,omitempty"`
	Path    string         `json:"path"`
	Method  string         `json:"method,omitempty"`
	Headers map[string]any `json:"headers,omitempty"`
}

type QUICSettings struct {
	Security string   `json:"security,omitempty"`
	Key      string   `json:"key"`
	Header   *Headers `json:"header,omitempty"`
}

type ProxySettings struct {
	Tag            string `json:"tag,omitempty"`
	TransportLayer bool   `json:"transportLayer,omitempty"`
}

type Mux struct {
	Enabled     bool  `json:"enabled"`
	Concurrency int64 `json:"concurrency,omitempty"`
}
