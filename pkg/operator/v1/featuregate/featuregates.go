/*
Copyright 2021 Alibaba Group Holding Limited.

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

package featuregate

var (
	featureGateStore = make(map[string]*FeatureGate)
)

type FeatureGate struct {
	key         string
	enabled     bool
	static      bool
	description string
}

func (g *FeatureGate) Key() string {
	return g.key
}

func (g *FeatureGate) Enabled() bool {
	return g.enabled
}

func (g *FeatureGate) Description() string {
	return g.description
}

func declareFeatureGate(key string, enabled, static bool, description string) *FeatureGate {
	_, ok := featureGateStore[key]
	if ok {
		panic("duplicate feature gate: " + key)
	}

	featureGate := &FeatureGate{
		key:         key,
		enabled:     enabled,
		static:      static,
		description: description,
	}

	featureGateStore[key] = featureGate
	return featureGate
}

// Feature gates to prevent unstable or developing codes from running.
var (
	StoreUpgrade                    = declareFeatureGate("StoreUpgrade", false, true, "Enable store upgrading.")
	StoreDynamicConfig              = declareFeatureGate("StoreDynamicConfig", false, true, "Enable dynamic config updating on stores.")
	AutoDataRebalance               = declareFeatureGate("AutoDataRebalance", true, true, "Rebalance data automatically when scaling.")
	WaitDrainedNodeToBeOffline      = declareFeatureGate("WaitDrainedNodeToBeOffline", true, true, "Enable waiting until drained nodes are marked offline when no CDC nodes found.")
	EnableGalaxyClusterMode         = declareFeatureGate("EnableGalaxyCluster", false, true, "Enable cluster mode on galaxy store engine.")
	EnforceQoSGuaranteed            = declareFeatureGate("EnforceQoSGuaranteed", false, false, "Enforce pod's QoS to Guaranteed.")
	ResetTrustIpsBeforeStart        = declareFeatureGate("ResetTrustIpsBeforeStart", false, true, "Reset trust ips in CNs to avoid security problems.")
	EnableXStoreWithHeadlessService = declareFeatureGate("EnableXStoreWithHeadlessService", true, false, "Use headless services for pods in xstore.")
)

func EnableFeatureGates(featureGates []string) {
	for _, featureGate := range featureGates {
		fg := featureGateStore[featureGate]
		if fg != nil && !fg.static {
			fg.enabled = true
		}
	}
}
