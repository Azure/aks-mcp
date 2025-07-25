package inspektorgadget

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/inspektor-gadget/inspektor-gadget/pkg/datasource"
	igjson "github.com/inspektor-gadget/inspektor-gadget/pkg/datasource/formatters/json"
	"github.com/inspektor-gadget/inspektor-gadget/pkg/environment"
	gadgetcontext "github.com/inspektor-gadget/inspektor-gadget/pkg/gadget-context"
	"github.com/inspektor-gadget/inspektor-gadget/pkg/operators"
	"github.com/inspektor-gadget/inspektor-gadget/pkg/operators/simple"
	"github.com/inspektor-gadget/inspektor-gadget/pkg/runtime"
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
	StartGadget(ctx context.Context, name, image string, params map[string]string) (string, error)
	// StopGadget stops a running gadget by its ID
	StopGadget(ctx context.Context, id string) error
	// GetResults retrieves results for a gadget by its ID
	GetResults(ctx context.Context, id string) (string, error)
	// ListGadgets lists all running gadgets
	ListGadgets(ctx context.Context) ([]*GadgetInstance, error)
	// IsDeployed checks if the Inspektor Gadget is deployed in the environment
	IsDeployed(ctx context.Context) (bool, string, error)
	// Close closes the gadget manager and releases any resources
	Close() error
}

// GadgetInstance represents a running gadget instance
type GadgetInstance struct {
	ID           string            `json:"id"`
	GadgetName   string            `json:"gadgetName"`
	GadgetImage  string            `json:"gadgetImage"`
	GadgetParams map[string]string `json:"gadgetParams"`
	CreatedBy    string            `json:"createdBy,omitempty"`
	StartedAt    string            `json:"startedAt,omitempty"`
}

func init() {
	environment.Environment = environment.Kubernetes
}

func NewGadgetManager() (GadgetManager, error) {
	rt := grpcruntime.New(grpcruntime.WithConnectUsingK8SProxy)
	if err := rt.Init(nil); err != nil {
		return nil, fmt.Errorf("initializing gadget runtime: %w", err)
	}

	restConfig, err := KubernetesFlags.ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("creating REST config: %w", err)
	}
	rt.SetRestConfig(restConfig)

	return &manager{
		runtime: rt,
	}, nil
}

type manager struct {
	runtime runtime.Runtime
}

func (g *manager) RunGadget(ctx context.Context, image string, params map[string]string, duration time.Duration) (string, error) {
	var results strings.Builder
	gadgetCtx := gadgetcontext.New(
		ctx,
		image,
		gadgetcontext.WithDataOperators(
			outputDataOperator(func(data []byte) {
				results.Write(data)
				results.WriteByte('\n')
			}),
		),
		gadgetcontext.WithTimeout(duration),
	)

	if err := g.runtime.RunGadget(gadgetCtx, g.runtime.ParamDescs().ToParams(), params); err != nil {
		return "", fmt.Errorf("running gadget: %w", err)
	}

	return truncateResults(results.String()), nil
}

func truncateResults(results string) string {
	if len(results) > maxResultLen {
		return fmt.Sprintf("\n<isTruncated>true</isTruncated>\n<results>%s</results>\n", results[:maxResultLen]+"…")
	}
	return fmt.Sprintf("\n<results>%s</results>\n", results)
}

func outputDataOperator(cb func(data []byte)) operators.DataOperator {
	const opPriority = 50000
	return simple.New("outputDataOperator",
		simple.OnInit(func(gadgetCtx operators.GadgetContext) error {
			for _, d := range gadgetCtx.GetDataSources() {
				jsonFormatter, _ := igjson.New(d,
					igjson.WithShowAll(true),
				)

				// skip data sources that have the annotation "cli.default-output-mode"
				// set to "none"
				if m, ok := d.Annotations()["cli.default-output-mode"]; ok && m == "none" {
					continue
				}

				err := d.Subscribe(func(source datasource.DataSource, data datasource.Data) error {
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

func (g *manager) StartGadget(ctx context.Context, name, image string, params map[string]string) (string, error) {
	gadgetCtx := gadgetcontext.New(
		ctx,
		image,
	)

	p := g.runtime.ParamDescs().ToParams()

	newID := make([]byte, 16)
	_, err := rand.Read(newID)
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
	tags := []string{"createdBy=aks-mcp", "gadgetName=" + name}
	if err = p.Set(grpcruntime.ParamTags, strings.Join(tags, ",")); err != nil {
		return "", fmt.Errorf("setting gadget tags: %w", err)
	}
	if err := g.runtime.RunGadget(gadgetCtx, p, params); err != nil {
		return "", fmt.Errorf("running gadget: %w", err)
	}

	return idString, nil
}

func (g *manager) StopGadget(ctx context.Context, id string) error {
	if err := g.runtime.(*grpcruntime.Runtime).RemoveGadgetInstance(ctx, g.runtime.ParamDescs().ToParams(), id); err != nil {
		return fmt.Errorf("stopping to gadget: %w", err)
	}
	return nil
}

func (g *manager) GetResults(ctx context.Context, id string) (string, error) {
	to, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	var results strings.Builder
	gadgetCtx := gadgetcontext.New(
		to,
		id,
		gadgetcontext.WithDataOperators(
			outputDataOperator(func(data []byte) {
				results.Write(data)
				results.WriteByte('\n')
			}),
		),
		gadgetcontext.WithID(id),
		gadgetcontext.WithUseInstance(true),
		gadgetcontext.WithTimeout(time.Second),
	)

	if err := g.runtime.RunGadget(gadgetCtx, g.runtime.ParamDescs().ToParams(), map[string]string{}); err != nil {
		return "", fmt.Errorf("attaching to gadget: %w", err)
	}

	return truncateResults(results.String()), nil
}

func (g *manager) ListGadgets(ctx context.Context) ([]*GadgetInstance, error) {
	instances, err := g.runtime.(*grpcruntime.Runtime).GetGadgetInstances(ctx, g.runtime.ParamDescs().ToParams())
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
		fmt.Fprintf(os.Stderr, "No Inspektor Gadget pods found\n")
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

func (g *manager) Close() error {
	if g.runtime != nil {
		return g.runtime.Close()
	}
	return nil
}
