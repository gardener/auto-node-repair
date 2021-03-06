/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

This file was copied from the kubernetes autoscaler project
https://github.com/kubernetes/autoscaler/blob/cluster-autorepair-1.0.0/cluster-autoscaler/util/deletetaint/delete_test.go
*/

package deletetaint

import (
	"testing"
	"time"

	. "github.com/gardener/auto-node-repair/utils/test"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"

	"github.com/stretchr/testify/assert"
)

func TestMarkNodes(t *testing.T) {
	node := BuildTestNode("node", 1000, 1000)
	fakeClient, updatedNodes := buildFakeClientAndUpdateChannel(node)
	err := MarkToBeDeleted(node, fakeClient)
	assert.NoError(t, err)
	assert.Equal(t, node.Name, getStringFromChan(updatedNodes))
	assert.True(t, HasToBeDeletedTaint(node))
}

func TestCheckNodes(t *testing.T) {
	node := BuildTestNode("node", 1000, 1000)
	fakeClient, updatedNodes := buildFakeClientAndUpdateChannel(node)
	err := MarkToBeDeleted(node, fakeClient)
	assert.NoError(t, err)
	assert.Equal(t, node.Name, getStringFromChan(updatedNodes))
	assert.True(t, HasToBeDeletedTaint(node))

	val, err := GetToBeDeletedTime(node)
	assert.NoError(t, err)
	assert.True(t, time.Now().Sub(*val) < 10*time.Second)
}

func TestCleanNodes(t *testing.T) {
	node := BuildTestNode("node", 1000, 1000)
	addToBeDeletedTaint(node)
	fakeClient, updatedNodes := buildFakeClientAndUpdateChannel(node)

	cleaned, err := CleanToBeDeleted(node, fakeClient)
	assert.True(t, cleaned)
	assert.NoError(t, err)
	assert.Equal(t, node.Name, getStringFromChan(updatedNodes))
	assert.False(t, HasToBeDeletedTaint(node))
}

func buildFakeClientAndUpdateChannel(node *apiv1.Node) (*fake.Clientset, chan string) {
	fakeClient := &fake.Clientset{}
	updatedNodes := make(chan string, 10)
	fakeClient.Fake.AddReactor("get", "nodes", func(action core.Action) (bool, runtime.Object, error) {
		get := action.(core.GetAction)
		if get.GetName() == node.Name {
			return true, node, nil
		}
		return true, nil, errors.NewNotFound(apiv1.Resource("node"), get.GetName())
	})
	fakeClient.Fake.AddReactor("update", "nodes", func(action core.Action) (bool, runtime.Object, error) {
		update := action.(core.UpdateAction)
		obj := update.GetObject().(*apiv1.Node)
		updatedNodes <- obj.Name
		return true, obj, nil
	})
	return fakeClient, updatedNodes
}

func getStringFromChan(c chan string) string {
	select {
	case val := <-c:
		return val
	case <-time.After(time.Second * 10):
		return "Nothing returned"
	}
}
