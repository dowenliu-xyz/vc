package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/exp/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
	"vc/sub"
	"vc/sub/check"
	"vc/vc"
)

var (
	v2rayConfig = "/opt/v2ray/config.json"
	subUrl      = ""
	enableCheck = false
	v2rayAsset  = "/opt/v2ray/asset"
	v2rayBin    = "/opt/v2ray/v2ray"
	subPeriod   = time.Minute
	apiPort     = 0
)

func init() {
	if s := os.Getenv("V2RAY_CONFIG"); s != "" {
		slog.Info(fmt.Sprintf("use config path from environment: %s", s))
		v2rayConfig = s
	}
	if s := os.Getenv("VC_SUB_URL"); s != "" {
		slog.Info(fmt.Sprintf("subscription enabled"))
		subUrl = s
		if s, ok := os.LookupEnv("VC_SUB_CHECK"); ok && (s != "false" && s != "off") {
			slog.Info("connectivity check enabled")
			enableCheck = true
		}
	}
	if s := os.Getenv("VC_CHECK_PERIOD"); s != "" {
		if sec, err := strconv.ParseInt(s, 10, 64); err != nil {
			slog.Warn(fmt.Sprintf("invalid environment value: VC_CHECK_PERIOD=%s", s),
				slog.ErrorKey, err)
		} else {
			subPeriod = time.Second * time.Duration(sec)
		}
	}
	if s := os.Getenv("V2RAY_ASSET"); s != "" {
		slog.Info(fmt.Sprintf("use v2ray asset location from environment: %s", s))
		v2rayAsset = s
	}
	if s := os.Getenv("V2RAY_BIN"); s != "" {
		slog.Info(fmt.Sprintf("use v2ray bin from environment: %s", s))
		v2rayBin = s
	}
	if s := os.Getenv("VC_API_PORT"); s != "" {
		if p, err := strconv.ParseInt(s, 10, 32); err != nil {
			slog.Warn(fmt.Sprintf("invalid environment value: VC_API_PORT=%s", s))
		} else {
			apiPort = int(p)
		}
	}
}

var (
	mux        = &sync.Mutex{}
	servingCfg *vc.Config
	lastSubEps []sub.Endpoint
	lastOkEps  []sub.Endpoint
)

var epCheckCount = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "vc_sub_ep_check",
	Help: "subscription endpoint check result",
}, []string{"tag", "ok"})

func init() {
	prometheus.MustRegister(epCheckCount)
}

func main() {
	slog.Info(fmt.Sprintf("starting with config %s", v2rayConfig), "with-sub", subUrl != "", "with-check", enableCheck)
	ctx, cancel := waitSignal()
	defer cancel()
	slog.Info("loading config")
	err := readConfig()
	if err != nil {
		slog.Error("read v2ray config failed", err)
		return
	}
	filename, err := renderConfig()
	if err != nil {
		slog.Error("rendering config file failed", err)
		return
	}
	subNotify, checkNotify, restartNotify := make(chan string), make(chan string), make(chan struct{})
	if subUrl != "" {
		slog.Info("check subscription before starting core...")
		if changed, err := doSubscribe(filename); err != nil {
			slog.Warn("checking subscription failed, use base config")
		} else {
			if changed {
				slog.Info("config is modified by subscription")
			}
		}
	}
	slog.Info("starting core...")
	go func() {
		runCoreLoop(ctx, filename, restartNotify)
	}()
	if subUrl != "" {
		go func() {
			slog.Info("starting subscription check loop")
			subLoop(ctx, filename, subNotify, restartNotify)
		}()
		if enableCheck {
			go func() {
				slog.Info("starting connectivity check loop")
				checkLoop(ctx, filename, checkNotify, restartNotify)
			}()
		}
	}
	if apiPort > 0 {
		go func() {
			startApi(ctx, subNotify, checkNotify, restartNotify)
		}()
	}
	select {
	case <-ctx.Done():
		slog.Info("stopping...")
	}
}

func runCoreLoop(ctx context.Context, filename string, restart chan struct{}) {
	var cmd *exec.Cmd
	pup := (*unsafe.Pointer)(unsafe.Pointer(&cmd))
	go func() {
		for {
			select {
			case <-ctx.Done():
				break
			default:
			}
			slog.Info("run core...")
			newCmd := coreCmd(ctx, filename)
			atomic.StorePointer(pup, unsafe.Pointer(newCmd))
			if err := newCmd.Run(); err != nil {
				slog.Error("core running failed", err)
			} else {
				slog.Info("core stopped.")
			}
		}
	}()
	for {
		select {
		case <-ctx.Done():
			break
		case <-restart:
			slog.Info("stop core...")
			var (
				maxTries = 3
				tried    = 0
			)
			cmd := (*exec.Cmd)(atomic.LoadPointer(pup))
			if cmd == nil || cmd.Process == nil {
				slog.Info("no core is running, ignore.")
				continue
			}
			for tried < maxTries {
				if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
					slog.Warn("killing core with signal TERM fail", slog.ErrorKey, err)
				} else {
					slog.Info("Signal TERM has bean send to core.")
					break
				}
				tried++
			}
		}
	}
}

func startApi(ctx context.Context, subNotify chan string, checkNotify chan string, restartNotify chan struct{}) {
	http.HandleFunc("/api/sub", func(w http.ResponseWriter, r *http.Request) {
		subNotify <- fmt.Sprintf("An API request recieved, ")
		w.WriteHeader(http.StatusAccepted)
	})
	http.HandleFunc("/api/sub/check", func(w http.ResponseWriter, r *http.Request) {
		checkNotify <- fmt.Sprintf("An API request recieved, ")
		w.WriteHeader(http.StatusAccepted)
	})
	http.HandleFunc("/api/core/restart", func(w http.ResponseWriter, r *http.Request) {
		restartNotify <- struct{}{}
		w.WriteHeader(http.StatusAccepted)
	})
	http.Handle("/metrics", promhttp.Handler())
	server := http.Server{
		Addr: fmt.Sprintf(":%d", apiPort),
	}
	go func() {
		<-ctx.Done()
		_ = server.Shutdown(context.Background())
	}()
	if err := server.ListenAndServe(); err != nil && !errors.As(err, http.ErrServerClosed) {
		slog.Error("serving api failed", err)
	}
}

func readConfig() error {
	data, err := os.ReadFile(v2rayConfig)
	if err != nil {
		return err
	}
	cfg := &vc.Config{}
	err = json.Unmarshal(data, cfg)
	if err != nil {
		return err
	}
	servingCfg = cfg
	return nil
}

func renderConfig() (string, error) {
	data, err := json.Marshal(servingCfg)
	if err != nil {
		return "", errors.Wrap(err, "marshalling config failed")
	}
	tmpDir, err := os.MkdirTemp("", "v2ray-*")
	if err != nil {
		return "", errors.Wrap(err, "creating tmp dir failed")
	}
	filename := filepath.Join(tmpDir, "config.json")
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return "", errors.Wrap(err, "writing config file failed")
	}
	return filename, nil
}

func checkLoop(ctx context.Context, filename string, notify chan string, restartNotify chan<- struct{}) {
	go func() {
		for {
			reason, ok := <-notify
			if !ok {
				break
			}
			slog.Info(fmt.Sprintf("%scheck connectivity...", reason))
			if changed := doCheck(ctx, filename); !changed {
				continue
			}
			slog.Info("balancer endpoints updated, restart core...")
			restartNotify <- struct{}{}
		}
	}()
	for {
		select {
		case <-ctx.Done():
			slog.Info("stop endpoint checking loop")
			close(notify)
			return
		case <-time.After(subPeriod):
			notify <- fmt.Sprintf("%f seconds passed", subPeriod.Seconds())
		}
	}
}

func doCheck(ctx context.Context, filename string) bool {
	mux.Lock()
	defer mux.Unlock()
	if len(lastSubEps) == 0 {
		return false
	}
	newEps := check.Check(ctx, lastSubEps)
	okMap := make(map[string]bool, len(lastSubEps))
	for _, ep := range lastSubEps {
		okMap[ep.Tag()] = false
	}
	lastOk, newOk := make([]string, len(lastOkEps)), make([]string, len(newEps))
	for i, ep := range lastOkEps {
		lastOk[i] = ep.Share()
	}
	for i, ep := range newEps {
		newOk[i] = ep.Share()
		okMap[ep.Tag()] = true
	}
	for tag, ok := range okMap {
		epCheckCount.WithLabelValues(tag, strconv.FormatBool(ok)).Inc()
	}
	if reflect.DeepEqual(newOk, lastOk) {
		return false
	}
	newCfg, err := check.Balance(servingCfg, newEps)
	if err != nil {
		slog.Warn("balancing failed", slog.ErrorKey, err)
		return false
	}
	data, err := json.Marshal(newCfg)
	if err != nil {
		slog.Warn("marshalling new config failed", slog.ErrorKey, err)
		return false
	}
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		slog.Warn("writing new config content failed")
		return false
	}
	servingCfg = newCfg
	lastOkEps = newEps
	return true
}

func subLoop(ctx context.Context, filename string, notify chan string, restartNotify chan<- struct{}) {
	go func() {
		for {
			reason, ok := <-notify
			if !ok {
				break
			}
			slog.Info(fmt.Sprintf("%scheck subscription...", reason))
			changed, err := doSubscribe(filename)
			if err != nil {
				slog.Warn("checking subscription failed, keep using previous config", slog.ErrorKey, err)
				continue
			}
			if !changed {
				continue
			}
			slog.Info("config is modified by subscription, restart core...")
			restartNotify <- struct{}{}
		}
	}()
	for {
		select {
		case <-ctx.Done():
			slog.Info("stop subscription checking loop")
			close(notify)
			return
		case <-time.After(subPeriod):
			notify <- fmt.Sprintf("%f seconds passed, ", subPeriod.Seconds())
		}
	}
}

func doSubscribe(filename string) (bool, error) {
	if subUrl == "" {
		return false, nil
	}
	newEps, err := sub.FetchEndpoints(subUrl)
	if err != nil {
		return false, err
	}
	mux.Lock()
	defer mux.Unlock()
	newShares, lastShares := make([]string, len(newEps)), make([]string, len(lastSubEps))
	for i, ep := range newEps {
		newShares[i] = ep.Share()
	}
	for i, ep := range lastSubEps {
		lastShares[i] = ep.Share()
	}
	if reflect.DeepEqual(newShares, lastShares) {
		return false, nil
	}
	newCfg, err := sub.Override(servingCfg, newEps)
	if err != nil {
		return false, err
	}
	data, err := json.Marshal(newCfg)
	if err != nil {
		return false, errors.Wrap(err, "marshalling new config failed")
	}
	if err := os.WriteFile(v2rayConfig, data, 0644); err != nil {
		slog.Warn("update source config via subscription failed", slog.ErrorKey, err)
	}
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return false, errors.Wrap(err, "writing new config content failed")
	}
	servingCfg = newCfg
	lastSubEps = newEps
	lastOkEps = newEps
	return true, nil
}

func coreCmd(ctx context.Context, filename string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, v2rayBin, "-config", filename)
	cmd.Env = append(cmd.Env, fmt.Sprintf("V2RAY_LOCATION_ASSET=%s", v2rayAsset))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

func waitSignal() (context.Context, context.CancelFunc) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-ch
		cancel()
	}()
	return ctx, cancel
}
