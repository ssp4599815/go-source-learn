// +build integration,!no-etcd

/*
Copyright 2014 The Kubernetes Authors All rights reserved.

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

package integration

import (
	"strconv"
	"testing"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/testapi"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/tools"
	"k8s.io/kubernetes/pkg/tools/etcdtest"
	"k8s.io/kubernetes/pkg/watch"
	"k8s.io/kubernetes/test/integration/framework"
)

type stringCodec struct{}

type fakeAPIObject string

func (*fakeAPIObject) IsAnAPIObject() {}

func (c stringCodec) Encode(obj runtime.Object) ([]byte, error) {
	return []byte(*obj.(*fakeAPIObject)), nil
}

func (c stringCodec) Decode(data []byte) (runtime.Object, error) {
	o := fakeAPIObject(data)
	return &o, nil
}

func (c stringCodec) DecodeInto(data []byte, obj runtime.Object) error {
	o := obj.(*fakeAPIObject)
	*o = fakeAPIObject(data)
	return nil
}

func TestSetObj(t *testing.T) {
	client := framework.NewEtcdClient()
	helper := tools.EtcdHelper{Client: client, Codec: stringCodec{}}
	framework.WithEtcdKey(func(key string) {
		fakeObject := fakeAPIObject("object")
		if err := helper.SetObj(key, &fakeObject, nil, 0); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		resp, err := client.Get(key, false, false)
		if err != nil || resp.Node == nil {
			t.Fatalf("unexpected error: %v %v", err, resp)
		}
		if resp.Node.Value != "object" {
			t.Errorf("unexpected response: %#v", resp.Node)
		}
	})
}

func TestExtractObj(t *testing.T) {
	client := framework.NewEtcdClient()
	helper := tools.EtcdHelper{Client: client, Codec: stringCodec{}}
	framework.WithEtcdKey(func(key string) {
		_, err := client.Set(key, "object", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s := fakeAPIObject("")
		if err := helper.ExtractObj(key, &s, false); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s != "object" {
			t.Errorf("unexpected response: %#v", s)
		}
	})
}

func TestWatch(t *testing.T) {
	client := framework.NewEtcdClient()
	helper := tools.NewEtcdHelper(client, testapi.Codec(), etcdtest.PathPrefix())
	framework.WithEtcdKey(func(key string) {
		key = etcdtest.AddPrefix(key)
		resp, err := client.Set(key, runtime.EncodeOrDie(testapi.Codec(), &api.Pod{ObjectMeta: api.ObjectMeta{Name: "foo"}}), 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expectedVersion := resp.Node.ModifiedIndex

		// watch should load the object at the current index
		w, err := helper.Watch(key, 0, tools.Everything)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		event := <-w.ResultChan()
		if event.Type != watch.Added || event.Object == nil {
			t.Fatalf("expected first value to be set to ADDED, got %#v", event)
		}

		// version should match what we set
		pod := event.Object.(*api.Pod)
		if pod.ResourceVersion != strconv.FormatUint(expectedVersion, 10) {
			t.Errorf("expected version %d, got %#v", expectedVersion, pod)
		}

		// should be no events in the stream
		select {
		case event, ok := <-w.ResultChan():
			if !ok {
				t.Fatalf("channel closed unexpectedly")
			}
			t.Fatalf("unexpected object in channel: %#v", event)
		default:
		}

		// should return the previously deleted item in the watch, but with the latest index
		resp, err = client.Delete(key, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expectedVersion = resp.Node.ModifiedIndex
		event = <-w.ResultChan()
		if event.Type != watch.Deleted {
			t.Errorf("expected deleted event %#v", event)
		}
		pod = event.Object.(*api.Pod)
		if pod.ResourceVersion != strconv.FormatUint(expectedVersion, 10) {
			t.Errorf("expected version %d, got %#v", expectedVersion, pod)
		}
	})
}

func TestMigrateKeys(t *testing.T) {
	withEtcdKey(func(oldPrefix string) {
		client := newEtcdClient()
		helper := tools.NewEtcdHelper(client, testapi.Codec(), oldPrefix)

		key1 := oldPrefix + "/obj1"
		key2 := oldPrefix + "/foo/obj2"
		key3 := oldPrefix + "/foo/bar/obj3"

		// Create a new entres - these are the 'existing' entries with old prefix
		_, _ = helper.Client.Create(key1, "foo", 0)
		_, _ = helper.Client.Create(key2, "foo", 0)
		_, _ = helper.Client.Create(key3, "foo", 0)

		// Change the helper to a new prefix
		newPrefix := "/kubernetes.io"
		helper = tools.NewEtcdHelper(client, testapi.Codec(), newPrefix)

		// Migrate the keys
		err := helper.MigrateKeys(oldPrefix)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Check the resources are at the correct new location
		newNames := []string{
			newPrefix + "/obj1",
			newPrefix + "/foo/obj2",
			newPrefix + "/foo/bar/obj3",
		}
		for _, name := range newNames {
			_, err := helper.Client.Get(name, false, false)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
		}

		// Check the old locations are removed
		if _, err := helper.Client.Get(oldPrefix, false, false); err == nil {
			t.Fatalf("Old directory still exists.")
		}
	})
}
