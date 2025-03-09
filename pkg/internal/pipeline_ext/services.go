package pipeline_ext

import (
	"net/http"
	"net/url"

	"github.com/jellydator/ttlcache/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rkosegi/universal-exporter/pkg/types"
	"github.com/rkosegi/yaml-toolkit/fluent"
	. "github.com/rkosegi/yaml-toolkit/pipeline"
	"github.com/samber/lo"
)

var (
	defCfg = &types.HttpClientServiceConfig{
		Cache: &types.CacheConfig{
			TTL:      lo.ToPtr(types.DefaultCacheTTL),
			Capacity: lo.ToPtr(types.DefaultCacheCapacity),
			Instrumentation: &types.CacheInstrumentationConfig{
				Enabled: lo.ToPtr(true),
				Prefix:  lo.ToPtr(types.DefaultMetricPrefixHttpCache),
			},
		},
	}
)

func key2host(key string) string {
	if u, err := url.Parse(key); err == nil {
		return u.Host
	}
	return key
}

type hcServiceImpl struct {
	Cfg         types.HttpClientServiceConfig `yaml:"config"`
	started     bool
	ctx         ServiceContext
	cache       *ttlcache.Cache[string, *types.ParsedHttpResponse]
	reg         prometheus.Registerer
	hitCounter  *prometheus.CounterVec
	missCounter *prometheus.CounterVec
}

func (h *hcServiceImpl) Config() types.HttpClientServiceConfig {
	// TODO implement me
	panic("implement me")
}

func (h *hcServiceImpl) Client() http.Client {
	// TODO implement me
	panic("implement me")
}

func (h *hcServiceImpl) Doer() types.HTTPDoer {
	// TODO implement me
	panic("implement me")
}

func (h *hcServiceImpl) Describe(chan<- *prometheus.Desc) {}
func (h *hcServiceImpl) Collect(ch chan<- prometheus.Metric) {
	if *h.Cfg.Cache.Instrumentation.Enabled {
		h.missCounter.Collect(ch)
		h.hitCounter.Collect(ch)
	}
}

func (h *hcServiceImpl) Configure(ctx ServiceContext, cfg StrKeysAnyValues) Service {
	ApplyArgs(ctx, &h.Cfg, cfg)
	h.ctx = ctx
	return h
}

func (h *hcServiceImpl) Init() (err error) {
	h.ctx.Logger().Log("starting cache service")
	h.Cfg = *fluent.NewConfigHelper[types.HttpClientServiceConfig]().Add(*defCfg).Add(h.Config).Result()
	h.cache = ttlcache.New[string, *types.ParsedHttpResponse](
		ttlcache.WithDisableTouchOnHit[string, *types.ParsedHttpResponse](),
		ttlcache.WithTTL[string, *types.ParsedHttpResponse](*h.Cfg.Cache.TTL),
		ttlcache.WithCapacity[string, *types.ParsedHttpResponse](uint64(*h.Cfg.Cache.Capacity)),
	)
	go h.cache.Start()

	if *h.Cfg.Cache.Instrumentation.Enabled {
		h.hitCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: *h.Cfg.Cache.Instrumentation.Prefix + "_hit",
			Help: "HTTP response cache hit count",
		}, []string{"host"})
		h.missCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: *h.Cfg.Cache.Instrumentation.Prefix + "_miss",
			Help: "HTTP response cache miss count",
		}, []string{"host"})

		if err = h.reg.Register(h); err != nil {
			return err
		}
	}

	h.started = true
	return nil
}

func (h *hcServiceImpl) Close() error {
	if h.cache != nil {
		h.ctx.Logger().Log("stopping cache service")
		h.cache.Stop()
		h.cache = nil
	}
	h.ctx = nil
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

func (h *hcServiceImpl) Get(key string) *types.ParsedHttpResponse {
	i := h.cache.Get(key)
	if i == nil {
		h.onCacheMiss(key)
		return nil
	}
	h.onCacheHit(key)
	return i.Value()
}

func (h *hcServiceImpl) Put(key string, value *types.ParsedHttpResponse) {
	h.cache.Set(key, value, ttlcache.DefaultTTL)
}
