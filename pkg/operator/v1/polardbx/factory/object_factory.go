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

package factory

import (
	"github.com/alibaba/polardbx-operator/api/v1/polardbx"
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	polardbxv1 "github.com/alibaba/polardbx-operator/api/v1"
	"github.com/alibaba/polardbx-operator/pkg/operator/v1/polardbx/convention"
	polardbxv1reconcile "github.com/alibaba/polardbx-operator/pkg/operator/v1/polardbx/reconcile"
)

type ObjectFactory interface {
	NewService() (*corev1.Service, error)
	NewReadOnlyService() (*corev1.Service, error)
	NewCDCMetricsService() (*corev1.Service, error)

	NewDeployments4CN() (map[string]appsv1.Deployment, error)
	NewDeployments4CDC() (map[string]appsv1.Deployment, error)

	NewXStoreMyCnfOverlay4GMS() (string, error)
	NewXStoreMyCnfOverlay4DN(idx int) (string, error)
	NewXStoreGMS() (*polardbxv1.XStore, error)
	NewXStoreDN(idx int) (*polardbxv1.XStore, error)
	NewSecret() (*corev1.Secret, error)
	NewSecretFromPolarDBX(*corev1.Secret) (*corev1.Secret, error)
	NewSecuritySecret() (*corev1.Secret, error)
	NewConfigMap(cmType convention.ConfigMapType) (*corev1.ConfigMap, error)

	NewServiceMonitors() (map[string]promv1.ServiceMonitor, error)

	NewReadonlyPolardbx(*polardbx.ReadonlyParam) (*polardbxv1.PolarDBXCluster, error)
}

type Context struct {
	HasCdcXBinLog   bool
	BuildCdcXBinLog bool
}

type objectFactory struct {
	rc           *polardbxv1reconcile.Context
	buildContext Context
}

func NewObjectFactory(rc *polardbxv1reconcile.Context) ObjectFactory {
	return &objectFactory{rc: rc, buildContext: Context{}}
}
