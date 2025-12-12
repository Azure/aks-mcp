package inspektorgadget

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/inspektor-gadget/inspektor-gadget/pkg/datasource"
	igjson "github.com/inspektor-gadget/inspektor-gadget/pkg/datasource/formatters/json"
	"github.com/inspektor-gadget/inspektor-gadget/pkg/environment"
	gadgetcontext "github.com/inspektor-gadget/inspektor-gadget/pkg/gadget-context"
	"github.com/inspektor-gadget/inspektor-gadget/pkg/gadget-service/api"
	"github.com/inspektor-gadget/inspektor-gadget/pkg/operators"
	"github.com/inspektor-gadget/inspektor-gadget/pkg/operators/simple"
	grpcruntime "github.com/inspektor-gadget/inspektor-gadget/pkg/runtime/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
)

const maxResultLen = 64 * 1024 // 64kb

var KubernetesFlags = genericclioptions.NewConfigFlags(false)

// GadgetManager defines the interface for managing Inspektor Gadget gadgets
type GadgetManager interface {
	// RunGadget runs a gadget with the given parameters for a specified duration
	RunGadget(ctx context.Context, image string, params map[string]string, duration time.Duration) (string, error)
	// StartGadget starts a gadget with the given parameters
	StartGadget(ctx context.Context, image string, params map[string]string, tags []string) (string, error)
	// StopGadget stops a running gadget by its ID
	StopGadget(ctx context.Context, id string) error
	// GetResults retrieves results for a gadget by its ID
	GetResults(ctx context.Context, id string) (string, error)
	// ListGadgets lists all running gadgets
	ListGadgets(ctx context.Context) ([]*GadgetInstance, error)
	// IsDeployed checks if the Inspektor Gadget is deployed in the environment
	IsDeployed(ctx context.Context) (bool, string, error)
	// GetVersion retrieves the version of Inspektor Gadget installed in the cluster
	GetVersion() (string, error)
}

// GadgetInstance represents a running gadget instance
type GadgetInstance struct {
	ID           string            `json:"id"`
	GadgetName   string            `json:"gadgetName"`
	GadgetImage  string            `json:"gadgetImage"`
	FilterParams map[string]string `json:"filterParams,omitempty"`
	Namespaces   []string          `json:"namespaces,omitempty"`
	CreatedBy    string            `json:"createdBy,omitempty"`
	StartedAt    string            `json:"startedAt,omitempty"`
}

func init() {
	environment.Environment = environment.Kubernetes
}

// NewGadgetManager creates a new instance of GadgetManager
func NewGadgetManager() GadgetManager {
	return &manager{}
}

type manager struct {
	formatterMu sync.Mutex
}

// RunGadget runs a gadget with the specified image and parameters for a given duration
func (g *manager) RunGadget(ctx context.Context, image string, params map[string]string, duration time.Duration) (string, error) {
	var results strings.Builder
	gadgetCtx := gadgetcontext.New(
		ctx,
		image,
		gadgetcontext.WithDataOperators(
			g.outputDataOperator(func(data []byte) {
				results.Write(data)
				results.WriteByte('\n')
			}),
		),
		gadgetcontext.WithTimeout(duration),
	)

	rt, err := getRuntime()
	if err != nil {
		return "", fmt.Errorf("getting runtime: %w", err)
	}

	runtimeParams := rt.ParamDescs().ToParams()
	if node, ok := params[paramNode]; ok && node != "" {
		if err = runtimeParams.Set(paramNode, node); err != nil {
			return "", fmt.Errorf("setting node parameter: %w", err)
		}
		delete(params, paramNode)
	}

	if err := rt.RunGadget(gadgetCtx, runtimeParams, params); err != nil {
		return "", fmt.Errorf("running gadget: %w", err)
	}

	return truncateResults(results.String(), false), nil
}

func truncateResults(results string, latest bool) string {
	if len(results) <= maxResultLen {
		return fmt.Sprintf("\n<results>%s</results>\n", results)
	}

	var truncated string
	if latest {
		truncated = results[len(results)-maxResultLen:]
	} else {
		truncated = results[:maxResultLen] + "…"
	}

	return fmt.Sprintf("\n<isTruncated>true</isTruncated>\n<results>%s</results>\n", truncated)
}

func (g *manager) outputDataOperator(cb func(data []byte)) operators.DataOperator {
	const opPriority = 50000
	return simple.New("outputDataOperator",
		simple.OnInit(func(gadgetCtx operators.GadgetContext) error {
			for _, d := range gadgetCtx.GetDataSources() {
				// skip data sources that have the annotation "cli.default-output-mode"
				// set to "none"
				if m, ok := d.Annotations()["cli.default-output-mode"]; ok && m == "none" {
					continue
				}

				// handle adding a raw string field for certain content types
				restField := d.Annotations()["ebpf.rest.name"]
				var restAcc datasource.FieldAccessor
				var restStrAcc datasource.FieldAccessor
				var err error
				if restField != "" {
					restAcc = d.GetField(restField)
					ct, ok := restAcc.Annotations()["content-type"]
					if ok && ct == "application/x-raw-packet" {
						restStrAcc, err = d.AddField(restField+"_string", api.Kind_String)
						if err != nil {
							return fmt.Errorf("adding raw string field accessor: %w", err)
						}
					}
				}

				jsonFormatter, err := igjson.New(d,
					// TODO: Maybe we should use specific fields or remove unwanted fields
					igjson.WithShowAll(true),
				)
				if err != nil {
					return fmt.Errorf("creating JSON formatter: %w", err)
				}

				err = d.Subscribe(func(source datasource.DataSource, data datasource.Data) error {
					g.formatterMu.Lock()
					defer g.formatterMu.Unlock()
					if restAcc != nil && restStrAcc != nil {
						pktStr := gopacket.NewPacket(restAcc.Get(data), layers.LinkTypeEthernet, gopacket.Default).String()
						err = restStrAcc.Set(data, []byte(pktStr))
						if err != nil {
							return fmt.Errorf("setting raw string field: %w", err)
						}
					}
					jsonData := jsonFormatter.Marshal(data)
					cb(jsonData)
					return nil
				}, opPriority)
				if err != nil {
					return fmt.Errorf("subscribing to data source %q: %w", d.Name(), err)
				}
			}
			return nil
		}),
	)
}

// StartGadget starts a gadget with the specified name and image, returning its ID
func (g *manager) StartGadget(ctx context.Context, image string, params map[string]string, tags []string) (string, error) {
	gadgetCtx := gadgetcontext.New(
		ctx,
		image,
	)

	rt, err := getRuntime()
	if err != nil {
		return "", fmt.Errorf("getting runtime: %w", err)
	}
	p := rt.ParamDescs().ToParams()

	newID := make([]byte, 16)
	_, err = rand.Read(newID)
	if err != nil {
		return "", fmt.Errorf("generating new gadget ID: %w", err)
	}
	idString := hex.EncodeToString(newID)

	err = p.Set(grpcruntime.ParamID, idString)
	if err != nil {
		return "", fmt.Errorf("setting gadget ID: %w", err)
	}
	err = p.Set(grpcruntime.ParamDetach, "true")
	if err != nil {
		return "", fmt.Errorf("setting detach parameter: %w", err)
	}
	if err = p.Set(grpcruntime.ParamTags, strings.Join(append(tags, "createdBy=aks-mcp"), ",")); err != nil {
		return "", fmt.Errorf("setting gadget tags: %w", err)
	}
	if err = rt.RunGadget(gadgetCtx, p, params); err != nil {
		return "", fmt.Errorf("running gadget: %w", err)
	}

	return idString, nil
}

// StopGadget stops a running gadget by its ID
func (g *manager) StopGadget(ctx context.Context, id string) error {
	rt, err := getRuntime()
	if err != nil {
		return fmt.Errorf("getting runtime: %w", err)
	}

	if err = rt.RemoveGadgetInstance(ctx, rt.ParamDescs().ToParams(), id); err != nil {
		return fmt.Errorf("stopping gadget: %w", err)
	}
	return nil
}

// GetResults retrieves results for a gadget by its ID
func (g *manager) GetResults(ctx context.Context, id string) (string, error) {
	to, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	var results strings.Builder
	gadgetCtx := gadgetcontext.New(
		to,
		id,
		gadgetcontext.WithDataOperators(
			g.outputDataOperator(func(data []byte) {
				results.Write(data)
				results.WriteByte('\n')
			}),
		),
		gadgetcontext.WithID(id),
		gadgetcontext.WithUseInstance(true),
		gadgetcontext.WithTimeout(time.Second),
	)

	rt, err := getRuntime()
	if err != nil {
		return "", fmt.Errorf("getting runtime: %w", err)
	}

	if err = rt.RunGadget(gadgetCtx, rt.ParamDescs().ToParams(), map[string]string{}); err != nil {
		return "", fmt.Errorf("attaching to gadget: %w", err)
	}

	return truncateResults(results.String(), true), nil
}

// ListGadgets lists all running gadgets and returns their instances
func (g *manager) ListGadgets(ctx context.Context) ([]*GadgetInstance, error) {
	rt, err := getRuntime()
	if err != nil {
		return nil, fmt.Errorf("getting runtime: %w", err)
	}

	instances, err := rt.GetGadgetInstances(ctx, rt.ParamDescs().ToParams())
	if err != nil {
		return nil, fmt.Errorf("listing gadgets: %w", err)
	}

	var gadgetInstances []*GadgetInstance
	for _, instance := range instances {
		inst := gadgetInstanceFromAPI(instance)
		if inst != nil {
			gadgetInstances = append(gadgetInstances, inst)
		}
	}
	return gadgetInstances, nil
}

// IsDeployed checks if the Inspektor Gadget is deployed in the Kubernetes
func (g *manager) IsDeployed(ctx context.Context) (bool, string, error) {
	restConfig, err := KubernetesFlags.ToRESTConfig()
	if err != nil {
		return false, "", fmt.Errorf("creating RESTConfig: %w", err)
	}

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return false, "", fmt.Errorf("setting up trace client: %w", err)
	}

	opts := metav1.ListOptions{LabelSelector: "k8s-app=gadget"}
	pods, err := client.CoreV1().Pods("").List(ctx, opts)
	if err != nil {
		return false, "", fmt.Errorf("getting pods: %w", err)
	}
	if len(pods.Items) == 0 {
		return false, "", nil
	}

	var namespaces []string
	for _, pod := range pods.Items {
		if !slices.Contains(namespaces, pod.Namespace) {
			namespaces = append(namespaces, pod.Namespace)
		}
	}
	if len(namespaces) > 1 {
		return false, "", fmt.Errorf("multiple namespaces found for Inspektor Gadget pods: %v", namespaces)
	}
	return true, namespaces[0], nil
}

func (g *manager) GetVersion() (string, error) {
	rt, err := getRuntime()
	if err != nil {
		return "", fmt.Errorf("getting runtime: %w", err)
	}

	info, err := rt.GetInfo()
	if err != nil {
		return "", fmt.Errorf("getting info: %w", err)
	}
	return info.ServerVersion, nil
}

// getRuntime sets up a runtime, ensuring we always use the latest kubeconfig
func getRuntime() (*grpcruntime.Runtime, error) {
	rt := grpcruntime.New(grpcruntime.WithConnectUsingK8SProxy)
	if err := rt.Init(nil); err != nil {
		return nil, fmt.Errorf("initializing gadget runtime: %w", err)
	}

	restConfig, err := KubernetesFlags.ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("creating REST config: %w", err)
	}
	rt.SetRestConfig(restConfig)

	return rt, nil
}
