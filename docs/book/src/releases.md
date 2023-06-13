# Releases

Primaza is released via [GitHub Releases](https://github.com/primaza/primaza/releases) and [GitHub Container Registry (ghcr)](https://github.com/orgs/primaza/packages?repo_name=primaza).

To create a new release you need to push a Tag respecting the [Semantic Versioning (SemVer) specification](https://semver.org/).

Once a SemVer tag is pushed, [a GitHub Action](https://github.com/primaza/primaza/actions/workflows/release.yaml) will run.
The Action builds and push the Primaza docker images, bakes the manifests, and creates a draft release.

The pushed images are the following:
* [ghcr.io/primaza/primaza](https://github.com/primaza/primaza/pkgs/container/primaza): the Control Plane image
* [ghcr.io/primaza/primaza-agentapp](https://github.com/orgs/primaza/packages/container/package/primaza-agentapp): the Application Agent image
* [ghcr.io/primaza/primaza-agentsvc](https://github.com/orgs/primaza/packages/container/package/primaza-agentsvc): the Service Agent image

The manifests baked are the following:
* `application_namespace_config_<TAG>.yaml`: manifests for configuring an Application Namespace
* `service_namespace_config_<TAG>.yaml`: manifests for configuring a Service Namespace
* `crds_config_<TAG>.yaml`: manifests for installing Primaza's CRDs
* `control_plane_config_<TAG>.yaml`: manifests for installing the Control Plane

The manifests are published as part of the GitHub Release artifacts.
