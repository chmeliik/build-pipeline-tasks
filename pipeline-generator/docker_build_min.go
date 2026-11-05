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
		"quay.io/konflux-ci/tekton-catalog/task-git-clone-oci-ta-min:0.2.6@sha256:43e6bffbddc284eedd33e818531e880d661bd4eb669d0cd9c5c243181996bb1e",
	)
	p.SetTaskRef(
		"prefetch-dependencies",
		"prefetch-dependencies-oci-ta-min",
		"quay.io/konflux-ci/tekton-catalog/task-prefetch-dependencies-oci-ta-min:0.11.0@sha256:674a4445fef678d0a6fa0f4c0faf6ebdcac50bc7e360a61b4989dd9339e67408",
	)
	p.SetTaskRef(
		"build-container",
		"buildah-oci-ta-min",
		"quay.io/konflux-ci/tekton-catalog/task-buildah-oci-ta-min:0.14.1@sha256:0395d4c5d37f316c98c03ff413b758e87aaf4443e7769e538a1df0b66e766abb",
	)
	p.SetTaskRef(
		"build-image-index",
		"build-image-index-min",
		"quay.io/konflux-ci/tekton-catalog/task-build-image-index-min:0.3.1@sha256:ee8a340c2b82ab4bafdf9125a44d8dd87fe47fccc9599ba6db1695bdc919f56a",
	)
	p.SetTaskRef(
		"clamav-scan",
		"clamav-scan-min",
		"quay.io/konflux-ci/tekton-catalog/task-clamav-scan-min:0.3@sha256:1980e52877dd25435d7d9d40947c80db5cfb2080d73e3bd52a9dbb22f4cb474c",
	)
	p.SetTaskRef(
		"sast-shell-check",
		"sast-shell-check-oci-ta-min",
		"quay.io/konflux-ci/tekton-catalog/task-sast-shell-check-oci-ta-min:0.1@sha256:e48587ffa34224ab113ea62f1237cf3c5f80d97a7ac732ad354f417d17ef9099",
	)
	p.SetTaskRef(
		"sast-unicode-check",
		"sast-unicode-check-oci-ta-min",
		"quay.io/konflux-ci/tekton-catalog/task-sast-unicode-check-oci-ta-min:0.4@sha256:f4c3381a2e86c7e2c46f3556ddf3a829086582c438ecfb78100e37e68395a8a3",
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
		"quay.io/konflux-ci/tekton-catalog/task-tpa-scan:0.1@sha256:97469c1cf2e9d74993bdcc964d8bb1d799bc2ff3e9d72d95379fb24efa587b85",
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
