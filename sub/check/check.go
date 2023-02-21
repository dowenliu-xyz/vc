package check

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"golang.org/x/exp/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
	"vc/sub"
	"vc/vc"
)

var (
	timeoutSec = 5
	testUrl    = "https://httpbin.org/get"
)

func init() {
	if s, ok := os.LookupEnv("VC_CHECK_TIMEOUT"); ok {
		i, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			slog.Info(fmt.Sprintf("Invalid timeout value: %q, use default value: %d", s, timeoutSec))
		} else {
			timeoutSec = int(i)
		}
	}
	if s, ok := os.LookupEnv("VC_CHECK_URL"); ok {
		_, err := url.Parse(s)
		if err != nil {
			slog.Info(fmt.Sprintf("Invalid check url: %q, use default value: %q", s, testUrl))
		}
		testUrl = s
	}
}

func check(ctx context.Context, ep sub.Endpoint) bool {
	tr := &http.Transport{
		Proxy: func(_ *http.Request) (*url.URL, error) {
			return url.Parse(fmt.Sprintf("socks5://127.0.0.1:%d", ep.CheckPort()))
		},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   time.Second * time.Duration(timeoutSec),
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, testUrl, nil)
	resp, err := client.Do(req)
	if err != nil {
		slog.Info(fmt.Sprintf("failed accessing %q, via ep: %s: %+v", testUrl, ep.Tag(), err))
		return false
	}
	if resp.StatusCode != http.StatusOK {
		slog.Info(fmt.Sprintf("getting %q responses %s %d, via ep: %s",
			testUrl, resp.Status, resp.StatusCode, ep.Tag()))
		return false
	}
	return true
}

func Check(ctx context.Context, eps []sub.Endpoint) []sub.Endpoint {
	ok := make([]sub.Endpoint, 0, len(eps))
	for _, ep := range eps {
		if check(ctx, ep) {
			ok = append(ok, ep)
		}
	}
	return ok
}

func Balance(cfg *vc.Config, eps []sub.Endpoint) (*vc.Config, error) {
	if len(eps) == 0 {
		return nil, errors.Errorf("cannot balance on empty endpoints")
	}
	cfg, err := vc.DeepClone(cfg)
	if err != nil {
		return nil, err
	}
	tags := make([]string, 0, len(eps))
	for _, ep := range eps {
		tags = append(tags, ep.Tag())
	}
	cfg.Routing.Balancers[0].Selector = tags
	return cfg, nil
}
