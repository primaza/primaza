# Testing Primaza

Primaza has both unit tests and acceptance tests to ensure that we don't break
user scenarios during development.

## Unit tests

To run primaza's unit tests, run:
```bash
make test
```

## Acceptance tests

You'll need a recent copy of `kind` to run acceptance tests.  In the future, we
will likely lift this restriction and allow users to test against provided
clusters.

To run primaza's acceptance tests, run:
```bash
make test-acceptance
```

The test runner will run each of the scenarios under
`test/acceptance/features/` and check if they pass.  Failing scenaios will be
reported upon test completion.

### Environment variables

Acceptance tests can be controlled with a few key environment variables:

- `PRIMAZA_CONTROLLER_IMAGE_REF`: use the image provided as the primaza controller.  Defaults to `primaza-controller:latest`
- `PRIMAZA_AGENTAPP_IMAGE_REF`: use the image provided as the application agent controller.  Defaults to `agentapp:latest`
- `PRIMAZA_AGENTSVC_IMAGE_REF`: use the image provided as the service agent controller.  Defaults to `agentsvc:latest`
- `PULL_IMAGES`: due to technical restrictions related to our use of `kind`,
  images need to be in the local docker registry in order to be tested.  To
  ensure these images exist locally, set this env var to have the test runner
  pull these images before running test scenarios.

  The default behavior is not to pull images.
