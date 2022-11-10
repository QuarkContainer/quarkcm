module github.com/CentaurusInfra/quarkcm

go 1.14

require (
	github.com/ahmetalpbalkan/go-linq v3.0.0+incompatible
	github.com/containernetworking/cni v0.8.1
	github.com/containernetworking/plugins v0.9.1
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/google/uuid v1.1.2
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/smartystreets/goconvey v1.7.2
	github.com/spf13/cobra v0.0.1
	github.com/vishvananda/netlink v1.2.0-beta
	google.golang.org/genproto v0.0.0-20220407144326-9054f6ed7bac // indirect
	google.golang.org/grpc v1.45.0
	google.golang.org/protobuf v1.28.1
	k8s.io/api v0.0.0-20221108053747-3f61c95cab71
	k8s.io/apimachinery v0.0.0-20221108052757-4fe4321a9d5e
	k8s.io/client-go v0.0.0-20221108054908-3daf180aa6b1
	k8s.io/code-generator v0.0.0-20221108000200-7429fbb99432
	k8s.io/klog v1.0.0
	k8s.io/klog/v2 v2.80.1
)

replace (
	k8s.io/api => k8s.io/api v0.0.0-20221108053747-3f61c95cab71
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20221108052757-4fe4321a9d5e
	k8s.io/client-go => k8s.io/client-go v0.0.0-20221108054908-3daf180aa6b1
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20221108000200-7429fbb99432
)
