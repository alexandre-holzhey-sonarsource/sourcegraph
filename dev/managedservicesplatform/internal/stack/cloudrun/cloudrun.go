package cloudrun

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/aws/jsii-runtime-go"
	"github.com/sourcegraph/managed-services-platform-cdktf/gen/google/cloudrunv2service"
	"github.com/sourcegraph/managed-services-platform-cdktf/gen/google/cloudrunv2serviceiammember"
	"github.com/sourcegraph/managed-services-platform-cdktf/gen/google/projectiamcustomrole"

	"github.com/sourcegraph/sourcegraph/dev/managedservicesplatform/googlesecretsmanager"
	"github.com/sourcegraph/sourcegraph/dev/managedservicesplatform/internal/resource/bigquery"
	"github.com/sourcegraph/sourcegraph/dev/managedservicesplatform/internal/resource/cloudflare"
	"github.com/sourcegraph/sourcegraph/dev/managedservicesplatform/internal/resource/cloudflareorigincert"
	"github.com/sourcegraph/sourcegraph/dev/managedservicesplatform/internal/resource/gsmsecret"
	"github.com/sourcegraph/sourcegraph/dev/managedservicesplatform/internal/resource/loadbalancer"
	"github.com/sourcegraph/sourcegraph/dev/managedservicesplatform/internal/resource/managedcert"
	"github.com/sourcegraph/sourcegraph/dev/managedservicesplatform/internal/resource/privatenetwork"
	"github.com/sourcegraph/sourcegraph/dev/managedservicesplatform/internal/resource/random"
	"github.com/sourcegraph/sourcegraph/dev/managedservicesplatform/internal/resource/redis"
	"github.com/sourcegraph/sourcegraph/dev/managedservicesplatform/internal/resource/serviceaccount"
	"github.com/sourcegraph/sourcegraph/dev/managedservicesplatform/internal/resourceid"
	"github.com/sourcegraph/sourcegraph/dev/managedservicesplatform/internal/stack"
	"github.com/sourcegraph/sourcegraph/dev/managedservicesplatform/internal/stack/cloudrun/internal/builder"
	"github.com/sourcegraph/sourcegraph/dev/managedservicesplatform/internal/stack/cloudrun/internal/builder/job"
	"github.com/sourcegraph/sourcegraph/dev/managedservicesplatform/internal/stack/cloudrun/internal/builder/service"
	"github.com/sourcegraph/sourcegraph/dev/managedservicesplatform/internal/stack/options/cloudflareprovider"
	"github.com/sourcegraph/sourcegraph/dev/managedservicesplatform/internal/stack/options/googleprovider"
	"github.com/sourcegraph/sourcegraph/dev/managedservicesplatform/internal/stack/options/randomprovider"
	"github.com/sourcegraph/sourcegraph/dev/managedservicesplatform/spec"
	"github.com/sourcegraph/sourcegraph/lib/errors"
	"github.com/sourcegraph/sourcegraph/lib/pointers"
)

type Output struct{}

type Variables struct {
	ProjectID string

	Service     spec.ServiceSpec
	Image       string
	Environment spec.EnvironmentSpec
}

const StackName = "cloudrun"

// Hardcoded variables.
var (
	// gcpRegion is currently hardcoded.
	gcpRegion = "us-central1"
	// serviceAccountRoles are granted to the service account for the Cloud Run service.
	serviceAccountRoles = []serviceaccount.Role{
		// Allow env vars to source from secrets
		{ID: "role_secret_accessor", Role: "roles/secretmanager.secretAccessor"},
		// Allow service to access private networks
		{ID: "role_compute_networkuser", Role: "roles/compute.networkUser"},
		// Allow service to emit observability
		{ID: "role_cloudtrace_agent", Role: "roles/cloudtrace.agent"},
		{ID: "role_monitoring_metricwriter", Role: "roles/monitoring.metricWriter"},
		// Allow service to publish Cloud Profiler profiles
		{ID: "role_cloudprofiler_agent", Role: "roles/cloudprofiler.agent"},
	}
)

// makeServiceEnvVarPrefix returns the env var prefix for service-specific
// env vars that will be set on the Cloud Run service, i.e.
//
// - ${local.env_var_prefix}_BIGQUERY_PROJECT_ID
// - ${local.env_var_prefix}_BIGQUERY_DATASET
// - ${local.env_var_prefix}_BIGQUERY_TABLE
//
// The prefix is an all-uppercase underscore-delimited version of the service ID,
// for example:
//
//	cody-gateway
//
// The prefix for various env vars will be:
//
//	CODY_GATEWAY_
//
// Note that some variables conforming to conventions like DIAGNOSTICS_SECRET,
// GOOGLE_PROJECT_ID, and REDIS_ENDPOINT do not get prefixed, and custom env
// vars configured on an environment are not automatically prefixed either.
func makeServiceEnvVarPrefix(serviceID string) string {
	return strings.ToUpper(strings.ReplaceAll(serviceID, "-", "_")) + "_"
}

// NewStack instantiates the MSP cloudrun stack, which is currently a pretty
// monolithic stack that encompasses all the core components of an MSP service,
// including networking and dependencies like Redis.
func NewStack(stacks *stack.Set, vars Variables) (*Output, error) {
	stack := stacks.New(StackName,
		googleprovider.With(vars.ProjectID),
		cloudflareprovider.With(gsmsecret.DataConfig{
			Secret:    googlesecretsmanager.SecretCloudflareAPIToken,
			ProjectID: googlesecretsmanager.ProjectID,
		}),
		randomprovider.With())

	// Set up a service-specific env var prefix to avoid conflicts where relevant
	serviceEnvVarPrefix := pointers.Deref(
		vars.Service.EnvVarPrefix,
		makeServiceEnvVarPrefix(vars.Service.ID))

	diagnosticsSecret := random.New(stack, resourceid.New("diagnostics-secret"), random.Config{
		ByteLength: 8,
	})

	id := resourceid.New("cloudrun")

	var customRole projectiamcustomrole.ProjectIamCustomRole
	if vars.Service.IAM != nil && len(vars.Service.IAM.Permissions) > 0 {
		customRole = projectiamcustomrole.NewProjectIamCustomRole(stack, id.ResourceID("custom-role"), &projectiamcustomrole.ProjectIamCustomRoleConfig{
			RoleId:      pointers.Ptr(fmt.Sprintf("%s_custom_role", id.DisplayName())),
			Title:       pointers.Ptr(fmt.Sprintf("%s custom role", id.DisplayName())),
			Project:     &vars.ProjectID,
			Permissions: jsii.Strings(vars.Service.IAM.Permissions...),
		})
	}

	// Set up configuration for the Cloud Run resources
	var cloudRunBuilder builder.Builder
	switch pointers.Deref(vars.Service.Kind, spec.ServiceKindService) {
	case spec.ServiceKindService:
		cloudRunBuilder = service.NewBuilder()
	case spec.ServiceKindJob:
		cloudRunBuilder = job.NewBuilder()
	}

	// Required to enable tracing etc.
	//
	// We don't use serviceEnvVarPrefix here because this is a
	// convention to indicate the environment's project.
	cloudRunBuilder.AddEnv("GOOGLE_CLOUD_PROJECT", vars.ProjectID)

	// Set up secret that service should accept for diagnostics
	// endpoints.
	//
	// We don't use serviceEnvVarPrefix here because this is a
	// convention across MSP services.
	cloudRunBuilder.AddEnv("DIAGNOSTICS_SECRET", diagnosticsSecret.HexValue)

	// Set up build configuration.
	cloudRunBuildVars := builder.Variables{
		Service:     vars.Service,
		Image:       vars.Image,
		Environment: vars.Environment,
		GCPRegion:   gcpRegion,
		ServiceAccount: serviceaccount.New(stack,
			id,
			serviceaccount.Config{
				ProjectID: vars.ProjectID,
				AccountID: fmt.Sprintf("%s-sa", vars.Service.ID),
				DisplayName: fmt.Sprintf("%s Service Account",
					pointers.Deref(vars.Service.Name, vars.Service.ID)),
				Roles: func() []serviceaccount.Role {
					if vars.Service.IAM != nil && len(vars.Service.IAM.Roles) > 0 {
						var rs []serviceaccount.Role
						for _, r := range vars.Service.IAM.Roles {
							rs = append(rs, serviceaccount.Role{
								ID:   matchNonAlphaNumericRegex.ReplaceAllString(r, "_"),
								Role: r,
							})
						}
						serviceAccountRoles = append(rs, serviceAccountRoles...)
					}
					if customRole != nil {
						serviceAccountRoles = append(serviceAccountRoles, serviceaccount.Role{
							ID:   "role_cloudrun_custom_role",
							Role: *customRole.Name(),
						})
					}
					return serviceAccountRoles
				}(),
			}),
		DiagnosticsSecret: diagnosticsSecret,
	}

	if vars.Environment.Resources.NeedsCloudRunConnector() {
		cloudRunBuildVars.PrivateNetwork = privatenetwork.New(stack, privatenetwork.Config{
			ProjectID: vars.ProjectID,
			ServiceID: vars.Service.ID,
			Region:    gcpRegion,
		})
	}

	// redisInstance is only created and non-nil if Redis is configured for the
	// environment.
	if vars.Environment.Resources != nil && vars.Environment.Resources.Redis != nil {
		redisInstance, err := redis.New(stack,
			resourceid.New("redis"),
			redis.Config{
				ProjectID: vars.ProjectID,
				Network:   cloudRunBuildVars.PrivateNetwork.Network,
				Region:    gcpRegion,
				Spec:      *vars.Environment.Resources.Redis,
			})
		if err != nil {
			return nil, errors.Wrap(err, "failed to render Redis instance")
		}
		// We don't use serviceEnvVarPrefix here because this is a
		// Sourcegraph-wide convention.
		cloudRunBuilder.AddEnv("REDIS_ENDPOINT", redisInstance.Endpoint)

		caCertVolumeName := "redis-ca-cert"
		// TODO
		cloudRun.AdditionalVolumes = append(cloudRun.AdditionalVolumes,
			&cloudrunv2service.CloudRunV2ServiceTemplateVolumes{
				Name: pointers.Ptr(caCertVolumeName),
				Secret: &cloudrunv2service.CloudRunV2ServiceTemplateVolumesSecret{
					Secret: &redisInstance.Certificate.ID,
					Items: []*cloudrunv2service.CloudRunV2ServiceTemplateVolumesSecretItems{{
						Version: &redisInstance.Certificate.Version,
						Path:    pointers.Ptr("redis-ca-cert.pem"),
						Mode:    pointers.Float64(292), // 0444 read-only
					}},
				},
			})
		cloudRun.AdditionalVolumeMounts = append(cloudRun.AdditionalVolumeMounts,
			&cloudrunv2service.CloudRunV2ServiceTemplateContainersVolumeMounts{
				Name: pointers.Ptr(caCertVolumeName),
				// TODO: Use subpath if google_cloud_run_v2_service adds support for it:
				// https://registry.terraform.io/providers/hashicorp/google-beta/latest/docs/resources/cloud_run_v2_service#mount_path
				MountPath: pointers.Ptr("/etc/ssl/custom-certs"),
			})
	}

	// bigqueryDataset is only created and non-nil if BigQuery is configured for
	// the environment.
	if vars.Environment.Resources != nil && vars.Environment.Resources.BigQueryTable != nil {
		bigqueryDataset, err := bigquery.New(stack, resourceid.New("bigquery"), bigquery.Config{
			DefaultProjectID: vars.ProjectID,
			Spec:             *vars.Environment.Resources.BigQueryTable,
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to render BigQuery dataset")
		}
		cloudRun.AdditionalEnv = append(cloudRun.AdditionalEnv,
			&cloudrunv2service.CloudRunV2ServiceTemplateContainersEnv{
				Name:  pointers.Ptr(serviceEnvVarPrefix + "BIGQUERY_PROJECT_ID"),
				Value: pointers.Ptr(bigqueryDataset.ProjectID),
			}, &cloudrunv2service.CloudRunV2ServiceTemplateContainersEnv{
				Name:  pointers.Ptr(serviceEnvVarPrefix + "BIGQUERY_DATASET"),
				Value: pointers.Ptr(bigqueryDataset.Dataset),
			}, &cloudrunv2service.CloudRunV2ServiceTemplateContainersEnv{
				Name:  pointers.Ptr(serviceEnvVarPrefix + "BIGQUERY_TABLE"),
				Value: pointers.Ptr(bigqueryDataset.Table),
			})
	}

	// Finally, create the Cloud Run service with the finalized service
	// configuration
	service, err := cloudRun.Build(stack, vars)
	if err != nil {
		return nil, err
	}

	// Allow IAM-free access to the service - auth should be handled generally
	// by the service itself.
	//
	// TODO: Parameterize this so internal services can choose to auth only via
	// GCP IAM?
	_ = cloudrunv2serviceiammember.NewCloudRunV2ServiceIamMember(stack, pointers.Ptr("cloudrun-allusers-runinvoker"), &cloudrunv2serviceiammember.CloudRunV2ServiceIamMemberConfig{
		Name:     service.Name(),
		Location: service.Location(),
		Project:  &vars.ProjectID,
		Member:   pointers.Ptr("allUsers"),
		Role:     pointers.Ptr("roles/run.invoker"),
	})

	// Then whatever the user requested to expose the service publicly
	switch domain := vars.Environment.Domain; domain.Type {
	case "", spec.EnvironmentDomainTypeNone:
		// do nothing

	case spec.EnvironmentDomainTypeCloudflare:
		// set zero value for convenience
		if domain.Cloudflare == nil {
			return nil, errors.Newf("domain type %q specified but Cloudflare configuration is nil",
				domain.Type)
		}
		if domain.Cloudflare.Subdomain == "" || domain.Cloudflare.Zone == "" {
			return nil, errors.Newf("domain type %q requires 'cloudflare.subdomain' and 'cloudflare.zone' to be set",
				domain.Type)
		}

		// Provision SSL cert
		var sslCertificate loadbalancer.SSLCertificate
		if domain.Cloudflare.Proxied {
			sslCertificate = cloudflareorigincert.New(stack,
				resourceid.New("cf-origin-cert"),
				cloudflareorigincert.Config{
					ProjectID: vars.ProjectID,
				}).Certificate
		} else {
			sslCertificate = managedcert.New(stack,
				resourceid.New("managed-cert"),
				managedcert.Config{
					ProjectID: vars.ProjectID,
					Domain:    fmt.Sprintf("%s.%s", domain.Cloudflare.Subdomain, domain.Cloudflare.Zone),
				}).Certificate
		}

		// Create load-balancer pointing to Cloud Run service
		lb, err := loadbalancer.New(stack, resourceid.New("loadbalancer"), loadbalancer.Config{
			ProjectID:      vars.ProjectID,
			Region:         gcpRegion,
			TargetService:  service,
			SSLCertificate: sslCertificate,
		})
		if err != nil {
			return nil, errors.Wrap(err, "loadbalancer.New")
		}

		// Now set up a DNS record in Cloudflare to route to the load balancer
		if _, err := cloudflare.New(stack, resourceid.New("cf"), cloudflare.Config{
			Spec:   *vars.Environment.Domain.Cloudflare,
			Target: *lb,
		}); err != nil {
			return nil, err
		}
	}

	return &Output{}, nil
}

var (
	matchNonAlphaNumericRegex = regexp.MustCompile("[^a-zA-Z0-9]+")
)
