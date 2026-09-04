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
		"quay.io/konflux-ci/tekton-catalog/task-git-clone-oci-ta:0.2.6@sha256:2e8fe30b6d5c8a8a3e6bbc0ea5a55e05b6170d4a399830f25ea43e17881ce544",
	)
	p.AddTaskParam("clone-repository", "ociStorage", StringValue("$(params.output-image).git"))
	p.AddTaskParam("clone-repository", "ociArtifactExpiresAfter", StringValue("$(params.image-expires-after)"))

	p.SetTaskRef(
		"prefetch-dependencies",
		"prefetch-dependencies-oci-ta",
		"quay.io/konflux-ci/tekton-catalog/task-prefetch-dependencies-oci-ta:0.10.2@sha256:374f776bcb2048c3adeaf4dbb460c52d001bc6020320df5e878c2d05391302da",
	)
	p.AddTaskParam("prefetch-dependencies", "SOURCE_ARTIFACT", StringValue("$(tasks.clone-repository.results.SOURCE_ARTIFACT)"))
	p.AddTaskParam("prefetch-dependencies", "ociStorage", StringValue("$(params.output-image).prefetch"))
	p.AddTaskParam("prefetch-dependencies", "ociArtifactExpiresAfter", StringValue("$(params.image-expires-after)"))

	p.SetTaskRef(
		"build-container",
		"buildah-oci-ta",
		"quay.io/konflux-ci/tekton-catalog/task-buildah-oci-ta:0.12.1@sha256:d44fb2d0e1bb5eb916777bf129b9d6b5e8c08a14f6c4505dc67bce22660b3d9f",
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
		"quay.io/konflux-ci/tekton-catalog/task-sast-snyk-check-oci-ta:0.5@sha256:67a409de3c99aeaee4596da3081f26955ca6201a7f12cb6a9912659bdbcc4d01",
	)
	p.AddTaskParam("sast-snyk-check", "SOURCE_ARTIFACT", StringValue("$(tasks.prefetch-dependencies.results.SOURCE_ARTIFACT)"))
	p.AddTaskParam("sast-snyk-check", "CACHI2_ARTIFACT", StringValue("$(tasks.prefetch-dependencies.results.CACHI2_ARTIFACT)"))

	p.SetTaskRef(
		"sast-shell-check",
		"sast-shell-check-oci-ta",
		"quay.io/konflux-ci/tekton-catalog/task-sast-shell-check-oci-ta:0.1@sha256:afa8ba8859739e48b672f66fa2af357d27f4d96846a7c0ad84e38f21b043f695",
	)
	p.AddTaskParam("sast-shell-check", "SOURCE_ARTIFACT", StringValue("$(tasks.prefetch-dependencies.results.SOURCE_ARTIFACT)"))
	p.AddTaskParam("sast-shell-check", "CACHI2_ARTIFACT", StringValue("$(tasks.prefetch-dependencies.results.CACHI2_ARTIFACT)"))

	p.SetTaskRef(
		"sast-unicode-check",
		"sast-unicode-check-oci-ta",
		"quay.io/konflux-ci/tekton-catalog/task-sast-unicode-check-oci-ta:0.4@sha256:69d5fca2fb94dcc7df32e36e4828e6fb24b9ba55b1837c331a83e20d8dfd479e",
	)
	p.AddTaskParam("sast-unicode-check", "SOURCE_ARTIFACT", StringValue("$(tasks.prefetch-dependencies.results.SOURCE_ARTIFACT)"))
	p.AddTaskParam("sast-unicode-check", "CACHI2_ARTIFACT", StringValue("$(tasks.prefetch-dependencies.results.CACHI2_ARTIFACT)"))

	p.SetTaskRef(
		"push-dockerfile",
		"push-dockerfile-oci-ta",
		"quay.io/konflux-ci/tekton-catalog/task-push-dockerfile-oci-ta:0.3.1@sha256:ef00a86cb22259fcfdefa15a5116b63d0f24ee35c95d05ff9815ee8f84beb548",
	)
	p.AddTaskParam("push-dockerfile", "SOURCE_ARTIFACT", StringValue("$(tasks.prefetch-dependencies.results.SOURCE_ARTIFACT)"))

	p.ReorderArrays()
	return p.Pipeline, nil
}
