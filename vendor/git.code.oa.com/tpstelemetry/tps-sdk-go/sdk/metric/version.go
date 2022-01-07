// Copyright 2021 The TpsTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package metric metric 子系统
package metric

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"

	tpstelemetry "git.code.oa.com/tpstelemetry/tps-sdk-go"
)

// sdkMetadata sdk版本信息
var sdkMetadata = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "tpstelemetry_sdk_metadata",
	Help: "tpstelemetry sdk metadata version",
}, []string{"tps_version", "otel_version"})

func init() {
	sdkMetadata.WithLabelValues(tpstelemetry.Version(), otel.Version()).Set(1)
}
