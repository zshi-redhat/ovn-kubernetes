package cni

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/containernetworking/cni/pkg/types/current"
	"k8s.io/client-go/kubernetes"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"

	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/cni/types"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/util"
)

// serverRunDir is the default directory for CNIServer runtime files
const serverRunDir string = "/var/run/ovn-kubernetes/cni/"

const serverSocketName string = "ovn-cni-server.sock"
const serverSocketPath string = serverRunDir + "/" + serverSocketName

// KubeAPIAuth contains information necessary to create a Kube API client
type KubeAPIAuth struct {
	// Kubeconfig is the path to a kubeconfig
	Kubeconfig string `json:"kubeconfig,omitempty"`
	// KubeAPIServer is the URL of a Kubernetes API server (not required if kubeconfig is given)
	KubeAPIServer string `json:"kube-api-server,omitempty"`
	// KubeAPIToken is a Kubernetes API token (not required if kubeconfig is given)
	KubeAPIToken string `json:"kube-api-token,omitempty"`
	// KubeAPITokenFile is the path to Kubernetes API token
	// If set, it is periodically read and takes precedence over KubeAPIToken
	KubeAPITokenFile string `json:"kube-api-token-file,omitempty"`
	// KubeCAData is the Base64-ed Kubernetes API CA certificate data (not required if kubeconfig is given)
	KubeCAData string `json:"kube-ca-data,omitempty"`
}

// PodInterfaceInfo consists of interface info result from cni server if cni client configure's interface
type PodInterfaceInfo struct {
	PodUID       string `json:"pod-uid"`
	VfNetdevName string `json:"vf-netdev-name"`
	util.PodAnnotation
	Ingress              int64 `json:"ingress"`
	Egress               int64 `json:"egress"`
	MTU                  int   `json:"mtu"`
	RoutableMTU          int   `json:"routable-mtu"`
	CheckExtIDs          bool  `json:"check-external-ids"`
	IsDPUHostMode        bool  `json:"is-dpu-host-mode"`
	EnableUDPAggregation bool  `json:"enable-udp-aggregation"`
}

// Explicit type for CNI commands the server handles
type command string

// CNIAdd is the command representing add operation for a new pod
const CNIAdd command = "ADD"

// CNIUpdate is the command representing update operation for an existing pod
const CNIUpdate command = "UPDATE"

// CNIDel is the command representing delete operation on a pod that is to be torn down
const CNIDel command = "DEL"

// CNICheck is the command representing check operation on a pod
const CNICheck command = "CHECK"

// Request sent to the Server by the OVN CNI plugin
type Request struct {
	// CNI environment variables, like CNI_COMMAND and CNI_NETNS
	Env map[string]string `json:"env,omitempty"`
	// CNI configuration passed via stdin to the CNI plugin
	Config []byte `json:"config,omitempty"`
}

// CNIRequestMetrics info to report from CNI shim to CNI server
type CNIRequestMetrics struct {
	Command     command `json:"command"`
	ElapsedTime float64 `json:"elapsedTime"`
	HasErr      bool    `json:"hasErr"`
}

// Response sent to the OVN CNI plugin by the Server
type Response struct {
	Result    *current.Result
	PodIFInfo *PodInterfaceInfo
	KubeAuth  *KubeAPIAuth
}

func (response *Response) Marshal() ([]byte, error) {
	return json.Marshal(response)
}

// Filter out kubeAuth, since it might contain sensitive information.
func (response *Response) MarshalForLogging() ([]byte, error) {
	var noAuthJSON []byte
	var err error
	var noAuth interface{}

	if response == nil {
		return nil, nil
	}

	// Only one of Result and PodIFInfo is ever set by cmdAdd/cmdDel
	if response.Result != nil {
		noAuth = response.Result
	} else {
		noAuth = response.PodIFInfo
	}

	if noAuthJSON, err = json.Marshal(noAuth); err != nil {
		klog.Errorf("Could not JSON-encode the extracted response: %v", err)
		return nil, err
	}

	return noAuthJSON, nil
}

// PodRequest structure built from Request which is passed to the
// handler function given to the Server at creation time
type PodRequest struct {
	timestamp    time.Time
	ctx          context.Context
	CNIConf      *types.NetConf
	cancel       context.CancelFunc
	Command      command
	Netns        string
	IfName       string
	PodNamespace string
	PodName      string
	PodUID       string
	SandboxID    string
}

type cniRequestFunc func(request *PodRequest, podLister corev1listers.PodLister, useOVSExternalIDs bool, kclient kubernetes.Interface, kubeAuth *KubeAPIAuth) ([]byte, error)

// Server object that listens for JSON-marshaled Request objects
// on a private root-only Unix domain socket.
type Server struct {
	kclient     kubernetes.Interface
	podLister   corev1listers.PodLister
	kubeAuth    *KubeAPIAuth
	requestFunc cniRequestFunc
	rundir      string
	http.Server
	useOVSExternalIDs int32
}
