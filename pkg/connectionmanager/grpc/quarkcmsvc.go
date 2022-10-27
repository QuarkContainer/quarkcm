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
	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/utils"
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
			Ip:                utils.ConvertIP(nodeEventObject.NodeObject.IP),
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
		Ip:                utils.ConvertIP(nodeEventObject.NodeObject.IP),
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
			Ip:              utils.ConvertIP(podEventObject.PodObject.IP),
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
		Ip:              utils.ConvertIP(podEventObject.PodObject.IP),
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

func (s *server) ListService(ctx context.Context, in *emptypb.Empty) (*ServiceListMessage, error) {
	klog.Info("grpc Service called ListService")

	serviceEventObjects := datastore.ListService(0)
	length := len(serviceEventObjects)
	serviceMessages := make([]*ServiceMessage, 0, length)
	for i := 0; i < length; i++ {
		serviceEventObject := serviceEventObjects[i]
		serviceMessages = append(serviceMessages, &ServiceMessage{
			Name:            serviceEventObject.ServiceObject.Name,
			ClusterIp:       utils.ConvertIP(serviceEventObject.ServiceObject.ClusterIP),
			Ports:           serviceEventObject.ServiceObject.Ports,
			ResourceVersion: int32(serviceEventObject.ResourceVersion),
			EventType:       serviceEventObject.EventType,
		})
	}

	return &ServiceListMessage{Services: serviceMessages}, nil
}

func (s *server) WatchService(maxResourceVersionMessage *MaxResourceVersionMessage, stream QuarkCMService_WatchServiceServer) error {
	klog.Info("grpc Service called WatchService")

	key := uuid.New()
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	defer queue.ShutDown()
	datastore.AddServiceQueue(key, queue)
	defer datastore.RemoveServiceQueue(key)

	serviceEventObjects := datastore.ListService(int(maxResourceVersionMessage.MaxResourceVersion))
	for _, serviceEventObject := range serviceEventObjects {
		if err := sendServiceStream(stream, &serviceEventObject); err != nil {
			return err
		}
	}

	for {
		exit, err := processNextService(queue, stream)
		if exit {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func processNextService(queue workqueue.RateLimitingInterface, stream QuarkCMService_WatchServiceServer) (bool, error) {
	serviceEventObject, exit := dequeueService(queue)
	if exit {
		return exit, nil
	}
	return exit, sendServiceStream(stream, serviceEventObject)
}

func dequeueService(queue workqueue.RateLimitingInterface) (*objects.ServiceEventObject, bool) {
	queueItem, exit := queue.Get()
	if exit {
		return nil, exit
	}
	serviceEventObject := queueItem.(objects.ServiceEventObject)
	queue.Forget(queueItem)
	// defer queue.Done(queueItem)
	queue.Done(queueItem)
	return &serviceEventObject, exit
}

func sendServiceStream(stream QuarkCMService_WatchServiceServer, serviceEventObject *objects.ServiceEventObject) error {
	serviceMessage := &ServiceMessage{
		Name:            serviceEventObject.ServiceObject.Name,
		ClusterIp:       utils.ConvertIP(serviceEventObject.ServiceObject.ClusterIP),
		Ports:           serviceEventObject.ServiceObject.Ports,
		ResourceVersion: int32(serviceEventObject.ResourceVersion),
		EventType:       serviceEventObject.EventType,
	}
	if err := stream.Send(serviceMessage); err != nil {
		return err
	}
	return nil
}

func (s *server) ListEndpoints(ctx context.Context, in *emptypb.Empty) (*EndpointsListMessage, error) {
	klog.Info("grpc Endpoints called ListEndpoints")

	endpointsEventObjects := datastore.ListEndpoints(0)
	length := len(endpointsEventObjects)
	endpointsMessages := make([]*EndpointsMessage, 0, length)
	for i := 0; i < length; i++ {
		endpointsEventObject := endpointsEventObjects[i]
		endpointsMessages = append(endpointsMessages, &EndpointsMessage{
			Name:            endpointsEventObject.EndpointsObject.Name,
			IpWithPorts:     endpointsEventObject.EndpointsObject.IPWithPorts,
			ResourceVersion: int32(endpointsEventObject.ResourceVersion),
			EventType:       endpointsEventObject.EventType,
		})
	}

	return &EndpointsListMessage{Endpointses: endpointsMessages}, nil
}

func (s *server) WatchEndpoints(maxResourceVersionMessage *MaxResourceVersionMessage, stream QuarkCMService_WatchEndpointsServer) error {
	klog.Info("grpc Endpoints called WatchEndpoints")

	key := uuid.New()
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	defer queue.ShutDown()
	datastore.AddEndpointsQueue(key, queue)
	defer datastore.RemoveEndpointsQueue(key)

	endpointsEventObjects := datastore.ListEndpoints(int(maxResourceVersionMessage.MaxResourceVersion))
	for _, endpointsEventObject := range endpointsEventObjects {
		if err := sendEndpointsStream(stream, &endpointsEventObject); err != nil {
			return err
		}
	}

	for {
		exit, err := processNextEndpoints(queue, stream)
		if exit {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func processNextEndpoints(queue workqueue.RateLimitingInterface, stream QuarkCMService_WatchEndpointsServer) (bool, error) {
	endpointsEventObject, exit := dequeueEndpoints(queue)
	if exit {
		return exit, nil
	}
	return exit, sendEndpointsStream(stream, endpointsEventObject)
}

func dequeueEndpoints(queue workqueue.RateLimitingInterface) (*objects.EndpointsEventObject, bool) {
	queueItem, exit := queue.Get()
	if exit {
		return nil, exit
	}
	endpointsEventObject := queueItem.(objects.EndpointsEventObject)
	queue.Forget(queueItem)
	// defer queue.Done(queueItem)
	queue.Done(queueItem)
	return &endpointsEventObject, exit
}

func sendEndpointsStream(stream QuarkCMService_WatchEndpointsServer, endpointsEventObject *objects.EndpointsEventObject) error {
	endpointsMessage := &EndpointsMessage{
		Name:            endpointsEventObject.EndpointsObject.Name,
		IpWithPorts:     endpointsEventObject.EndpointsObject.IPWithPorts,
		ResourceVersion: int32(endpointsEventObject.ResourceVersion),
		EventType:       endpointsEventObject.EventType,
	}
	if err := stream.Send(endpointsMessage); err != nil {
		return err
	}
	return nil
}

func (s *server) ListConfigMap(ctx context.Context, in *emptypb.Empty) (*ConfigMapListMessage, error) {
	klog.Info("grpc ConfigMap called ListConfigMap")

	configMapEventObjects := datastore.ListConfigMap(0)
	length := len(configMapEventObjects)
	configMapMessages := make([]*ConfigMapMessage, 0, length)
	for i := 0; i < length; i++ {
		configMapEventObject := configMapEventObjects[i]
		configMapMessages = append(configMapMessages, &ConfigMapMessage{
			Name:            configMapEventObject.ConfigMapObject.Name,
			Value:           configMapEventObject.ConfigMapObject.Value,
			ResourceVersion: int32(configMapEventObject.ResourceVersion),
			EventType:       configMapEventObject.EventType,
		})
	}

	return &ConfigMapListMessage{ConfigMaps: configMapMessages}, nil
}

func (s *server) WatchConfigMap(maxResourceVersionMessage *MaxResourceVersionMessage, stream QuarkCMService_WatchConfigMapServer) error {
	klog.Info("grpc ConfigMap called WatchConfigMap")

	key := uuid.New()
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	defer queue.ShutDown()
	datastore.AddConfigMapQueue(key, queue)
	defer datastore.RemoveConfigMapQueue(key)

	configMapEventObjects := datastore.ListConfigMap(int(maxResourceVersionMessage.MaxResourceVersion))
	for _, configMapEventObject := range configMapEventObjects {
		if err := sendConfigMapStream(stream, &configMapEventObject); err != nil {
			return err
		}
	}

	for {
		exit, err := processNextConfigMap(queue, stream)
		if exit {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func processNextConfigMap(queue workqueue.RateLimitingInterface, stream QuarkCMService_WatchConfigMapServer) (bool, error) {
	configMapEventObject, exit := dequeueConfigMap(queue)
	if exit {
		return exit, nil
	}
	return exit, sendConfigMapStream(stream, configMapEventObject)
}

func dequeueConfigMap(queue workqueue.RateLimitingInterface) (*objects.ConfigMapEventObject, bool) {
	queueItem, exit := queue.Get()
	if exit {
		return nil, exit
	}
	configMapEventObject := queueItem.(objects.ConfigMapEventObject)
	queue.Forget(queueItem)
	// defer queue.Done(queueItem)
	queue.Done(queueItem)
	return &configMapEventObject, exit
}

func sendConfigMapStream(stream QuarkCMService_WatchConfigMapServer, configMapEventObject *objects.ConfigMapEventObject) error {
	configMapMessage := &ConfigMapMessage{
		Name:            configMapEventObject.ConfigMapObject.Name,
		Value:           configMapEventObject.ConfigMapObject.Value,
		ResourceVersion: int32(configMapEventObject.ResourceVersion),
		EventType:       configMapEventObject.EventType,
	}
	if err := stream.Send(configMapMessage); err != nil {
		return err
	}
	return nil
}

func (s *server) ListIngress(ctx context.Context, in *emptypb.Empty) (*IngressListMessage, error) {
	klog.Info("grpc Ingress called ListIngress")

	ingressEventObjects := datastore.ListIngress(0)
	length := len(ingressEventObjects)
	ingressMessages := make([]*IngressMessage, 0, length)
	for i := 0; i < length; i++ {
		ingressEventObject := ingressEventObjects[i]
		ingressMessages = append(ingressMessages, &IngressMessage{
			Name:            ingressEventObject.IngressObject.Name,
			Service:         ingressEventObject.IngressObject.Service,
			PortNumber:      uint32(ingressEventObject.IngressObject.PortNumber),
			ResourceVersion: int32(ingressEventObject.ResourceVersion),
			EventType:       ingressEventObject.EventType,
		})
	}

	return &IngressListMessage{Ingresss: ingressMessages}, nil
}

func (s *server) WatchIngress(maxResourceVersionMessage *MaxResourceVersionMessage, stream QuarkCMService_WatchIngressServer) error {
	klog.Info("grpc Ingress called WatchIngress")

	key := uuid.New()
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	defer queue.ShutDown()
	datastore.AddIngressQueue(key, queue)
	defer datastore.RemoveIngressQueue(key)

	ingressEventObjects := datastore.ListIngress(int(maxResourceVersionMessage.MaxResourceVersion))
	for _, ingressEventObject := range ingressEventObjects {
		if err := sendIngressStream(stream, &ingressEventObject); err != nil {
			return err
		}
	}

	for {
		exit, err := processNextIngress(queue, stream)
		if exit {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func processNextIngress(queue workqueue.RateLimitingInterface, stream QuarkCMService_WatchIngressServer) (bool, error) {
	ingressEventObject, exit := dequeueIngress(queue)
	if exit {
		return exit, nil
	}
	return exit, sendIngressStream(stream, ingressEventObject)
}

func dequeueIngress(queue workqueue.RateLimitingInterface) (*objects.IngressEventObject, bool) {
	queueItem, exit := queue.Get()
	if exit {
		return nil, exit
	}
	ingressEventObject := queueItem.(objects.IngressEventObject)
	queue.Forget(queueItem)
	// defer queue.Done(queueItem)
	queue.Done(queueItem)
	return &ingressEventObject, exit
}

func sendIngressStream(stream QuarkCMService_WatchIngressServer, ingressEventObject *objects.IngressEventObject) error {
	ingressMessage := &IngressMessage{
		Name:            ingressEventObject.IngressObject.Name,
		Service:         ingressEventObject.IngressObject.Service,
		PortNumber:      uint32(ingressEventObject.IngressObject.PortNumber),
		ResourceVersion: int32(ingressEventObject.ResourceVersion),
		EventType:       ingressEventObject.EventType,
	}
	if err := stream.Send(ingressMessage); err != nil {
		return err
	}
	return nil
}
