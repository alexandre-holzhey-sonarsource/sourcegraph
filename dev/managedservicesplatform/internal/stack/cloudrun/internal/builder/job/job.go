package job

import (
	"github.com/sourcegraph/managed-services-platform-cdktf/gen/google/cloudrunv2job"

	"github.com/sourcegraph/sourcegraph/dev/managedservicesplatform/internal/stack/cloudrun/internal/builder"
)

type jobBuilder struct {
	additionalEnv          []*cloudrunv2job.CloudRunV2JobTemplateTemplateContainersEnv
	additionalVolumes      []*cloudrunv2job.CloudRunV2JobTemplateTemplateVolumes
	additionalVolumeMounts []*cloudrunv2job.CloudRunV2JobTemplateTemplateContainersVolumeMounts
}

func NewBuilder() builder.Builder {
	return &jobBuilder{}
}
