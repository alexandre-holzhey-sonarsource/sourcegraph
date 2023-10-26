package builder

import (
	"github.com/hashicorp/terraform-cdk-go/cdktf"

	"github.com/sourcegraph/sourcegraph/dev/managedservicesplatform/internal/resource/privatenetwork"
	"github.com/sourcegraph/sourcegraph/dev/managedservicesplatform/internal/resource/random"
	"github.com/sourcegraph/sourcegraph/dev/managedservicesplatform/internal/resource/serviceaccount"
	"github.com/sourcegraph/sourcegraph/dev/managedservicesplatform/spec"
)

type Variables struct {
	Service     spec.ServiceSpec
	Image       string
	Environment spec.EnvironmentSpec

	// GCPProjectID for all resources.
	GCPProjectID string
	// GCPRegion for all resources.
	GCPRegion string
	// ServiceAccount for the Cloud Run resource
	ServiceAccount *serviceaccount.Output
	// DiagnosticsSecret is the secret for healthcheck or diagnostics endpoints
	DiagnosticsSecret *random.Output
	// PrivateNetwork is configured if required as an internal network for the
	// Cloud Run resource to talk to other GCP resources.
	PrivateNetwork *privatenetwork.Output
}

type Builder interface {
	// AddEnv adds an environment variable to the resource, and should only be
	// used before Build.
	AddEnv(key, value string)
	// AddSecretEnv adds an environment variable to the resource, and should only
	// be used before Build.
	AddSecretEnv(key, secretName string)
	// AddVolumeMount adds a volume mount to the resource, and should only be
	// used before Build.
	AddVolumeMount(mountPath, name string)

	// Build finalizes the resource.
	Build(cdktf.TerraformStack, Variables) (cdktf.TerraformResource, error)
}

const (
	// ServicePort is provided to the container as $PORT in Cloud Run:
	// https://cloud.google.com/run/docs/configuring/services/containers#configure-port
	ServicePort = 9992
	// HealthCheckEndpoint is the default healthcheck endpoint for all services.
	HealthCheckEndpoint = "/-/healthz"
	// DefaultMaxInstances is the default Scaling.MaxCount
	DefaultMaxInstances = 5
	// DefaultMaxConcurrentRequests is the default Scaling.MaxRequestConcurrency
	// It is set very high to prefer fewer instances, as Go services can generally
	// handle very high load without issue.
	DefaultMaxConcurrentRequests = 1000
)
