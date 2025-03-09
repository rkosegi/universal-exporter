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

package cache

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/rkosegi/universal-exporter/pkg/types"
	"github.com/stretchr/testify/assert"
)

type chCol struct {
	ch chan prometheus.Metric
}

func (c *chCol) get() *dto.Metric {
	dm := &dto.Metric{}

	select {
	case x, ok := <-c.ch:
		if ok {
			close(c.ch)
			_ = x.Write(dm)
		}
	default:
		break
	}

	return dm
}

func initChCol() *chCol {
	return &chCol{make(chan prometheus.Metric, 1)}
}

func TestCache(t *testing.T) {
	c := New(&types.CacheConfig{})
	r := c.Get("test")
	assert.Nil(t, r)
	c.Put("test", &types.ParsedHttpResponse{})
	r = c.Get("test")
	assert.NotNil(t, r)

	cc := initChCol()

	cnt := cc.get().Counter
	assert.Nil(t, cnt)
	c.(*impl).missCounter.Collect(cc.ch)
	cnt = cc.get().Counter
	assert.NotNil(t, cnt)
	assert.Equal(t, float64(1), *cnt.Value)
}
