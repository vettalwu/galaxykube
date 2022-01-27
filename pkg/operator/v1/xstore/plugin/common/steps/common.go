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

package steps

import (
	"time"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/alibaba/polardbx-operator/pkg/k8s/control"
	k8shelper "github.com/alibaba/polardbx-operator/pkg/k8s/helper"
	"github.com/alibaba/polardbx-operator/pkg/operator/v1/featuregate"
	"github.com/alibaba/polardbx-operator/pkg/operator/v1/xstore/command"
	"github.com/alibaba/polardbx-operator/pkg/operator/v1/xstore/convention"
	xstoremeta "github.com/alibaba/polardbx-operator/pkg/operator/v1/xstore/meta"
	"github.com/alibaba/polardbx-operator/pkg/operator/v1/xstore/plugin"
	"github.com/alibaba/polardbx-operator/pkg/operator/v1/xstore/plugin/common/channel"
	xstorev1reconcile "github.com/alibaba/polardbx-operator/pkg/operator/v1/xstore/reconcile"
)

func parseChannelFromConfigMap(cm *corev1.ConfigMap) (*channel.SharedChannel, error) {
	sharedChannel := &channel.SharedChannel{}
	err := sharedChannel.Load(cm.Data[channel.SharedChannelKey])
	if err != nil {
		return nil, err
	}
	return sharedChannel, nil
}

func transformPodsIntoNodesWithHeadlessServices(namespace string, pods []corev1.Pod) []channel.Node {
	nodes := transformPodsIntoNodes(namespace, pods)
	// Reset every host to DNS.
	for i := range nodes {
		// DNS record for {service} because by default searches {ns}.svc.cluster.local
		nodes[i].Host = convention.NewHeadlessServiceName(nodes[i].Pod)
	}
	return nodes
}

func transformPodsIntoNodes(namespace string, pods []corev1.Pod) []channel.Node {
	nodes := make([]channel.Node, 0, len(pods))
	for _, pod := range pods {
		paxosPort := k8shelper.MustGetPortFromContainer(
			k8shelper.MustGetContainerFromPod(&pod, convention.ContainerEngine),
			"paxos",
		).ContainerPort
		node := channel.Node{
			Pod:      pod.Name,
			Host:     pod.Status.PodIP,
			HostName: pod.Spec.NodeName,
			Port:     int(paxosPort),
			Role:     pod.Labels[xstoremeta.LabelNodeRole],
		}
		if len(pod.Spec.Subdomain) > 0 {
			node.Domain = pod.Name + "." + pod.Spec.Subdomain
		}
		nodes = append(nodes, node)
	}

	return nodes
}

var UnblockBootstrap = xstorev1reconcile.NewStepBinder("UnblockBootstrap",
	func(rc *xstorev1reconcile.Context, flow control.Flow) (reconcile.Result, error) {
		sharedCm, err := rc.GetXStoreConfigMap(convention.ConfigMapTypeShared)
		if err != nil {
			return flow.Error(err, "Unable to get shared config map.")
		}

		sharedChannel, err := parseChannelFromConfigMap(sharedCm)
		if err != nil {
			return flow.Error(err, "Unable to parse shared channel from config map.")
		}

		// Branch currently unblocked, just skip.
		if !sharedChannel.IsBlocked() {
			return flow.Pass()
		}

		// Unblock and set nodes info and others.
		sharedChannel.Unblock()

		pods, err := rc.GetXStorePods()
		if err != nil {
			return flow.Error(err, "Unable to get pods.")
		}

		if featuregate.EnableXStoreWithHeadlessService.Enabled() {
			sharedChannel.Nodes = transformPodsIntoNodesWithHeadlessServices(rc.Namespace(), pods)
		} else {
			sharedChannel.Nodes = transformPodsIntoNodes(rc.Namespace(), pods)
		}

		// update configmap.
		sharedCm.Data[channel.SharedChannelKey] = sharedChannel.String()
		err = rc.Client().Update(rc.Context(), sharedCm)
		if err != nil {
			return flow.Error(err, "Unable to update shared config map.")
		}
		return flow.Continue("Unblock via shared channel.")
	},
)

func setElectionWeightToOne(rc *xstorev1reconcile.Context, log logr.Logger, leaderPod *corev1.Pod, targetPods []corev1.Pod) error {
	cmd := command.NewCanonicalCommandBuilder().
		Consensus().
		ConfigureElectionWeight(1, k8shelper.ToObjectNames(targetPods)...).
		Build()

	return rc.ExecuteCommandOn(leaderPod, convention.ContainerEngine, cmd, control.ExecOptions{
		Logger:  log,
		Timeout: 2 * time.Second,
	})
}

var SetVoterElectionWeightToOne = plugin.NewStepBinder("common", "SetVoterElectionWeightToOne",
	func(rc *xstorev1reconcile.Context, flow control.Flow) (reconcile.Result, error) {
		pods, err := rc.GetXStorePods()
		if err != nil {
			return flow.Error(err, "Unable to get pods.")
		}

		voterPods := k8shelper.FilterPodsBy(pods, func(pod *corev1.Pod) bool {
			return xstoremeta.IsPodRoleVoter(pod)
		})

		if len(voterPods) == 0 {
			return flow.Pass()
		}

		leaderPod, err := rc.TryGetXStoreLeaderPod()
		if err != nil {
			return flow.Error(err, "Unable to get leader pod.")
		}
		if leaderPod == nil {
			return flow.Wait("No leader pod found.")
		}

		err = setElectionWeightToOne(rc, flow.Logger(), leaderPod, voterPods)
		if err != nil {
			return flow.Error(err, "Unable to set election weight to 1.",
				"leader-pod", leaderPod.Name,
				"voter-pods", k8shelper.ToObjectNames(voterPods))
		}

		return flow.Pass()
	},
)
