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
	"fmt"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rkosegi/universal-exporter/pkg/types"
	"github.com/samber/lo"
)

func NewMetricService(mos map[string]*types.MetricOptsSpec, l *slog.Logger) types.MetricService {
	return &promMetricService{
		mos: mos,
		l:   l,
	}
}

type promMetricService struct {
	*noopService
	mos map[string]*types.MetricOptsSpec
	l   *slog.Logger
}

func (ms *promMetricService) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range ms.mos {
		switch *m.Type {
		case "counter":
			m.MetricRef.(*prometheus.CounterVec).Describe(ch)
		case "gauge":
			m.MetricRef.(*prometheus.GaugeVec).Describe(ch)
		}
	}
}

func (ms *promMetricService) Collect(ch chan<- prometheus.Metric) {
	for _, m := range ms.mos {
		switch *m.Type {
		case "counter":
			m.MetricRef.(*prometheus.CounterVec).Collect(ch)
		case "gauge":
			m.MetricRef.(*prometheus.GaugeVec).Collect(ch)
		}
	}
}

func (ms *promMetricService) Start() error {
	for name, opt := range ms.mos {
		opt.Name = name
		if opt.Type == nil {
			opt.Type = lo.ToPtr("gauge")
		}
		ms.l.Info("Registering metric", "metric_opt", *opt)
		switch *opt.Type {
		case "gauge":
			opt.MetricRef = prometheus.NewGaugeVec(prometheus.GaugeOpts(opt.AsOpts()), opt.Labels)
		case "counter":
			opt.MetricRef = prometheus.NewCounterVec(prometheus.CounterOpts(opt.AsOpts()), opt.Labels)
		default:
			return fmt.Errorf("unsupported metric type: %s", *opt.Type)
		}
	}
	return nil
}

func (ms *promMetricService) GetRef(name string) (*types.MetricOptsSpec, error) {
	if opt, ok := ms.mos[name]; !ok {
		return nil, fmt.Errorf("no such metric: '%s'", name)
	} else {
		return opt, nil
	}
}
