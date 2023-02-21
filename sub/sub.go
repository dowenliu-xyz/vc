package sub

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/pkg/errors"
	"golang.org/x/exp/slog"
	"io"
	"net/http"
	"strings"
	"vc/vc"
)

func fetchHttp(address string) ([]byte, error) {
	resp, err := http.Get(address)
	if err != nil {
		return nil, errors.Wrap(err, "fetching http failed")
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "reading http response failed")
	}
	return data, nil
}

func decodeEndpoints(data []byte) []Endpoint {
	var eps []Endpoint
	scanner := bufio.NewScanner(base64.NewDecoder(base64.StdEncoding, bytes.NewReader(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		ep, err := FromShareUrl(line)
		if err != nil {
			slog.Warn(fmt.Sprintf("parsing share url failed: %+v", err))
		}
		eps = append(eps, ep)
	}
	if err := scanner.Err(); err != nil {
		slog.Warn(fmt.Sprintf("decoding && reading base64 data failed: %+v", err))
	}
	return eps
}

func FetchEndpoints(address string) ([]Endpoint, error) {
	encData, err := fetchHttp(address)
	if err != nil {
		return nil, err
	}
	eps := decodeEndpoints(encData)
	if len(eps) == 0 {
		return nil, errors.Errorf("got none endpoint")
	}
	return eps, nil
}

func Override(base *vc.Config, eps []Endpoint) (*vc.Config, error) {
	cfg, err := vc.DeepClone(base)
	if err != nil {
		return nil, err
	}
	// override
	ibp := 20000
	var inbounds []*vc.Inbound
	outbounds := make([]*vc.Outbound, 0, len(eps)+3)
	var rules []*vc.Rule
	tags := make([]string, 0, len(eps))
	for _, ep := range eps {
		// append test inbound
		ibp++
		ep.SetCheckPort(ibp)
		inbound := &vc.Inbound{
			Listen:   "127.0.0.1",
			Port:     ibp,
			Protocol: "socks",
			Settings: &vc.InboundCommonSettings{
				Auth: "noauth",
				IP:   "127.0.0.1",
			},
			Tag: "test-in-" + ep.Tag(),
			Sniffing: &vc.Sniffing{
				Enabled:      true,
				DestOverride: []string{"http", "tls"},
			},
		}
		inbounds = append(inbounds, inbound)
		// append outbound
		outbounds = append(outbounds, ep.Outbound())
		// append test route rule
		rule := &vc.Rule{
			Type:        "field",
			InboundTag:  []string{"test-in-" + ep.Tag()},
			OutboundTag: ep.Tag(),
		}
		rules = append(rules, rule)
		// append balancer selector
		tags = append(tags, ep.Tag())
	}
	// append base inbounds
	for _, inbound := range cfg.Inbounds {
		if strings.HasPrefix(inbound.Tag, "test-in-") {
			continue
		}
		inbounds = append(inbounds, inbound)
	}
	// append base rules
	for _, rule := range cfg.Routing.Rules {
		if rule.OutboundTag == "direct" ||
			rule.OutboundTag == "dns" ||
			rule.OutboundTag == "decline" ||
			rule.BalancerTag != "" {
			rules = append(rules, rule)
		}
	}
	// append default outbounds
	for _, outbound := range cfg.Outbounds {
		if outbound.Protocol != "dns" &&
			outbound.Protocol != "freedom" &&
			outbound.Protocol != "blackhole" {
			continue
		}
		outbounds = append(outbounds, outbound)
	}
	cfg.Inbounds = inbounds
	cfg.Routing.Rules = rules
	cfg.Routing.Balancers[0].Selector = tags
	cfg.Outbounds = outbounds
	return cfg, nil
}
