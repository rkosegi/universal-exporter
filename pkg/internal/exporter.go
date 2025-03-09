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

package internal

import (
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rkosegi/universal-exporter/pkg/types"
)

type pipelineCollector struct {
	gc *types.Config
	l  *slog.Logger
}

func (p *pipelineCollector) Describe(chan<- *prometheus.Desc) {}

func (p *pipelineCollector) Collect(ch chan<- prometheus.Metric) {
	for k, _ := range p.gc.Targets {
		p.l.Info("Processing target", "name", k)
		println(k)

	}
}

func NewExporter(cfg *types.Config, logger *slog.Logger) prometheus.Collector {
	return &pipelineCollector{gc: cfg, l: logger}
}
