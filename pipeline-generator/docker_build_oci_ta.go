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
		"quay.io/konflux-ci/tekton-catalog/task-git-clone-oci-ta:0.2.6@sha256:1d7ba568ae6e9e26800054f3942779de8abc80ddc48f0e44df6b1c2f098161dc",
	)
	p.AddTaskParam("clone-repository", "ociStorage", StringValue("$(params.output-image).git"))
	p.AddTaskParam("clone-repository", "ociArtifactExpiresAfter", StringValue("$(params.image-expires-after)"))

	p.SetTaskRef(
		"prefetch-dependencies",
		"prefetch-dependencies-oci-ta",
		"quay.io/konflux-ci/tekton-catalog/task-prefetch-dependencies-oci-ta:0.10.3@sha256:9fd7d251f92250d89a5465756e278e053af28c6356e5ccdedc4be87bbb13c190",
	)
	p.AddTaskParam("prefetch-dependencies", "SOURCE_ARTIFACT", StringValue("$(tasks.clone-repository.results.SOURCE_ARTIFACT)"))
	p.AddTaskParam("prefetch-dependencies", "ociStorage", StringValue("$(params.output-image).prefetch"))
	p.AddTaskParam("prefetch-dependencies", "ociArtifactExpiresAfter", StringValue("$(params.image-expires-after)"))

	p.SetTaskRef(
		"build-container",
		"buildah-oci-ta",
		"quay.io/konflux-ci/tekton-catalog/task-buildah-oci-ta:0.12.1@sha256:dd0c817c0e3ded8ff52b6c227681ea542e17211bf85de38101b9b11029d16bac",
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
		"quay.io/konflux-ci/tekton-catalog/task-sast-snyk-check-oci-ta:0.5@sha256:a973b32f0e11958aba08a635981e4d4e7eb200c417492371a1e692ff589b4e8f",
	)
	p.AddTaskParam("sast-snyk-check", "SOURCE_ARTIFACT", StringValue("$(tasks.prefetch-dependencies.results.SOURCE_ARTIFACT)"))
	p.AddTaskParam("sast-snyk-check", "CACHI2_ARTIFACT", StringValue("$(tasks.prefetch-dependencies.results.CACHI2_ARTIFACT)"))

	p.SetTaskRef(
		"sast-shell-check",
		"sast-shell-check-oci-ta",
		"quay.io/konflux-ci/tekton-catalog/task-sast-shell-check-oci-ta:0.1@sha256:d9b01530ce3c20287714e64980f154e28c0e94d17e2dbfbaf3c8bf77b1844b9e",
	)
	p.AddTaskParam("sast-shell-check", "SOURCE_ARTIFACT", StringValue("$(tasks.prefetch-dependencies.results.SOURCE_ARTIFACT)"))
	p.AddTaskParam("sast-shell-check", "CACHI2_ARTIFACT", StringValue("$(tasks.prefetch-dependencies.results.CACHI2_ARTIFACT)"))

	p.SetTaskRef(
		"sast-unicode-check",
		"sast-unicode-check-oci-ta",
		"quay.io/konflux-ci/tekton-catalog/task-sast-unicode-check-oci-ta:0.4@sha256:381750451fbcc86d90fa83579a48e0e6555d7cf374fe2c4af3f9262a17fc3c89",
	)
	p.AddTaskParam("sast-unicode-check", "SOURCE_ARTIFACT", StringValue("$(tasks.prefetch-dependencies.results.SOURCE_ARTIFACT)"))
	p.AddTaskParam("sast-unicode-check", "CACHI2_ARTIFACT", StringValue("$(tasks.prefetch-dependencies.results.CACHI2_ARTIFACT)"))

	p.SetTaskRef(
		"push-dockerfile",
		"push-dockerfile-oci-ta",
		"quay.io/konflux-ci/tekton-catalog/task-push-dockerfile-oci-ta:0.3.1@sha256:3e59d6303ca031c27fab079f7c743fbdecc53d007b8d86e32227de5164f06e31",
	)
	p.AddTaskParam("push-dockerfile", "SOURCE_ARTIFACT", StringValue("$(tasks.prefetch-dependencies.results.SOURCE_ARTIFACT)"))

	p.ReorderArrays()
	return p.Pipeline, nil
}
