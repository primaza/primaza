# Testing Primaza

Primaza has both unit tests and acceptance tests to ensure that we don't break
user scenarios during development.

## Unit tests

To run primaza's unit tests, run:
```bash
make test
```

## Acceptance tests

To run primaza's acceptance tests, run:
```bash
make test-acceptance
```

The test runner will run each of the scenarios under
`test/acceptance/features/` and check if they pass.
Failing scenaios will be reported upon test completion.

## Environment variables

Acceptance tests can be controlled with a few key environment variables.
Their semantics are documented here.

### Provided images

In some circumstances, it may be useful to provide pre-built images for use in
acceptance testing, rather than building images during acceptance tests.
These environment variables control which images are used and how they interact
with testing.

- `PRIMAZA_CONTROLLER_IMAGE_REF`: use the image provided as the primaza controller.
  Defaults to `primaza-controller:latest`
- `PRIMAZA_AGENTAPP_IMAGE_REF`: use the image provided as the application agent controller.
  Defaults to `agentapp:latest`
- `PRIMAZA_AGENTSVC_IMAGE_REF`: use the image provided as the service agent
  controller.  Defaults to `agentsvc:latest`
- `PULL_IMAGES`: due to technical restrictions related to our use of `kind`,
  images need to be in the local docker registry in order to be tested.
  To ensure these images exist locally, set this env var to have the test
  runner pull these images before running test scenarios.

  The default behavior is not to pull images.

### Testing against provided clusters

By default, the acceptance test suite will spin up and down `kind` clusters during the course of testing.
These clusters can be slow to start and stop, which can add up significantly with the number of acceptance tests we have.

As a way to make testing faster, the test suite can use external clusters instead of the ephemeral `kind` clusters.
There are a few environment variables that control how these clusters are provided.

- `CLUSTER_PROVIDER` sets which cluster provider to use.  Currently accepted values are:
    - `kind` - uses ephemeral clusters provided by `kind`
    - `external` - uses persistent clusters using the provided kubeconfigs
- `MAIN_KUBECONFIG` points to a kubeconfig manifest.  The cluster pointed to
  corresponds to the primaza cluster in acceptance tests.
- `WORKER_KUBECONFIG` points to a kubeconfig manifest.  The cluster pointed to
  corresponds to the worker cluster in acceptance tests.
