/*
Copyright 2019 The Kubernetes Authors.

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

package phases

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/cmd/kubeadm/app/cmd/phases/workflow"
)

// NewRescheduleDeploymentsPhase forces a rescheduling of pods that belong to deployments
func NewRescheduleDeploymentsPhase() workflow.Phase {
	return workflow.Phase{
		Name:    "reschedule-deployments",
		Short:   "Reschedule deployments",
		Example: controlPlaneJoinExample,
		Phases: []workflow.Phase{
			{
				Name:           "all",
				Short:          "Reschedule deployments",
				RunAllSiblings: true,
				ArgsValidator:  cobra.NoArgs,
			},
			newDNSRescheduleLocalSubphase(),
		},
	}
}

func newDNSRescheduleLocalSubphase() workflow.Phase {
	return workflow.Phase{
		Name:          "dns",
		Short:         "Reschedule DNS deployment",
		Run:           runDnsReschedulePhase,
		ArgsValidator: cobra.NoArgs,
	}
}

func runDnsReschedulePhase(c workflow.RunData) error {
	data, ok := c.(JoinData)
	if !ok {
		return errors.New("control-plane-join phase invoked with an invalid data struct")
	}

	client, err := data.ClientSet()
	if err != nil {
		return errors.Wrap(err, "couldn't create Kubernetes client")
	}

	dnsPods, err := client.CoreV1().Pods("kube-system").List(metav1.ListOptions{LabelSelector: "k8s-app=kube-dns"})
	if err != nil {
		return errors.Wrap(err, "couldn't list DNS pods")
	}
	if len(dnsPods.Items) <= 1 {
		// If we don't have more than one replica, nothing to do
		return nil
	}
	dnsNode := dnsPods.Items[0].Spec.NodeName
	for _, dnsPod := range dnsPods.Items[1:] {
		if dnsPod.Spec.NodeName != dnsNode {
			return nil
		}
	}
	for _, dnsPodToDelete := range dnsPods.Items[0 : len(dnsPods.Items)/2] {
		client.CoreV1().Pods("kube-system").Delete(dnsPodToDelete.Name, &metav1.DeleteOptions{})
	}
	return nil
}
