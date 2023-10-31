package job

import (
	"fmt"

	"github.com/hashicorp/terraform-cdk-go/cdktf"
	"github.com/sourcegraph/managed-services-platform-cdktf/gen/google/cloudrunv2job"

	"github.com/sourcegraph/sourcegraph/dev/managedservicesplatform/internal/stack/cloudrun/internal/builder"
	"github.com/sourcegraph/sourcegraph/lib/pointers"
)

type jobBuilder struct {
	env          []*cloudrunv2job.CloudRunV2JobTemplateTemplateContainersEnv
	volumes      []*cloudrunv2job.CloudRunV2JobTemplateTemplateVolumes
	volumeMounts []*cloudrunv2job.CloudRunV2JobTemplateTemplateContainersVolumeMounts
}

var _ builder.Builder = (*jobBuilder)(nil)

func NewBuilder() builder.Builder {
	return &jobBuilder{}
}

func (b *jobBuilder) AddEnv(key, value string) {
	b.env = append(b.env, &cloudrunv2job.CloudRunV2JobTemplateTemplateContainersEnv{
		Name:  pointers.Ptr(key),
		Value: pointers.Ptr(value),
	})
}

func (b *jobBuilder) AddSecretEnv(key string, secret builder.SecretRef) {
	b.env = append(b.env, &cloudrunv2job.CloudRunV2JobTemplateTemplateContainersEnv{
		Name: pointers.Ptr(key),
		ValueSource: &cloudrunv2job.CloudRunV2JobTemplateTemplateContainersEnvValueSource{
			SecretKeyRef: &cloudrunv2job.CloudRunV2JobTemplateTemplateContainersEnvValueSourceSecretKeyRef{
				Secret:  pointers.Ptr(secret.Name),
				Version: pointers.Ptr(secret.Version),
			},
		},
	})
}

func (b *jobBuilder) AddVolumeMount(name, mountPath string) {
	b.volumeMounts = append(b.volumeMounts, &cloudrunv2job.CloudRunV2JobTemplateTemplateContainersVolumeMounts{
		Name:      pointers.Ptr(name),
		MountPath: pointers.Ptr(mountPath),
	})
}

func (b *jobBuilder) AddSecretVolume(name, mountPath string, secret builder.SecretRef, mode int) {
	b.volumes = append(b.volumes, &cloudrunv2job.CloudRunV2JobTemplateTemplateVolumes{
		Name: pointers.Ptr(name),
		Secret: &cloudrunv2job.CloudRunV2JobTemplateTemplateVolumesSecret{
			Secret: &secret.Name,
			Items: []*cloudrunv2job.CloudRunV2JobTemplateTemplateVolumesSecretItems{{
				Version: pointers.Ptr(secret.Version),
				Mode:    pointers.Ptr(float64(mode)),
				Path:    &mountPath,
			}},
		},
	})
}

func (b *jobBuilder) Build(stack cdktf.TerraformStack, vars builder.Variables) (builder.Resource, error) {
	serviceImageTag, err := vars.Environment.Deploy.ResolveTag(vars.Image)
	if err != nil {
		return nil, err
	}

	var vpcAccess *cloudrunv2job.CloudRunV2JobTemplateTemplateVpcAccess
	if vars.PrivateNetwork != nil {
		vpcAccess = &cloudrunv2job.CloudRunV2JobTemplateTemplateVpcAccess{
			Connector: vars.PrivateNetwork.Connector.SelfLink(),
			Egress:    pointers.Ptr("PRIVATE_RANGES_ONLY"),
		}
	}

	return cloudrunv2job.NewCloudRunV2Job(stack, pointers.Ptr("cloudrun"), &cloudrunv2job.CloudRunV2JobConfig{
		Name:     pointers.Ptr(vars.Service.ID),
		Location: pointers.Ptr(vars.GCPRegion),

		Template: &cloudrunv2job.CloudRunV2JobTemplate{
			TaskCount: pointers.Ptr(float64(1)),

			Template: &cloudrunv2job.CloudRunV2JobTemplateTemplate{
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

				// Configuration for the single service container.
				Containers: []*cloudrunv2job.CloudRunV2JobTemplateTemplateContainers{{
					Name:  pointers.Ptr(vars.Service.ID),
					Image: pointers.Ptr(fmt.Sprintf("%s:%s", vars.Image, serviceImageTag)),

					Resources: &cloudrunv2job.CloudRunV2JobTemplateTemplateContainersResources{
						Limits: &vars.ResourceLimits,
					},

					Ports: []*cloudrunv2job.CloudRunV2JobTemplateTemplateContainersPorts{{
						// ContainerPort is provided to the container as $PORT in Cloud Run
						ContainerPort: pointers.Float64(builder.ServicePort),
						// Name is protocol, supporting 'h2c', 'http1', or nil (http1)
						Name: (*string)(vars.Service.Protocol),
					}},

					Env: b.env,

					VolumeMounts: b.volumeMounts,

					// TODO: Should we do healthchecks with authorization based
					// on MSP convention?
					StartupProbe:  nil,
					LivenessProbe: nil,
				}},

				Volumes: b.volumes,
			},
		}}), nil
}
