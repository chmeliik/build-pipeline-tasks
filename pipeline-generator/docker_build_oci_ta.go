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
		"quay.io/konflux-ci/tekton-catalog/task-git-clone-oci-ta:0.2.6@sha256:a3678910c2bf9187a66a96066193cb41046fbb7097160bf2be6a374ed5e49d30",
	)
	p.AddTaskParam("clone-repository", "ociStorage", StringValue("$(params.output-image).git"))
	p.AddTaskParam("clone-repository", "ociArtifactExpiresAfter", StringValue("$(params.image-expires-after)"))

	p.SetTaskRef(
		"prefetch-dependencies",
		"prefetch-dependencies-oci-ta",
		"quay.io/konflux-ci/tekton-catalog/task-prefetch-dependencies-oci-ta:0.10.3@sha256:0c386f2c26b46b978280c5c54caac26bcd524975f8f7cc06bb080ba00eae3ad4",
	)
	p.AddTaskParam("prefetch-dependencies", "SOURCE_ARTIFACT", StringValue("$(tasks.clone-repository.results.SOURCE_ARTIFACT)"))
	p.AddTaskParam("prefetch-dependencies", "ociStorage", StringValue("$(params.output-image).prefetch"))
	p.AddTaskParam("prefetch-dependencies", "ociArtifactExpiresAfter", StringValue("$(params.image-expires-after)"))

	p.SetTaskRef(
		"build-container",
		"buildah-oci-ta",
		"quay.io/konflux-ci/tekton-catalog/task-buildah-oci-ta:0.12.1@sha256:9eef3bf4c8a6a28605c713d56048fef5349bc4c791f289d8b95396ab039eda6b",
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
		"quay.io/konflux-ci/tekton-catalog/task-sast-snyk-check-oci-ta:0.5@sha256:17b9587eba53fabaab35061275306bfe1f99661d55e14e22f62321fa7cb5ff3e",
	)
	p.AddTaskParam("sast-snyk-check", "SOURCE_ARTIFACT", StringValue("$(tasks.prefetch-dependencies.results.SOURCE_ARTIFACT)"))
	p.AddTaskParam("sast-snyk-check", "CACHI2_ARTIFACT", StringValue("$(tasks.prefetch-dependencies.results.CACHI2_ARTIFACT)"))

	p.SetTaskRef(
		"sast-shell-check",
		"sast-shell-check-oci-ta",
		"quay.io/konflux-ci/tekton-catalog/task-sast-shell-check-oci-ta:0.1@sha256:d00c884c78489a52dc638e6ced0a01429670d0754f5b1305ad576709cf660312",
	)
	p.AddTaskParam("sast-shell-check", "SOURCE_ARTIFACT", StringValue("$(tasks.prefetch-dependencies.results.SOURCE_ARTIFACT)"))
	p.AddTaskParam("sast-shell-check", "CACHI2_ARTIFACT", StringValue("$(tasks.prefetch-dependencies.results.CACHI2_ARTIFACT)"))

	p.SetTaskRef(
		"sast-unicode-check",
		"sast-unicode-check-oci-ta",
		"quay.io/konflux-ci/tekton-catalog/task-sast-unicode-check-oci-ta:0.4@sha256:15d654c9576b9c4ee1f78b09fce541acf9afe33f1c2363fd6219a79dbf62c95d",
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
