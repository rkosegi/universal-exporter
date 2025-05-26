/*
Copyright 2025 Richard Kosegi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package services

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/jellydator/ttlcache/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rkosegi/universal-exporter/pkg/types"
	"github.com/rkosegi/yaml-toolkit/fluent"
)

var (
	defCfg = &types.HttpClientServiceConfig{}
)

func key2host(key string) string {
	if u, err := url.Parse(key); err == nil {
		return u.Host
	}
	return key
}

type hcServiceImpl struct {
	*noopService
	cfg         types.HttpClientServiceConfig `yaml:"config"`
	started     bool
	l           *slog.Logger
	cache       *ttlcache.Cache[string, *types.ParsedHttpResponse]
	reg         prometheus.Registerer
	hitCounter  *prometheus.CounterVec
	missCounter *prometheus.CounterVec
	counter     *prometheus.CounterVec
	histVec     *prometheus.HistogramVec
	hc          http.Client
}

func (h *hcServiceImpl) RoundTrip(req *http.Request) (*http.Response, error) {
	var (
		resp       *http.Response
		cachedResp *types.ParsedHttpResponse
		body       bytes.Buffer
		err        error
	)

	if *h.cfg.Cache.Enabled {
		if cachedResp = h.get(req.URL.String()); cachedResp != nil {
			h.l.Debug("using cached response", "url", req.URL.String())
			return cachedResp.AsHttpResponse(), nil
		}
	}

	c, cancel := context.WithCancel(context.TODO())
	timer := time.AfterFunc(*h.cfg.Timeout, func() {
		cancel()
	})
	defer timer.Stop()

	req = req.WithContext(c)
	resp, err = h.hc.Do(req)
	if err != nil {
		return nil, err
	}

	if *h.cfg.Cache.Enabled {
		defer func(Body io.ReadCloser) {
			if err = Body.Close(); err != nil {
				h.l.Warn("unable to close response body", "err", err.Error())
			}
		}(resp.Body)
		_, err = io.Copy(&body, resp.Body)
		if err != nil {
			return nil, err
		}
		cachedResp = &types.ParsedHttpResponse{
			StatusCode: resp.StatusCode,
			Header:     resp.Header,
			Body:       body.Bytes(),
		}

		h.cache.Set(req.URL.String(), cachedResp, ttlcache.DefaultTTL)
		return cachedResp.AsHttpResponse(), nil
	}

	return resp, nil
}

func (h *hcServiceImpl) RoundTripper() http.RoundTripper {
	return h
}

func (h *hcServiceImpl) Describe(ch chan<- *prometheus.Desc) {
	if *h.cfg.Cache.Enabled && *h.cfg.Cache.Instrumentation.Enabled {
		h.missCounter.Describe(ch)
		h.hitCounter.Describe(ch)
	}
	if *h.cfg.Instrumentation.Enabled {
		h.counter.Describe(ch)
		h.hitCounter.Describe(ch)
	}
}

func (h *hcServiceImpl) Collect(ch chan<- prometheus.Metric) {
	if *h.cfg.Cache.Enabled && *h.cfg.Cache.Instrumentation.Enabled {
		h.missCounter.Collect(ch)
		h.hitCounter.Collect(ch)
	}
	if *h.cfg.Instrumentation.Enabled {
		h.counter.Collect(ch)
		h.hitCounter.Collect(ch)
	}
}

func (h *hcServiceImpl) Start() (err error) {
	h.l.Info("starting cache service")
	h.cfg = *fluent.NewConfigHelper[types.HttpClientServiceConfig]().Add(*defCfg).Add(h.cfg).Result()
	h.cache = ttlcache.New[string, *types.ParsedHttpResponse](
		ttlcache.WithDisableTouchOnHit[string, *types.ParsedHttpResponse](),
		ttlcache.WithTTL[string, *types.ParsedHttpResponse](*h.cfg.Cache.TTL),
		ttlcache.WithCapacity[string, *types.ParsedHttpResponse](uint64(*h.cfg.Cache.Capacity)),
	)
	go h.cache.Start()

	if *h.cfg.Cache.Instrumentation.Enabled {
		h.hitCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: *h.cfg.Cache.Instrumentation.Prefix + "_hit",
			Help: "HTTP response cache hit count",
		}, []string{"host"})
		h.missCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: *h.cfg.Cache.Instrumentation.Prefix + "_miss",
			Help: "HTTP response cache miss count",
		}, []string{"host"})
	}

	hc := http.Client{}
	if h.cfg.Timeout != nil {
		hc.Timeout = *h.cfg.Timeout
	}

	if *h.cfg.Instrumentation.Enabled {
		h.counter = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: *h.cfg.Instrumentation.Prefix,
				Name:      "http_client_requests_total",
				Help:      "A counter for requests from the wrapped client.",
			},
			[]string{"code", "method"},
		)

		h.histVec = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: *h.cfg.Instrumentation.Prefix,
				Name:      "http_client_request_duration_seconds",
				Help:      "A histogram of request latencies.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{},
		)

		hc.Transport = promhttp.InstrumentRoundTripperCounter(h.counter,
			promhttp.InstrumentRoundTripperDuration(h.histVec, http.DefaultTransport),
		)
	}

	h.hc = hc
	h.started = true
	return nil
}

func (h *hcServiceImpl) Close() error {
	if h.cache != nil {
		h.l.Info("stopping cache service")
		h.cache.Stop()
		h.cache = nil
	}
	h.started = false
	return nil
}

func (h *hcServiceImpl) onCacheMiss(key string) {
	if h.missCounter != nil {
		h.missCounter.WithLabelValues(key2host(key)).Inc()
	}
}

func (h *hcServiceImpl) onCacheHit(key string) {
	if h.hitCounter != nil {
		h.hitCounter.WithLabelValues(key2host(key)).Inc()
	}
}

func (h *hcServiceImpl) get(key string) *types.ParsedHttpResponse {
	i := h.cache.Get(key)
	if i == nil {
		h.onCacheMiss(key)
		return nil
	}
	h.onCacheHit(key)
	return i.Value()
}

func NewHttpClient(cfg types.HttpClientServiceConfig, l *slog.Logger, reg prometheus.Registerer) types.HttpClientService {
	return &hcServiceImpl{
		cfg: cfg,
		l:   l,
		reg: reg,
	}
}
