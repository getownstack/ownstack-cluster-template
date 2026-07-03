# AGENTS.md

## Scope

These instructions apply to the `ownstack-cluster-template` repository.

This repository is the heart of Ownstack. The SaaS app is only a convenient UI and automation wrapper around this template. This repo defines what a customer actually receives: a self-owned, single-VPS DevOps platform with Kubernetes, ingress, TLS, registry, CI/CD, reusable pipeline logic, and application deployment conventions.

Treat this repository as product infrastructure. Changes here affect every newly provisioned customer environment and may affect reruns against existing environments.

## Product Goal

Ownstack should let a user bring a fresh Ubuntu VPS and a domain, then receive a working DevOps stack that they own:

- k3s Kubernetes on the VPS.
- Traefik as the public ingress layer.
- Let's Encrypt TLS through Cloudflare DNS-01.
- Harbor as the private container registry.
- Jenkins as the CI/CD control plane.
- Jenkins Kubernetes agents for build/deploy jobs.
- A shared Jenkins pipeline library stored in the customer's own cluster repo.
- A shared Helm library used by application charts.
- `dev`, `qa`, and `prod` namespaces prepared for deployments.
- Harbor robot credentials wired into Jenkins and Kubernetes pull secrets.
- A simple application repo convention: a tiny `Jenkinsfile`, a Dockerfile, and a Helm chart.
- Optional pre-deploy hooks for app-specific steps such as database migrations.

The user should own the VPS, cluster, registry, Jenkins instance, and generated cluster repo. Ownstack should not require ongoing hosted control-plane access after setup.

## Architecture Overview

The setup path is:

1. The Ownstack SaaS app creates a customer-owned GitHub repo from this template.
2. The SaaS app SSHes into the customer's VPS.
3. The VPS clones the generated repo into `/root/ownstack-cluster`.
4. The VPS runs `setup_system.sh`.
5. `setup_system.sh` is the stable shell entrypoint. It runs `ownstackctl` from a local binary, configured binary URL, pinned GitHub Release, or local Go source, then falls back to the legacy shell installer only if needed.
6. `ownstackctl apply` validates the environment contract, emits structured progress events, and delegates the current install implementation to `scripts/setup_system_legacy.sh`.
7. The legacy installer installs system prerequisites, configures DNS, applies `system/helmfile.yaml.gotmpl`, creates Kubernetes secrets, and calls `setup_jenkins_harbor.sh`.
8. Helmfile installs Jenkins, Harbor, Traefik, and miscellaneous RBAC.
9. `setup_jenkins_harbor.sh` configures Harbor, Kubernetes namespaces, Jenkins credentials, and the Jenkins GitHub organization folder.
10. Application repos can then use the shared pipeline library to build, push, run optional pre-deploy hooks, and deploy apps.

High-level runtime shape:

```text
Internet
  |
  | HTTPS for *.admin.<domain> and app hostnames
  v
Traefik on k3s
  |
  +--> Jenkins at jenkins.admin.<domain>
  +--> Harbor at harbor.admin.<domain>
  +--> Traefik dashboard at traefik.admin.<domain>
  +--> Deployed customer apps through Traefik IngressRoute

Jenkins
  |
  +--> starts Kubernetes pod agents
  +--> builds Docker images
  +--> pushes to Harbor
  +--> runs optional app pre-deploy hooks
  +--> deploys Helm charts to Kubernetes
```

## Repository Map

- `setup_system.sh`: stable bootstrap entrypoint run on the VPS. It chooses and executes `ownstackctl`.
- `cmd/ownstackctl/`: Go CLI entrypoint. Commands include `plan`, `doctor`, `apply`, and `status`.
- `internal/installer/`: Go installer package for environment validation, stage definitions, structured progress events, and apply orchestration.
- `scripts/setup_system_legacy.sh`: current shell implementation of the installer stages. `ownstackctl apply` delegates here while orchestration migrates into Go.
- `docs/ownstackctl.md`: customer-facing explanation of the CLI and runtime download behavior.
- `.github/workflows/release-ownstackctl.yml`: builds Linux `amd64` and `arm64` `ownstackctl` binaries and checksums on `ownstackctl-v*` tags.
- `setup_jenkins_harbor.sh`: post-Helmfile integration script for Harbor, Jenkins, namespaces, and credentials.
- `system/helmfile.yaml.gotmpl`: declares the core system Helm releases.
- `system/jenkins.gotmpl`: Jenkins Helm values and JCasC.
- `system/harbor.gotmpl`: Harbor Helm values.
- `system/traefik.gotmpl`: Traefik Helm values, dashboard route, ACME DNS-01 configuration.
- `system/misc-manifests/`: small chart for Kubernetes manifests needed by the system, currently Jenkins cluster-admin RBAC.
- `jenkins_pipeline_library/vars/`: Jenkins shared library global steps.
- `common_helm_library/`: Helm library chart for app deployment helpers.
- `installations/`: optional add-ons that are not part of the default bootstrap.

## Core Bootstrap: `setup_system.sh` and `ownstackctl`

`setup_system.sh` is the stable customer-environment bootstrap entrypoint. It must remain parameterized by environment variables. Do not hardcode customer-specific values.

`setup_system.sh` runs the installer in this order:

1. `./bin/ownstackctl`, when a local binary exists.
2. `OWNSTACKCTL_URL`, when an explicit binary URL is provided.
3. A pinned GitHub Release version through `OWNSTACKCTL_VERSION`, defaulting to `ownstackctl-v0.1.0`.
4. `go run ./cmd/ownstackctl`, when Go is installed.
5. `scripts/setup_system_legacy.sh` as a fallback.

Release downloads use `OWNSTACKCTL_REPO`, defaulting to `getownstack/ownstack-cluster-template`, and verify checksums when downloading from versioned releases.

`ownstackctl` is customer-visible and source-controlled in this template. Keep it auditable. The CLI should make the stages and environment contract clear rather than hiding infrastructure behavior behind an opaque binary.

Expected environment contract:

- `git_username`
- `git_user_path`
- `github_pat`
- `system_root_app_repo`
- `system_root_app_path`
- `cloudflare_token`
- `harbor_hostname`
- `harbor_initial_password`
- `harbor_chart_version`
- `traefik_dashboard`
- `traefik_email`
- `traefik_chart_version`
- `jenkins_hostname`
- `jenkins_initial_password`
- `jenkins_chart_version`
- `jenkins_pipeline_library_repo`
- `jenkins_pipeline_library_path`
- `jenkins_github_org_folder_name` optional, default `Repositories`
- `jenkins_github_org_folder_repo_filter` optional, default `*`
- `jenkins_jenkinsfile_path` optional, default `infrastructure/Jenkinsfile`

Current bootstrap responsibilities:

- Validate required environment variables before mutating infrastructure.
- Emit structured progress lines such as `ownstack.event=stage_started id="platform" title="Platform install"` for UI consumers.
- Install k3s with bundled Traefik disabled.
- Configure kubeconfig at `~/.kube/config`.
- Install Docker and base packages.
- Install Helm.
- Install Helmfile.
- Install yq.
- Run `helmfile sync` from `system/`.
- Create the Cloudflare token Kubernetes secret in namespace `traefik`.
- Create a Jenkins-compatible GitHub PAT credential secret in namespace `jenkins`.
- Run `setup_jenkins_harbor.sh`.
- Log elapsed setup duration.

`scripts/setup_system_legacy.sh` owns most implementation details today. Move behavior into Go incrementally, preserving the external environment contract and structured event output.

Current retry behavior uses `retry_command`, which repeats commands until success. This is useful for fresh VPS package/network timing, but be careful when wrapping non-idempotent commands. Prefer idempotent `kubectl apply`, `--dry-run=client -o yaml | kubectl apply -f -`, existence checks, or explicit create-or-skip logic.

## Post-Setup Integration: `setup_jenkins_harbor.sh`

This script wires the installed services together after Helmfile has installed them.

Responsibilities:

- Wait until Harbor reports healthy. Bootstrap-time service checks use curl with `--insecure` because first-run internal checks can see temporary/self-signed TLS before public ACME trust is usable from the VPS.
- Delete existing Harbor projects.
- Create the `product` Harbor project.
- Create a system-level Harbor robot account named `supermario`.
- Store Harbor robot credentials as a Jenkins username/password credential.
- Create Kubernetes namespaces `dev`, `qa`, and `prod`.
- Create image pull secrets named `harbor-robot` in those namespaces.
- Create a Jenkins GitHub Organization Folder.
- Trigger the first Jenkins organization-folder scan.

Be especially cautious here:

- Deleting all existing Harbor projects is destructive. Do not broaden this behavior without explicit product intent.
- `kubectl create namespace` is not currently idempotent. If making reruns cleaner, prefer create-or-apply style.
- Jenkins XML is generated in shell. Preserve escaping for `$` values, XML-sensitive data, and URL encoding.
- Do not log robot account secrets or PATs.

## System Helmfile

`system/helmfile.yaml.gotmpl` installs the core platform:

- `misc-manifests`
- Jenkins from `https://charts.jenkins.io`
- Harbor from `https://helm.goharbor.io`
- Traefik from `https://traefik.github.io/charts`

Values are pulled from required environment variables. Preserve `requiredEnv` for required customer inputs so missing parameters fail clearly.

Release namespaces:

- Jenkins: `jenkins`
- Harbor: `harbor`
- Traefik: `traefik`
- Misc manifests: `default`

## Traefik Layer

Traefik is the ingress and TLS layer for the whole system.

Current behavior in `system/traefik.gotmpl`:

- Enables the dashboard `IngressRoute`.
- Uses `websecure`.
- Enables Kubernetes Gateway provider.
- Allows gateway namespace policy `All`.
- Uses ACME with Cloudflare DNS-01.
- Stores ACME state at `/data/acme.json` on a persistent volume.
- Reads `CF_DNS_API_TOKEN` from the Kubernetes secret `cloudflare-token`.
- Redirects HTTP to HTTPS.

Operational assumptions:

- Cloudflare is authoritative DNS for the domain.
- Cloudflare proxy should be disabled for service subdomains unless the architecture is deliberately changed.
- DNS-01 is used so certificates can be issued without relying on HTTP reachability during initial setup.
- The VPS is expected to expose public HTTP/HTTPS traffic directly.

When changing Traefik:

- Keep certificate issuance reliable on a fresh VPS.
- Avoid requiring cloud load balancers.
- Preserve app compatibility with Traefik `IngressRoute` from `common_helm_library`.
- Test chart rendering where possible before touching live clusters.

## Harbor Layer

Harbor is the private image registry.

Current behavior:

- Exposed at `https://<harbor_hostname>`.
- Uses Traefik ingress annotations and ACME resolver `main`.
- Initial admin password is injected from the environment.
- Trivy and Notary are disabled.
- A `product` project is created post-install.
- A system robot account is created for push/pull.
- Jenkins receives the robot credentials as credential ID `harbor-robot`.
- Runtime namespaces receive matching Docker registry pull secrets.

Pipeline image layout:

```text
<harbor_hostname>/product/<applicationId>:<branch>-<shortSha>
```

When changing Harbor behavior:

- Preserve a low-friction first-run path.
- Keep Jenkins push credentials and Kubernetes pull credentials in sync.
- Avoid printing robot secrets.
- Be clear whether changes affect new installs only or reruns.

## Jenkins Layer

Jenkins is the CI/CD engine.

Current behavior in `system/jenkins.gotmpl`:

- Exposed at `https://<jenkins_hostname>`.
- Admin user is `admin`; password is injected from environment.
- Installs plugins for JCasC, workflows, Kubernetes agents, GitHub, Docker workflow, credentials, Blue Ocean, workspace cleanup, timestamps, pipeline utility steps, and Slack.
- Configures global env vars:
  - `IMAGE_REGISTRY`
  - `COMMON_HELM_LIBRARY_GITHUB_REPO`
- Configures a Kubernetes cloud for pod agents.
- Configures the shared library `jenkins_pipeline_library` from the customer's cluster repo.
- Uses GitHub PAT credential ID `github-pat`.
- Sets Git host key verification to accept first connection.

Post-install, `setup_jenkins_harbor.sh` creates a GitHub Organization Folder:

- Folder name defaults to `Repositories`.
- Repo filter defaults to `*`.
- Jenkinsfile path defaults to `infrastructure/Jenkinsfile`.
- Discovers branches and pull requests.
- Triggers a first scan.

When changing Jenkins:

- Keep the bootstrap fully automated.
- Keep shared library loading from the customer's own cluster repo.
- Keep application repos simple; do not require large Jenkinsfiles.
- Make credential IDs deliberate and documented.
- Be careful with plugin list changes because plugin availability and compatibility can break fresh installs.

## Jenkins Shared Pipeline Library

The shared library lives in `jenkins_pipeline_library/vars/`. These Groovy files are Jenkins global variables callable from a Jenkinsfile.

Current globals:

- `runPipeline(applicationIds)`: orchestrates build, push, and deploy stages. Accepts one string or a list/string array.
- `dockerBuild(applicationId)`: builds `product/<applicationId>:<branch>-<sha>` from `infrastructure/<applicationId>/Dockerfile`.
- `dockerPush(applicationId)`: tags and pushes the image to `IMAGE_REGISTRY` using `harbor-robot`.
- `beforeDeployment(applicationId)`: if `infrastructure/<applicationId>/before-deployment.sh` exists, runs it from the Helm/Kubectl container after image push and before Helm deploy. It passes `APPLICATION_ID`, `RELEASE_NAME`, `NAMESPACE`, `IMAGE`, `IMAGE_REGISTRY`, `IMAGE_REPOSITORY_PROJECT`, and `IMAGE_TAG`.
- `helmDeploy(applicationId)`: checks out the cluster repo, copies `common_helm_library` into the app chart, and deploys with Helm to `prod`.
- `agentPodTemplate()`: defines the Kubernetes pod agent with Docker and Helm containers, mounting the host Docker socket.
- `values()`: shared defaults such as credential ID, container images, and image project name.

Application repo contract:

```text
infrastructure/Jenkinsfile
infrastructure/<applicationId>/Dockerfile
infrastructure/<applicationId>/before-deployment.sh  # optional
infrastructure/<applicationId>/helm/Chart.yaml
infrastructure/<applicationId>/helm/values.yaml
infrastructure/<applicationId>/helm/templates/*.yaml
```

Minimal Jenkinsfile:

```groovy
runPipeline('my-app')
```

Multiple apps:

```groovy
runPipeline(['api', 'worker'])
```

Important current behavior:

- Build and push stages run app IDs in parallel.
- Before-deploy hooks run after push and before Helm deploy.
- Deploy stage runs app IDs sequentially.
- Deploy namespace is currently hardcoded to `prod`.
- Docker builds use the host Docker socket through the agent pod.
- Slack notifications are sent to `#general`.

When changing the pipeline:

- Preserve the tiny Jenkinsfile goal.
- Avoid introducing repo-specific assumptions into the shared library.
- Keep image tag calculation consistent across build, push, and deploy.
- Be explicit if changing namespace behavior, branch promotion, Slack settings, credential IDs, or Docker strategy.
- Consider backwards compatibility for already-generated customer cluster repos.

## Common Helm Library

`common_helm_library` is a Helm library chart. It currently defines:

- `public-url-application`

This helper emits:

- Deployment
- Service
- Traefik `IngressRoute`

Expected values:

- `id`
- `image`
- `url`
- optional `replicaCount`
- optional `pathPrefix`

The reference app uses:

```yaml
{{ include "public-url-application" . }}
```

When changing the helper:

- Preserve simple app charts.
- Keep `imagePullSecrets: harbor-robot` unless the registry model changes.
- Maintain compatibility with Traefik CRDs.
- Be cautious with required values because all app charts may depend on this helper.

## Namespaces and Environments

Current namespaces created by setup:

- `dev`
- `qa`
- `prod`

Current pipeline deployment target:

- `prod`

The existence of `dev` and `qa` signals the intended future model, but promotion/environment selection is not implemented yet. If you implement it, document:

- How branch names map to namespaces.
- Whether PRs deploy.
- How promotion between environments works.
- How secrets differ by namespace.
- How rollbacks are performed.

## Optional Installations

`installations/` contains optional Helmfiles. They are not part of the default bootstrap.

Current add-ons:

- `headlamp`: Kubernetes UI.
- `helm-dashboard`: Helm release dashboard.
- `kubernetes-dashboard`: Kubernetes dashboard with an admin service account.
- `n8n`: workflow automation with example hostnames.
- `postgresql`: CloudNativePG operator plus a single-instance cluster manifest.

When editing optional installations:

- Keep them independently applicable.
- Do not assume they are installed by default.
- Replace placeholder hostnames before presenting them as production-ready.
- Be clear about privileges; dashboard admin accounts are sensitive.

## Security Model

Ownstack intentionally creates a powerful single-tenant DevOps stack. That means security-sensitive defaults must be deliberate.

Current high-privilege areas:

- Jenkins service accounts receive cluster-admin through `misc-manifests`.
- Jenkins agents mount the host Docker socket.
- Jenkins stores GitHub PAT and Harbor robot credentials.
- Harbor robot account has broad project permissions.
- Kubernetes dashboard optional install creates a cluster-admin service account.
- `setup_system.sh`, `ownstackctl apply`, and legacy setup scripts run as root on the VPS.
- Versioned `ownstackctl` release downloads are checksum-verified by the bootstrap entrypoint.

Rules for agents:

- Do not weaken security silently.
- Do not add new broad privileges unless the user asks or the architecture requires it.
- Document privilege expansions in the relevant file and final response.
- Prefer scoped credentials and namespace-scoped access where practical.
- Never print or commit token values, robot secrets, Jenkins passwords, or Cloudflare tokens.

## Idempotency and Reruns

The product should support safe reruns where possible, but not every current operation is idempotent.

Already more idempotent:

- GitHub cluster repo creation is handled by the SaaS app and reuses an existing repo.
- Kubernetes secrets created through `--dry-run=client -o yaml | kubectl apply -f -`.
- Jenkins organization folder creation checks whether the folder already exists.

Risky or non-idempotent:

- Harbor projects are deleted and recreated.
- Namespaces are created with plain `kubectl create namespace`.
- Docker/k3s/package installation is run directly.
- Helmfile sync changes live cluster state.

When improving rerun behavior:

- Prefer create-or-update operations.
- Do not hide destructive behavior under retry loops.
- Make destructive resets explicit and opt-in.
- Keep logs clear enough for a user to understand what changed.

## Validation and Testing

There is no full automated test suite. Do not run live infrastructure commands without explicit user approval.

Safe static checks:

```bash
go test ./...
bash -n setup_system.sh
bash -n scripts/setup_system_legacy.sh
bash -n setup_jenkins_harbor.sh
```

For Helm changes, useful checks may require Helm/Helmfile and chart downloads. Ask before running commands that fetch dependencies or talk to a cluster.

Potential checks, when appropriate and approved:

```bash
cd system
helmfile lint
helmfile template
```

For Groovy shared-library changes, at minimum do a careful static review. Jenkins pipeline syntax often needs a real Jenkins context, so mention any residual validation gap.

For `ownstackctl` release changes, verify local CLI behavior with safe commands such as:

```bash
go run ./cmd/ownstackctl plan
go run ./cmd/ownstackctl doctor
```

`doctor` validates the environment contract; use dummy values only when you are not invoking `apply`.

## Change Guidelines

- Keep customer-specific data out of the repo.
- Keep setup configurable through environment variables.
- Make the desired DevOps architecture visible in code and comments where it matters.
- Keep application developer ergonomics simple.
- Favor Kubernetes-native declarative resources where practical.
- Avoid adding a central hosted dependency that reduces customer ownership.
- Explain migration or rerun implications for changes to existing resources.
- Preserve the repo as a GitHub Template Repository suitable for copying into customer accounts.

## Commands That Require Explicit User Intent

Do not run these just to "test":

```bash
./setup_system.sh
./setup_jenkins_harbor.sh
helmfile sync
kubectl apply ...
kubectl create ...
curl ... GitHub/Harbor/Jenkins/Cloudflare APIs with real credentials
```

These commands mutate local or remote infrastructure, credentials, or external services. Ask first and explain the blast radius.
