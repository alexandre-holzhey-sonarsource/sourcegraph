package service

import (
	"fmt"

	"github.com/hashicorp/terraform-cdk-go/cdktf"
	"github.com/sourcegraph/managed-services-platform-cdktf/gen/google/cloudrunv2service"

	"github.com/sourcegraph/sourcegraph/dev/managedservicesplatform/internal/stack/cloudrun/internal/builder"
	"github.com/sourcegraph/sourcegraph/dev/managedservicesplatform/spec"
	"github.com/sourcegraph/sourcegraph/lib/pointers"
)

// builder parameterizes configurable components of the core
// Cloud Run Service. It's particularly useful for strongly typing fields that
// the generated CDKTF library accepts as interface{} types.
type serviceBuilder struct {
	additionalEnv          []*cloudrunv2service.CloudRunV2ServiceTemplateContainersEnv
	additionalVolumes      []*cloudrunv2service.CloudRunV2ServiceTemplateVolumes
	additionalVolumeMounts []*cloudrunv2service.CloudRunV2ServiceTemplateContainersVolumeMounts
}

var _ builder.Builder = (*serviceBuilder)(nil)

func NewBuilder() builder.Builder {
	return &serviceBuilder{}
}

func (c *serviceBuilder) AddEnv(key, value string) {
	c.additionalEnv = append(c.additionalEnv, &cloudrunv2service.CloudRunV2ServiceTemplateContainersEnv{
		Name:  pointers.Ptr(key),
		Value: pointers.Ptr(value),
	})
}

func (c *serviceBuilder) AddSecretEnv(key, secretName string) {
	c.additionalEnv = append(c.additionalEnv, &cloudrunv2service.CloudRunV2ServiceTemplateContainersEnv{
		Name: pointers.Ptr(key),
		ValueSource: &cloudrunv2service.CloudRunV2ServiceTemplateContainersEnvValueSource{
			SecretKeyRef: &cloudrunv2service.CloudRunV2ServiceTemplateContainersEnvValueSourceSecretKeyRef{
				Secret:  pointers.Ptr(secretName),
				Version: pointers.Ptr("latest"),
			},
		},
	})
}

func (c *serviceBuilder) AddVolumeMount(name, mountPath string) {
	c.additionalVolumeMounts = append(c.additionalVolumeMounts, &cloudrunv2service.CloudRunV2ServiceTemplateContainersVolumeMounts{
		MountPath: pointers.Ptr(mountPath),
		Name:      pointers.Ptr(name),
	})
}

func (c *serviceBuilder) Build(stack cdktf.TerraformStack, vars builder.Variables) (cdktf.TerraformResource, error) {
	// TODO Make this fancier, for now this is just a sketch of maybe CD?
	serviceImageTag, err := vars.Environment.Deploy.ResolveTag()
	if err != nil {
		return nil, err
	}

	var vpcAccess *cloudrunv2service.CloudRunV2ServiceTemplateVpcAccess
	if vars.PrivateNetwork != nil {
		vpcAccess = &cloudrunv2service.CloudRunV2ServiceTemplateVpcAccess{
			Connector: vars.PrivateNetwork.Connector.SelfLink(),
			Egress:    pointers.Ptr("PRIVATE_RANGES_ONLY"),
		}
	}

	containerEnvVars, err := makeContainerEnvVars(
		vars.Environment.Env,
		vars.Environment.SecretEnv,
		envVariablesData{
			ProjectID:      vars.GCPProjectID,
			ServiceDnsName: fmt.Sprintf("%s.%s", vars.Environment.Domain.Cloudflare.Subdomain, vars.Environment.Domain.Cloudflare.Zone),
		},
	)

	return cloudrunv2service.NewCloudRunV2Service(stack, pointers.Ptr("cloudrun"), &cloudrunv2service.CloudRunV2ServiceConfig{
		Name:     pointers.Ptr(vars.Service.ID),
		Location: pointers.Ptr(vars.GCPRegion),

		//  Disallows direct traffic from public internet, we have a LB set up for that.
		Ingress: pointers.Ptr("INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER"),

		Template: &cloudrunv2service.CloudRunV2ServiceTemplate{
			// Act under our provisioned service account
			ServiceAccount: pointers.Ptr(vars.ServiceAccount.Email),

			// Connect to VPC connector for talking to other GCP services.
			VpcAccess: vpcAccess,

			// Set a high limit that matches our default Cloudflare zone's
			// timeout:
			//
			//   export CF_API_TOKEN=$(gcloud secrets versions access latest --secret CLOUDFLARE_API_TOKEN --project sourcegraph-secrets)
			//   curl -H "Authorization: Bearer $CF_API_TOKEN" https://api.cloudflare.com/client/v4/zones | jq '.result[]  | select(.name == "sourcegraph.com") | .id'
			//   curl -H "Authorization: Bearer $CF_API_TOKEN" https://api.cloudflare.com/client/v4/zones/$CF_ZONE_ID/settings | jq '.result[] | select(.id == "proxy_read_timeout")'
			//
			// Result should be something like:
			//
			//   {
			//     "id": "proxy_read_timeout",
			//     "value": "300",
			//     "modified_on": "2022-02-08T23:10:35.772888Z",
			//     "editable": true
			//   }
			//
			// The service should implement tighter timeouts on its own if desired.
			Timeout: pointers.Ptr("300s"),

			// Scaling configuration
			MaxInstanceRequestConcurrency: pointers.Float64(
				pointers.Deref(vars.Environment.Instances.Scaling.MaxRequestConcurrency, builder.DefaultMaxConcurrentRequests)),
			Scaling: &cloudrunv2service.CloudRunV2ServiceTemplateScaling{
				MinInstanceCount: pointers.Float64(vars.Environment.Instances.Scaling.MinCount),
				MaxInstanceCount: pointers.Float64(
					pointers.Deref(vars.Environment.Instances.Scaling.MaxCount, builder.DefaultMaxInstances)),
			},

			// Configuration for the single service container.
			Containers: []*cloudrunv2service.CloudRunV2ServiceTemplateContainers{{
				Name:  pointers.Ptr(vars.Service.ID),
				Image: pointers.Ptr(fmt.Sprintf("%s:%s", vars.Image, serviceImageTag)),

				Resources: &cloudrunv2service.CloudRunV2ServiceTemplateContainersResources{
					Limits: makeContainerResourceLimits(vars.Environment.Instances.Resources),
				},

				Ports: []*cloudrunv2service.CloudRunV2ServiceTemplateContainersPorts{{
					// ContainerPort is provided to the container as $PORT in Cloud Run
					ContainerPort: pointers.Float64(builder.ServicePort),
					// Name is protocol, supporting 'h2c', 'http1', or nil (http1)
					Name: (*string)(vars.Service.Protocol),
				}},

				Env: append(
					containerEnvVars,
					c.additionalEnv...),

				// Do healthchecks with authorization based on MSP convention.
				StartupProbe: func() *cloudrunv2service.CloudRunV2ServiceTemplateContainersStartupProbe {
					// Default: enabled
					if vars.Environment.StatupProbe != nil &&
						pointers.Deref(vars.Environment.StatupProbe.Disabled, false) {
						return nil
					}

					// Set zero value for ease of reference
					if vars.Environment.StatupProbe == nil {
						vars.Environment.StatupProbe = &spec.EnvironmentStartupProbeSpec{}
					}

					return &cloudrunv2service.CloudRunV2ServiceTemplateContainersStartupProbe{
						HttpGet: &cloudrunv2service.CloudRunV2ServiceTemplateContainersStartupProbeHttpGet{
							Path: pointers.Ptr(builder.HealthCheckEndpoint),
							HttpHeaders: []*cloudrunv2service.CloudRunV2ServiceTemplateContainersStartupProbeHttpGetHttpHeaders{{
								Name:  pointers.Ptr("Authorization"),
								Value: pointers.Ptr(fmt.Sprintf("Bearer %s", vars.DiagnosticsSecret.HexValue)),
							}},
						},
						InitialDelaySeconds: pointers.Float64(0),
						TimeoutSeconds:      pointers.Float64(pointers.Deref(vars.Environment.StatupProbe.Timeout, 1)),
						PeriodSeconds:       pointers.Float64(pointers.Deref(vars.Environment.StatupProbe.Interval, 1)),
						FailureThreshold:    pointers.Float64(3),
					}
				}(),
				LivenessProbe: func() *cloudrunv2service.CloudRunV2ServiceTemplateContainersLivenessProbe {
					// Default: disabled
					if vars.Environment.LivenessProbe == nil {
						return nil
					}
					return &cloudrunv2service.CloudRunV2ServiceTemplateContainersLivenessProbe{
						HttpGet: &cloudrunv2service.CloudRunV2ServiceTemplateContainersLivenessProbeHttpGet{
							Path: pointers.Ptr(builder.HealthCheckEndpoint),
							HttpHeaders: []*cloudrunv2service.CloudRunV2ServiceTemplateContainersLivenessProbeHttpGetHttpHeaders{{
								Name:  pointers.Ptr("Authorization"),
								Value: pointers.Ptr(fmt.Sprintf("Bearer %s", vars.DiagnosticsSecret.HexValue)),
							}},
						},
						TimeoutSeconds:   pointers.Float64(pointers.Deref(vars.Environment.LivenessProbe.Timeout, 1)),
						PeriodSeconds:    pointers.Float64(pointers.Deref(vars.Environment.LivenessProbe.Interval, 1)),
						FailureThreshold: pointers.Float64(2),
					}
				}(),

				VolumeMounts: c.additionalVolumeMounts,
			}},

			Volumes: c.additionalVolumes,
		}}), nil
}
