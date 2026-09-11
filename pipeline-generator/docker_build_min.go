package main

import (
	tektonapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

const dockerBuildMinDescription = `This pipeline is ideal for building demo container images from a Containerfile while maintaining trust after pipeline customization.

This version of pipeline has minimal resource requests set, it's good for demonstrating and testing.

_Uses ` + "`buildah`" + ` to create a container image leveraging [trusted artifacts](https://konflux-ci.dev/architecture/ADR/0036-trusted-artifacts.html). It also optionally creates a source image and runs some build-time tests. Information is shared between tasks using OCI artifacts instead of PVCs. EC will pass the [` + "`trusted_task.trusted`" + `](https://conforma.dev/docs/policy/packages/release_trusted_task.html#trusted_task__trusted) policy as long as all data used to build the artifact is generated from trusted tasks.
This pipeline is pushed as a Tekton bundle to [quay.io](https://quay.io/repository/konflux-ci/tekton-catalog/pipeline-docker-build-oci-ta?tab=tags)_
`

func GenerateDockerBuildMin(dockerBuildOciTa tektonapi.Pipeline, existing *tektonapi.Pipeline) (tektonapi.Pipeline, error) {
	p := NewPipelineEditor(dockerBuildOciTa, existing)

	p.Pipeline.Spec.Description = dockerBuildMinDescription
	p.Pipeline.Name = "docker-build-oci-ta-min"
	p.Pipeline.Labels = map[string]string{
		"pipelines.openshift.io/used-by":  "build-cloud",
		"pipelines.openshift.io/runtime":  "generic",
		"pipelines.openshift.io/strategy": "docker",
	}

	// Swap the resource-heavy tasks for their minimal variants.
	p.SetTaskRef(
		"clone-repository",
		"git-clone-oci-ta-min",
		"quay.io/konflux-ci/tekton-catalog/task-git-clone-oci-ta-min:0.2.6@sha256:94655465e278077b89d8a1e3f6255166f5c8c0c2acb5622c7f6f60a000e579e0",
	)
	p.SetTaskRef(
		"prefetch-dependencies",
		"prefetch-dependencies-oci-ta-min",
		"quay.io/konflux-ci/tekton-catalog/task-prefetch-dependencies-oci-ta-min:0.10.3@sha256:4e47f8cfd1bc369b911d24bbe9cf4c14726cbcc8cd038ef2e5a29d2a3e5f0bf7",
	)
	p.SetTaskRef(
		"build-container",
		"buildah-oci-ta-min",
		"quay.io/konflux-ci/tekton-catalog/task-buildah-oci-ta-min:0.12.1@sha256:e50f865f46c50a81738bba37372fd7abdba5d83896385a88e6a0a0df37a36086",
	)
	p.SetTaskRef(
		"build-image-index",
		"build-image-index-min",
		"quay.io/konflux-ci/tekton-catalog/task-build-image-index-min:0.3.1@sha256:ef401ea62e96f3496f4e1c7bbd9bcc87c18ecd64ba7e1d53d699b3a9c38ebf83",
	)
	p.SetTaskRef(
		"clamav-scan",
		"clamav-scan-min",
		"quay.io/konflux-ci/tekton-catalog/task-clamav-scan-min:0.3@sha256:622969ce15bd536b4605b98900e4b5378dfde0df536414a40e76df1df75c8635",
	)
	p.SetTaskRef(
		"sast-shell-check",
		"sast-shell-check-oci-ta-min",
		"quay.io/konflux-ci/tekton-catalog/task-sast-shell-check-oci-ta-min:0.1@sha256:e1f2b88cda75910c3d74bc3851a98e51e8fe8807fc62102beb455aada6515c57",
	)
	p.SetTaskRef(
		"sast-unicode-check",
		"sast-unicode-check-oci-ta-min",
		"quay.io/konflux-ci/tekton-catalog/task-sast-unicode-check-oci-ta-min:0.4@sha256:6f592a60e0a3aec0f8330a264b3bcfb559742fd1f3af07b12103e6cbc0e902a5",
	)

	// Add the TPA scan task.
	// Append without a taskRef, then SetTaskRef so its bundle is renovate-preserved like the others.
	p.Pipeline.Spec.Tasks = append(p.Pipeline.Spec.Tasks, tektonapi.PipelineTask{
		Name: "tpa-scan",
		Params: tektonapi.Params{
			{Name: "image-digest", Value: *StringValue("$(tasks.build-image-index.results.IMAGE_DIGEST)")},
			{Name: "image-url", Value: *StringValue("$(tasks.build-image-index.results.IMAGE_URL)")},
		},
		RunAfter: []string{"build-image-index"},
		When: tektonapi.WhenExpressions{
			{Input: "$(params.skip-checks)", Operator: "in", Values: []string{"false"}},
		},
	})
	p.SetTaskRef(
		"tpa-scan",
		"tpa-scan",
		"quay.io/konflux-ci/tekton-catalog/task-tpa-scan:0.1@sha256:3c69ebf5b740229f2cee531889ed885a96b83bc374c2eeaaea662cf4be0823c4",
	)

	// Drop the checks and steps not wanted in the minimal pipeline.
	p.RemoveTask("push-dockerfile")
	p.RemoveTask("apply-tags")
	p.RemoveTask("sast-snyk-check")
	p.RemoveTask("ecosystem-cert-preflight-checks")
	p.RemoveTask("clair-scan")
	p.RemoveTask("build-source-image")

	// build-source-image param is only used by the removed build-source-image task.
	p.RemoveParam("build-source-image")

	p.ReorderArrays()
	return p.Pipeline, nil
}
