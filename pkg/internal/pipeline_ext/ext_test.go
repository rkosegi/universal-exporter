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

package pipeline_ext

import (
	"testing"

	"github.com/rkosegi/yaml-toolkit/pipeline"
	"github.com/stretchr/testify/assert"
)

func TestApplyArgs(t *testing.T) {
	op := &httpFetchOp{}
	ac := pipeline.New().(pipeline.ActionContextFactory).NewActionContext(op)

	TryApplyArgs(ac, op, map[string]interface{}{
		"url": "http://example.com",
	})
	assert.Equal(t, "http://example.com", op.Url)
}
