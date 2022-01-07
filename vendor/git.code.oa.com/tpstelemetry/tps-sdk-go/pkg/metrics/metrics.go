// Copyright 2020 The TpsTelemetry Authors
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

// Package metrics metrics 属性
package metrics

import "github.com/prometheus/client_golang/prometheus"

func init() {
	prometheus.MustRegister(BatchProcessCounter)
	prometheus.MustRegister(DeferredProcessCounter)
}

var (
	// BatchProcessCounter batch processor counter
	BatchProcessCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: "tpstelemetry_sdk",
			Name:      "batch_process_counter",
			Help:      "Batch Process Counter",
		},
		[]string{"status", "telemetry"},
	)
	// DeferredProcessCounter defered processor counter
	DeferredProcessCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: "tpstelemetry_sdk",
			Name:      "defered_process_counter",
			Help:      "defered Process Counter",
		},
		[]string{"status", "telemetry"},
	)
)
