# `ec` a command line client for HACBS Enterprise Contract

The `ec` tool is used to evaluate Enterprise Contract policies for Software
Supply Chain. Various sub-commands can be used to assert facts about an artifact
such as:
  * Validating container image signature
  * Validating container image provenance
  * Evaluating Enterprise Contract [policies][pol] over the container image provenance
  * Fetching artifact authorization

Consult the [documentation][docs] for available sub-commands, descriptions and
examples of use.

## Building

Run `make build` from the root directory and use the `dist/ec` executable, or
run `make dist` to build for all supported architectures.

## Testing

Run `make test` to run the unit tests, and `make acceptance` to run the
acceptance tests.

## Linting

Run `make lint` to check for linting issues, and `make lint-fix` to fix linting
issues (formatting, import order, ...).

## Demo

Run `hack/demo.sh` to evaluate the policy against images that have been
built ahead of time.

To regenerate those images, say in case of change in the attestation data, run
`hack/rebuild.sh`.

[pol]: https://github.com/hacbs-contract/ec-policies/
[docs]: https://hacbs-contract.github.io/ec-cli/main/index.html
