/*
Copyright 2022 quarkcm Authors.

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

package grpc

import (
	context "context"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"

	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/datastore"
	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/objects"
)

var (
	port = flag.Int("port", 51051, "The server port")
)

type server struct {
	UnimplementedQuarkCMServiceServer
}

func StartServer() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		klog.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	RegisterQuarkCMServiceServer(s, &server{})
	klog.Infof("grpc server listening at %v", lis.Addr())
	go s.Serve(lis)
}

// To quickly test this function, execute:
// grpcurl -plaintext -import-path pkg/grpc/ -proto quarkcmsvc.proto -d '{"client_name": "a client name"}' [::]:51051 quarkcmsvc.QuarkCMService/TestPing
func (s *server) TestPing(ctx context.Context, in *TestRequestMessage) (*TestResponseMessage, error) {
	inStr, _ := json.Marshal(in)
	klog.Infof("grpc Service called TestPing %s", inStr)
	hostname, _ := os.Hostname()
	return &TestResponseMessage{ServerName: hostname}, nil
}

func (s *server) ListNode(ctx context.Context, in *emptypb.Empty) (*NodeListMessage, error) {
	klog.Info("grpc Service called ListNode")

	nodeEventObjects := datastore.ListNode(0)
	length := len(nodeEventObjects)
	nodeMessages := make([]*NodeMessage, 0, length)
	for i := 0; i < length; i++ {
		nodeEventObject := nodeEventObjects[i]
		nodeMessages = append(nodeMessages, &NodeMessage{
			Name:              nodeEventObject.NodeObject.Name,
			Hostname:          nodeEventObject.NodeObject.Hostname,
			Ip:                convertIP(nodeEventObject.NodeObject.IP),
			CreationTimestamp: nodeEventObject.NodeObject.CreationTimestamp,
			Subnet:            nodeEventObject.NodeObject.Subnet,
			NetMask:           nodeEventObject.NodeObject.NetMask,
			ResourceVersion:   int32(nodeEventObject.ResourceVersion),
			EventType:         nodeEventObject.EventType, // todo Hong: Change nodeevent to node object
		})
	}

	return &NodeListMessage{Nodes: nodeMessages}, nil
}

func (s *server) WatchNode(maxResourceVersionMessage *MaxResourceVersionMessage, stream QuarkCMService_WatchNodeServer) error {
	klog.Info("grpc Service called WatchNode")

	key := uuid.New()
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	defer queue.ShutDown()
	datastore.AddNodeQueue(key, queue)
	defer datastore.RemoveNodeQueue(key)

	nodeEventObjects := datastore.ListNode(int(maxResourceVersionMessage.MaxResourceVersion))
	for _, nodeEventObject := range nodeEventObjects {
		if err := sendNodeStream(stream, &nodeEventObject); err != nil {
			return err
		}
	}

	for {
		exit, err := processNextNode(queue, stream)
		if exit {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func processNextNode(queue workqueue.RateLimitingInterface, stream QuarkCMService_WatchNodeServer) (bool, error) {
	nodeEventObject, exit := dequeueNode(queue)
	if exit {
		return exit, nil
	}
	return exit, sendNodeStream(stream, nodeEventObject)
}

func dequeueNode(queue workqueue.RateLimitingInterface) (*objects.NodeEventObject, bool) {
	queueItem, exit := queue.Get()
	if exit {
		return nil, exit
	}
	nodeEventObject := queueItem.(objects.NodeEventObject)
	queue.Forget(queueItem)
	// defer queue.Done(queueItem)
	queue.Done(queueItem)
	return &nodeEventObject, exit
}

func sendNodeStream(stream QuarkCMService_WatchNodeServer, nodeEventObject *objects.NodeEventObject) error {
	nodeMessage := &NodeMessage{
		Name:              nodeEventObject.NodeObject.Name,
		Hostname:          nodeEventObject.NodeObject.Hostname,
		Ip:                convertIP(nodeEventObject.NodeObject.IP),
		CreationTimestamp: nodeEventObject.NodeObject.CreationTimestamp,
		ResourceVersion:   int32(nodeEventObject.ResourceVersion),
		EventType:         nodeEventObject.EventType,
	}
	if err := stream.Send(nodeMessage); err != nil {
		return err
	}
	return nil
}

func (s *server) ListPod(ctx context.Context, in *emptypb.Empty) (*PodListMessage, error) {
	klog.Info("grpc Service called ListPod")

	podEventObjects := datastore.ListPod(0)
	length := len(podEventObjects)
	podMessages := make([]*PodMessage, 0, length)
	for i := 0; i < length; i++ {
		podEventObject := podEventObjects[i]
		podMessages = append(podMessages, &PodMessage{
			Key:             podEventObject.PodObject.Key,
			Ip:              convertIP(podEventObject.PodObject.IP),
			NodeName:        podEventObject.PodObject.NodeName,
			ContainerId:     podEventObject.PodObject.ContainerID,
			ResourceVersion: int32(podEventObject.ResourceVersion),
			EventType:       podEventObject.EventType,
		})
	}

	return &PodListMessage{Pods: podMessages}, nil
}

func (s *server) WatchPod(maxResourceVersionMessage *MaxResourceVersionMessage, stream QuarkCMService_WatchPodServer) error {
	klog.Info("grpc Service called WatchPod")

	key := uuid.New()
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	defer queue.ShutDown()
	datastore.AddPodQueue(key, queue)
	defer datastore.RemovePodQueue(key)

	podEventObjects := datastore.ListPod(int(maxResourceVersionMessage.MaxResourceVersion))
	for _, podEventObject := range podEventObjects {
		if err := sendPodStream(stream, &podEventObject); err != nil {
			return err
		}
	}

	for {
		exit, err := processNextPod(queue, stream)
		if exit {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func processNextPod(queue workqueue.RateLimitingInterface, stream QuarkCMService_WatchPodServer) (bool, error) {
	podEventObject, exit := dequeuePod(queue)
	if exit {
		return exit, nil
	}
	return exit, sendPodStream(stream, podEventObject)
}

func dequeuePod(queue workqueue.RateLimitingInterface) (*objects.PodEventObject, bool) {
	queueItem, exit := queue.Get()
	if exit {
		return nil, exit
	}
	podEventObject := queueItem.(objects.PodEventObject)
	queue.Forget(queueItem)
	// defer queue.Done(queueItem)
	queue.Done(queueItem)
	return &podEventObject, exit
}

func sendPodStream(stream QuarkCMService_WatchPodServer, podEventObject *objects.PodEventObject) error {
	podMessage := &PodMessage{
		Key:             podEventObject.PodObject.Key,
		Ip:              convertIP(podEventObject.PodObject.IP),
		NodeName:        podEventObject.PodObject.NodeName,
		ContainerId:     podEventObject.PodObject.ContainerID,
		ResourceVersion: int32(podEventObject.ResourceVersion),
		EventType:       podEventObject.EventType,
	}
	if err := stream.Send(podMessage); err != nil {
		return err
	}
	return nil
}

func convertIP(ip string) uint32 {
	return binary.LittleEndian.Uint32(net.ParseIP(ip).To4())
}
