package pipeline_ext

import (
	"errors"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rkosegi/universal-exporter/pkg/types"
	"github.com/rkosegi/yaml-toolkit/pipeline"
)

type (
	promCounterVecOp struct {
		Ref         string   `yaml:"ref"`
		IncBy       *float64 `yaml:"incBy,omitempty"`
		LabelValues []string `yaml:"labelValues"`
	}
	promGaugeVecOp struct {
		DescRef string  `yaml:"descRef"`
		SetTo   float64 `yaml:"setTo"`
	}
)

func (p *promCounterVecOp) String() string {
	return fmt.Sprintf("PromCounterOp[ref=%s]", p.Ref)
}

func (p *promCounterVecOp) Do(ctx pipeline.ActionContext) error {
	incBy := float64(1)
	if len(p.Ref) == 0 {
		return errors.New("empty metric reference")
	}

	if p.IncBy != nil {
		incBy = *p.IncBy
	}

	var (
		svc       interface{}
		ms        types.NamedLookup[types.MetricOptsSpec]
		ok        bool
		metricOps types.MetricOptsSpec
	)

	if svc = ctx.Ext().GetService("metricsMap"); svc != nil {
		return errors.New("no metrics map found in ActionContext")
	}

	if ms, ok = svc.(types.NamedLookup[types.MetricOptsSpec]); !ok {
		return errors.New("service with name 'metricMap' does not implement NamedLookup")
	}

	if metricOps, ok = ms.Get(p.Ref); !ok {
		return fmt.Errorf("no MetricOptsSpec with name '%s' found in metricMap", p.Ref)
	}

	cv := prometheus.NewCounterVec(prometheus.CounterOpts(metricOps.AsOpts()), metricOps.LabelsNames)

	cv.WithLabelValues().Add(incBy)
	return nil
}

func (p *promCounterVecOp) CloneWith(ctx pipeline.ActionContext) pipeline.Action {
	return &promCounterVecOp{}
}
