package main

import (
	tektonapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

const dockerBuildOciTaDescription = `This pipeline is ideal for building container images from a Containerfile while maintaining trust after pipeline customization.

_Uses ` + "`buildah`" + ` to create a container image leveraging [trusted artifacts](https://konflux-ci.dev/architecture/ADR/0036-trusted-artifacts.html). It also optionally creates a source image and runs some build-time tests. Information is shared between tasks using OCI artifacts instead of PVCs. EC will pass the [` + "`trusted_task.trusted`" + `](https://conforma.dev/docs/policy/packages/release_trusted_task.html#trusted_task__trusted) policy as long as all data used to build the artifact is generated from trusted tasks.
This pipeline is pushed as a Tekton bundle to [quay.io](https://quay.io/repository/konflux-ci/tekton-catalog/pipeline-docker-build-oci-ta?tab=tags)_
`

func GenerateDockerBuildOciTa(dockerBuild tektonapi.Pipeline, existing *tektonapi.Pipeline) (tektonapi.Pipeline, error) {
	p := NewPipelineEditor(dockerBuild, existing)

	p.Pipeline.Spec.Description = dockerBuildOciTaDescription
	p.Pipeline.Name = "docker-build-oci-ta"
	p.Pipeline.Labels = map[string]string{
		"pipelines.openshift.io/used-by":  "build-cloud",
		"pipelines.openshift.io/runtime":  "generic",
		"pipelines.openshift.io/strategy": "docker",
	}

	// This pipeline shares data using trusted artifacts instead of the workspace,
	// so drop it along with every task binding to it.
	p.RemovePipelineWorkspace("workspace")

	p.SetTaskRef(
		"clone-repository",
		"git-clone-oci-ta",
		"quay.io/konflux-ci/tekton-catalog/task-git-clone-oci-ta:0.2.6@sha256:590db5c909fa2f3b89cb46e10adb48d278ed7f259d7f613bc2c2b775504c9c11",
	)
	p.AddTaskParam("clone-repository", "ociStorage", StringValue("$(params.output-image).git"))
	p.AddTaskParam("clone-repository", "ociArtifactExpiresAfter", StringValue("$(params.image-expires-after)"))

	p.SetTaskRef(
		"prefetch-dependencies",
		"prefetch-dependencies-oci-ta",
		"quay.io/konflux-ci/tekton-catalog/task-prefetch-dependencies-oci-ta:0.11.0@sha256:6c63bc1ef05e823bb07fe73b4626a82b602ee3e6d3315fc5afa58dd5c532c270",
	)
	p.AddTaskParam("prefetch-dependencies", "SOURCE_ARTIFACT", StringValue("$(tasks.clone-repository.results.SOURCE_ARTIFACT)"))
	p.AddTaskParam("prefetch-dependencies", "ociStorage", StringValue("$(params.output-image).prefetch"))
	p.AddTaskParam("prefetch-dependencies", "ociArtifactExpiresAfter", StringValue("$(params.image-expires-after)"))

	p.SetTaskRef(
		"build-container",
		"buildah-oci-ta",
		"quay.io/konflux-ci/tekton-catalog/task-buildah-oci-ta:0.14.1@sha256:2c44f7f4e3ceba75851814e051f6aa02041072cc95da1d098dc6833a753ab8d8",
	)
	p.AddTaskParam("build-container", "SOURCE_ARTIFACT", StringValue("$(tasks.prefetch-dependencies.results.SOURCE_ARTIFACT)"))
	p.AddTaskParam("build-container", "CACHI2_ARTIFACT", StringValue("$(tasks.prefetch-dependencies.results.CACHI2_ARTIFACT)"))

	p.SetTaskRef(
		"build-source-image",
		"source-build-oci-ta",
		"quay.io/konflux-ci/tekton-catalog/task-source-build-oci-ta:0.3@sha256:6a54d74739332eedf1228f3f37a434f810ec33bbf6db1d311b293a2b770239c5",
	)
	p.AddTaskParam("build-source-image", "SOURCE_ARTIFACT", StringValue("$(tasks.prefetch-dependencies.results.SOURCE_ARTIFACT)"))
	p.AddTaskParam("build-source-image", "CACHI2_ARTIFACT", StringValue("$(tasks.prefetch-dependencies.results.CACHI2_ARTIFACT)"))

	p.SetTaskRef(
		"sast-snyk-check",
		"sast-snyk-check-oci-ta",
		"quay.io/konflux-ci/tekton-catalog/task-sast-snyk-check-oci-ta:0.5@sha256:53443febc3a87ea6d4baacffc70d6f7dbdd1b4e69e0852337661097c73433519",
	)
	p.AddTaskParam("sast-snyk-check", "SOURCE_ARTIFACT", StringValue("$(tasks.prefetch-dependencies.results.SOURCE_ARTIFACT)"))
	p.AddTaskParam("sast-snyk-check", "CACHI2_ARTIFACT", StringValue("$(tasks.prefetch-dependencies.results.CACHI2_ARTIFACT)"))

	p.SetTaskRef(
		"sast-shell-check",
		"sast-shell-check-oci-ta",
		"quay.io/konflux-ci/tekton-catalog/task-sast-shell-check-oci-ta:0.1@sha256:83722286ec3297e3f66b0c220cec8895e0f745337e464c1d1c038ed737cf69fd",
	)
	p.AddTaskParam("sast-shell-check", "SOURCE_ARTIFACT", StringValue("$(tasks.prefetch-dependencies.results.SOURCE_ARTIFACT)"))
	p.AddTaskParam("sast-shell-check", "CACHI2_ARTIFACT", StringValue("$(tasks.prefetch-dependencies.results.CACHI2_ARTIFACT)"))

	p.SetTaskRef(
		"sast-unicode-check",
		"sast-unicode-check-oci-ta",
		"quay.io/konflux-ci/tekton-catalog/task-sast-unicode-check-oci-ta:0.4@sha256:2ee89730856f542d851d99b5d945963583a840c295354dc4257a0052841e35ab",
	)
	p.AddTaskParam("sast-unicode-check", "SOURCE_ARTIFACT", StringValue("$(tasks.prefetch-dependencies.results.SOURCE_ARTIFACT)"))
	p.AddTaskParam("sast-unicode-check", "CACHI2_ARTIFACT", StringValue("$(tasks.prefetch-dependencies.results.CACHI2_ARTIFACT)"))

	p.SetTaskRef(
		"push-dockerfile",
		"push-dockerfile-oci-ta",
		"quay.io/konflux-ci/tekton-catalog/task-push-dockerfile-oci-ta:0.3.1@sha256:c6874d32bad698f23f34d9b3ba833d3960286e03d09dc8e60ae38f54f8babb2c",
	)
	p.AddTaskParam("push-dockerfile", "SOURCE_ARTIFACT", StringValue("$(tasks.prefetch-dependencies.results.SOURCE_ARTIFACT)"))

	p.ReorderArrays()
	return p.Pipeline, nil
}
