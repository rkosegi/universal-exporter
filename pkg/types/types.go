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

package types

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rkosegi/yaml-pipeline/pkg/pipeline"
)

// MetricOptsSpec is configuration that includes prometheus.Opts and label names
type MetricOptsSpec struct {
	// Name is fully qualified metric name
	Name string
	// Help is info about metric
	Help string
	// LabelsNames are label names that must be populated when setting metric value
	Labels []string `json:"labels" yaml:"labels"`
	// ConstLabels are fixed labels with their values that will be attached to metric
	ConstLabels map[string]string `json:"constLabels" yaml:"constLabels"`
	// Metric type. Currently only "gauge" and "counter" are supported.
	// Default value is "gauge".
	Type *string `json:"type,omitempty" yaml:"type,omitempty"`
	// Metric holds native value
	MetricRef interface{} `json:"-" yaml:"-"`
}

func (p *MetricOptsSpec) AsOpts() prometheus.Opts {
	return prometheus.Opts{
		Name:        p.Name,
		Help:        p.Help,
		ConstLabels: p.ConstLabels,
	}
}

// ScrapeTarget defines how target is being scraped
type ScrapeTarget struct {
	// Vars are target-specific variables that will be merged with global ones before they are used.
	Vars map[string]string

	// Steps are actual pipeline steps
	Steps pipeline.ChildActions `json:"steps,omitempty" yaml:"steps,omitempty"`
}

type HttpClientServiceConfig struct {
	// Request timeout
	Timeout *time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`

	// Instrumentation enables instrumentation
	Instrumentation *InstrumentationConfigFragment `json:"instrumentation,omitempty" yaml:"instrumentation,omitempty"`

	// Cache configures HTTP response cache
	Cache *CacheConfig `json:"cache" yaml:"cache"`
}

type InstrumentationConfigFragment struct {
	// Enabled specifies whether instrumentation is enabled on current configuration level.
	Enabled *bool `json:"enabled,omitempty" yaml:"enabled,omitempty"`

	// Prefix is metric name prefix for all metrics at current configuration level.
	Prefix *string `json:"prefix,omitempty" yaml:"prefix,omitempty"`
}

// CacheConfig is used to configure TTL cache for HTTP responses.
type CacheConfig struct {
	// Enabled specifies whether to enable cache or not.
	// It's enabled by default.
	Enabled *bool `json:"enabled,omitempty" yaml:"enabled,omitempty"`

	// TTL cache time-to-live
	TTL *time.Duration `json:"ttl,omitempty" yaml:"ttl,omitempty"`

	// Capacity determines max. number of items in cache
	Capacity *int `json:"capacity,omitempty" yaml:"capacity,omitempty"`

	// Instrumentation enables cache instrumentation
	Instrumentation *InstrumentationConfigFragment `json:"instrumentation,omitempty" yaml:"instrumentation,omitempty"`
}

// DefaultExportersConfig configuration of default exporters
type DefaultExportersConfig struct {
	// BuildInfo flag to enable collectors.NewBuildInfoCollector()
	BuildInfo *bool `json:"buildInfo,omitempty" yaml:"buildInfo,omitempty"`

	// Go flag to enable collectors.NewGoCollector()
	Go *bool `json:"go,omitempty" yaml:"go,omitempty"`

	// Process flag to enable collectors.NewProcessCollector()
	Process *bool `json:"process,omitempty" yaml:"process,omitempty"`

	// Version flag to enable version.NewCollector(name)
	Version *bool `json:"version,omitempty" yaml:"version,omitempty"`

	// InstrumentHttpHandler flag to wrap http handler with promhttp.InstrumentMetricHandler
	InstrumentHttpHandler *bool `json:"instrumentHttpHandler,omitempty" yaml:"instrumentHttpHandler,omitempty"`
}

type ServerConfig struct {
	// HealthEndpoint HTTP route for health check. Default value is /healthz
	HealthEndpoint *string `json:"healthEndpoint,omitempty" yaml:"healthEndpoint,omitempty"`

	// MetricsPath HTTP route for handling metrics. Default value is /metrics
	MetricsPath *string `json:"metricsPath,omitempty" yaml:"metricsPath,omitempty"`
}

type Config struct {
	// HttpClient configures HttpClientService
	HttpClient *HttpClientServiceConfig `json:"httpClient" yaml:"httpClient"`

	// Server is server configuration
	Server *ServerConfig `json:"server,omitempty" yaml:"server,omitempty"`

	// Vars is map of global variables
	Vars map[string]string `json:"vars" yaml:"vars"`

	// Metrics is map of names to MetricOptsSpec.
	// These are referred to by pipeline functions during transformation.
	Metrics map[string]*MetricOptsSpec `json:"metrics" yaml:"metrics"`

	// Whether to register built-in descriptors such as Go GC, process etc.
	DefaultExporters *DefaultExportersConfig `json:"defaultExporters,omitempty" yaml:"defaultExporters,omitempty"`

	// Targets to scrape. REQUIRED.
	Targets map[string]ScrapeTarget `json:"targets" yaml:"targets"`
}

// misc structs

type ParsedHttpResponse struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

func (r *ParsedHttpResponse) AsHttpResponse() *http.Response {
	return &http.Response{
		StatusCode: r.StatusCode,
		Header:     r.Header,
		Body:       io.NopCloser(bytes.NewReader(r.Body)),
	}
}
