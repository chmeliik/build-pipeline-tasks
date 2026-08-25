# pipeline-generator

This tool generates most of the pipelines in [`../pipelines`](../pipelines),
using [`../pipelines/docker-build`](../pipelines/docker-build) as the base.

The transformations are simply Go functions that operate on a `tektonapi.Pipeline` object.

## MintMaker compatibility

Importantly, the generation takes into account the existing target pipeline:

- Prefers the resolved bundle references from the existing pipeline
  rather than the ones specified in the code.

On top of that, the [`../hack/generate-pipelines.sh`](../hack/generate-pipelines.sh)
script implements the up-to-date check in such a way that cosmetic-only changes
do not cause CI failures.

The goal is to make the generator get out of the way of MintMaker updates:

- Task bundles can get updates without requiring generator changes.
- Migration scripts can make changes that don't agree stylistically with the generator
  without causing CI failure.

### Caveats

- Even though stylistic differences won't cause CI failures,
  running the generator locally *will* make those changes.
  Revert them or add them as a separate commit when making changes to pipelines.
- Not all migrations will pass without generator changes, manual fixes will be required
  every once in a while.
- Deleting an existing pipeline and generating from scratch would have different results
  than re-running against the existing file:
  - Due to respecting existing task bundle references, running the generator from scratch
    would likely downgrade the bundle references compared to the original state.

> [!NOTE]
> The task bundle reference handling is a workaround for [pmt] limitations.
> If we wanted MintMaker to update the references in the generator code,
> `pmt` would try to run on `*.go` files and crash. If `pmt` fixes this,
> we can add renovate.json configuration to update the refs and simplify the generator.

[pmt]: https://github.com/konflux-ci/pipeline-migration-tool
