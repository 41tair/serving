/*
Copyright 2018 The Knative Authors.

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

package config

import (
	"context"
	"math/rand"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	pkglogging "knative.dev/pkg/logging"
	logtesting "knative.dev/pkg/logging/testing"
	pkgmetrics "knative.dev/pkg/metrics"
	"knative.dev/serving/pkg/autoscaler"
	deployment "knative.dev/serving/pkg/deployment"
	"knative.dev/serving/pkg/logging"
	"knative.dev/serving/pkg/metrics"
	"knative.dev/serving/pkg/network"
	pkgtracing "knative.dev/serving/pkg/tracing/config"

	. "knative.dev/pkg/configmap/testing"
)

func TestStoreLoadWithContext(t *testing.T) {
	defer logtesting.ClearAll()
	store := NewStore(logtesting.TestLogger(t))

	deploymentConfig := ConfigMapFromTestFile(t, deployment.ConfigName, deployment.QueueSidecarImageKey)
	networkConfig := ConfigMapFromTestFile(t, network.ConfigName)
	observabilityConfig := ConfigMapFromTestFile(t, pkgmetrics.ConfigMapName())
	loggingConfig := ConfigMapFromTestFile(t, pkglogging.ConfigMapName())
	tracingConfig := ConfigMapFromTestFile(t, pkgtracing.ConfigName)
	autoscalerConfig := ConfigMapFromTestFile(t, autoscaler.ConfigName)

	store.OnConfigChanged(deploymentConfig)
	store.OnConfigChanged(networkConfig)
	store.OnConfigChanged(observabilityConfig)
	store.OnConfigChanged(loggingConfig)
	store.OnConfigChanged(tracingConfig)
	store.OnConfigChanged(autoscalerConfig)

	config := FromContext(store.ToContext(context.Background()))

	t.Run("Deployment", func(t *testing.T) {
		expected, _ := deployment.NewConfigFromConfigMap(deploymentConfig)
		if diff := cmp.Diff(expected, config.Deployment); diff != "" {
			t.Errorf("Unexpected deployment (-want, +got): %v", diff)
		}
	})

	t.Run("network", func(t *testing.T) {
		expected, _ := network.NewConfigFromConfigMap(networkConfig)
		ignoreDT := cmpopts.IgnoreFields(network.Config{}, "DomainTemplate")

		if diff := cmp.Diff(expected, config.Network, ignoreDT); diff != "" {
			t.Errorf("Unexpected controller config (-want, +got): %v", diff)
		}
	})

	t.Run("observability", func(t *testing.T) {
		expected, _ := metrics.NewObservabilityConfigFromConfigMap(observabilityConfig)
		if diff := cmp.Diff(expected, config.Observability); diff != "" {
			t.Errorf("Unexpected observability config (-want, +got): %v", diff)
		}
	})

	t.Run("logging", func(t *testing.T) {
		expected, _ := logging.NewConfigFromConfigMap(loggingConfig)
		if diff := cmp.Diff(expected, config.Logging); diff != "" {
			t.Errorf("Unexpected logging config (-want, +got): %v", diff)
		}
	})

	t.Run("tracing", func(t *testing.T) {
		expected, _ := pkgtracing.NewTracingConfigFromConfigMap(tracingConfig)
		if diff := cmp.Diff(expected, config.Tracing); diff != "" {
			t.Errorf("Unexpected tracing config (-want, +got): %v", diff)
		}
	})

	t.Run("autoscaler", func(t *testing.T) {
		expected, _ := autoscaler.NewConfigFromConfigMap(autoscalerConfig)
		if diff := cmp.Diff(expected, config.Autoscaler); diff != "" {
			t.Errorf("Unexpected autoscaler config (-want, +got): %v", diff)
		}
	})
}

func TestStoreImmutableConfig(t *testing.T) {
	defer logtesting.ClearAll()
	store := NewStore(logtesting.TestLogger(t))

	store.OnConfigChanged(ConfigMapFromTestFile(t, deployment.ConfigName, deployment.QueueSidecarImageKey))
	store.OnConfigChanged(ConfigMapFromTestFile(t, network.ConfigName))
	store.OnConfigChanged(ConfigMapFromTestFile(t, pkgmetrics.ConfigMapName()))
	store.OnConfigChanged(ConfigMapFromTestFile(t, pkglogging.ConfigMapName()))
	store.OnConfigChanged(ConfigMapFromTestFile(t, pkgtracing.ConfigName))
	store.OnConfigChanged(ConfigMapFromTestFile(t, autoscaler.ConfigName))

	config := store.Load()

	config.Deployment.QueueSidecarImage = "mutated"
	config.Network.IstioOutboundIPRanges = "mutated"
	config.Logging.LoggingConfig = "mutated"
	config.Autoscaler.MaxScaleUpRate = rand.Float64()

	newConfig := store.Load()

	if newConfig.Deployment.QueueSidecarImage == "mutated" {
		t.Error("Controller config is not immutable")
	}
	if newConfig.Network.IstioOutboundIPRanges == "mutated" {
		t.Error("Network config is not immutable")
	}
	if newConfig.Logging.LoggingConfig == "mutated" {
		t.Error("Logging config is not immutable")
	}
	if newConfig.Autoscaler.MaxScaleUpRate == config.Autoscaler.MaxScaleUpRate {
		t.Error("Autoscaler config is not immutable")
	}
}
