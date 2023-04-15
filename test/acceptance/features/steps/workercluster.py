import base64
import os
import polling2
import tempfile
import yaml
from behave import given, step
from typing import Dict
from kubernetes import client
from kubernetes.client.rest import ApiException
from steps.command import Command
from steps.cluster import Cluster


class WorkerCluster(Cluster):
    """
    Base class for instances of Worker clusters.
    Implements functionalities for configuration of kubernetes clusters that will act as Primaza workers,
    like Service Account management, and Primaza CRD installation.
    """

    def __init__(self, cluster_provisioner: str, cluster_name: str):
        super().__init__(cluster_provisioner, cluster_name)

    def start(self):
        super().start()

    def install_primaza_crd(self):
        """
        Installs Primaza's CRD into the cluster
        """
        self.__install_crd(component="primaza")

    def install_agentapp_crd(self):
        """
        Installs Application Agent's CRD into the cluster
        """
        self.__install_crd(component="agentapp")

    def install_agentsvc_crd(self):
        """
        Installs Service Agent's CRD into the cluster
        """
        self.__install_crd(component="agentsvc")

    def __install_crd(self, component: str):
        """
        Installs component's CRD into the cluster
        """
        kubeconfig = self.cluster_provisioner.kubeconfig()
        with tempfile.NamedTemporaryFile(prefix=f"kubeconfig-{self.cluster_name}-") as t:
            t.write(kubeconfig.encode("utf-8"))
            t.flush()

            out, err = Command() \
                .setenv("HOME", os.getenv("HOME")) \
                .setenv("USER", os.getenv("USER")) \
                .setenv("KUBECONFIG", t.name) \
                .setenv("GOCACHE", os.getenv("GOCACHE", "/tmp/gocache")) \
                .setenv("GOPATH", os.getenv("GOPATH", "/tmp/go")) \
                .run(f"make {component} deploy-cert-manager install")

            print(out)
            assert err == 0, f"error installing {component}'s manifests"

    def create_primaza_user(self, tenant: str, cluster_environment: str, timeout: int = 60):
        """
        Creates a ServiceAccount for `tenant` and `cluster_environment`,
        the secret with the JWT token, and creates the needed roles and role bindings.

        This operation is usually performed via `primazactl`.
        """
        sa_name = f"primaza-{tenant}-{cluster_environment}"
        sa_ns = "kube-system"
        api_client = self.get_api_client()
        corev1 = client.CoreV1Api(api_client)

        body = client.V1ServiceAccount(metadata=client.V1ObjectMeta(name=sa_name))
        sa = corev1.create_namespaced_service_account(namespace=sa_ns, body=body)

        sec_name = f"tkn-pmz-{tenant}-{cluster_environment}"
        tkn = client.V1Secret(
            metadata=client.V1ObjectMeta(
                name=sec_name,
                annotations={
                    "kubernetes.io/service-account.name": sa.metadata.name,
                },
                owner_references=[
                    client.V1OwnerReference(
                        api_version=sa.api_version,
                        kind=sa.kind,
                        name=sa.metadata.name,
                        uid=sa.metadata.uid),
                ]),
            type="kubernetes.io/service-account-token")

        corev1.create_namespaced_secret(namespace=sa_ns, body=tkn)
        polling2.poll(
            target=lambda: corev1.read_namespaced_secret(name=sec_name, namespace=sa_ns),
            check_success=lambda s: s is not None and s.data is not None and "token" in s.data,
            step=1,
            timeout=timeout)

    def create_application_namespace(self, namespace: str, tenant: str, cluster_environment: str):
        api_client = self.get_api_client()
        corev1 = client.CoreV1Api(api_client)

        self.install_agentapp_crd()

        # create application namespace
        try:
            ns = client.V1Namespace(metadata=client.V1ObjectMeta(name=namespace))
            corev1.create_namespace(ns)
        except ApiException as e:
            if e.reason != "Conflict":
                raise e

        self.__create_application_agent_identity(namespace)
        self.__allow_primaza_access_to_namespace(namespace, "app", tenant, cluster_environment)

    def create_service_namespace(self, namespace: str, tenant: str, cluster_environment: str):
        api_client = self.get_api_client()
        corev1 = client.CoreV1Api(api_client)

        self.install_agentsvc_crd()

        # create service namespace
        try:
            ns = client.V1Namespace(metadata=client.V1ObjectMeta(name=namespace))
            corev1.create_namespace(ns)
        except ApiException as e:
            if e.reason != "Conflict":
                raise e

        self.__create_service_agent_identity(namespace)
        self.__allow_primaza_access_to_namespace(namespace, "svc", tenant, cluster_environment)

    def __allow_primaza_access_to_namespace(self, namespace: str, nstype: str, tenant: str, cluster_environment: str):
        api_client = self.get_api_client()
        rbacv1 = client.RbacAuthorizationV1Api(api_client)

        sa_name = f"primaza-{tenant}-{cluster_environment}"
        sa_namespace = "kube-system"
        role_name = f"primaza:controlplane:{nstype}"
        pmz_rules = [client.V1PolicyRule(
            api_groups=["primaza.io"],
            resources=["serviceclasses"] if nstype == "svc" else ["servicebindings", "servicecatalogs"],
            verbs=["get", "list", "watch", "create", "update", "patch", "delete"])]

        r = client.V1Role(
            metadata=client.V1ObjectMeta(name=role_name, namespace=namespace),
            rules=[
                client.V1PolicyRule(
                    api_groups=[""],
                    resources=["secrets"],
                    verbs=["create"]),
                client.V1PolicyRule(
                    api_groups=[""],
                    resources=["secrets"],
                    verbs=["update", "patch"],
                    resource_names=[f"kubeconfig-primaza-{nstype}"]),
                client.V1PolicyRule(
                    api_groups=["apps"],
                    resources=["deployments"],
                    verbs=["create"]),
                client.V1PolicyRule(
                    api_groups=["apps"],
                    resources=["deployments"],
                    verbs=["delete"],
                    resource_names=[f"primaza-{nstype}-agent"]),
            ] + pmz_rules)
        rbacv1.create_namespaced_role(namespace, r)

        # bind role to service account
        rb = client.V1RoleBinding(
            metadata=client.V1ObjectMeta(name=role_name, namespace=namespace),
            role_ref=client.V1RoleRef(
                api_group="rbac.authorization.k8s.io",
                kind="Role",
                name=role_name),
            subjects=[
                client.V1Subject(
                    api_group="",
                    kind="ServiceAccount",
                    name=sa_name,
                    namespace=sa_namespace),
            ])
        rbacv1.create_namespaced_role_binding(namespace=namespace, body=rb)

    def __create_application_agent_identity(self, namespace: str):
        self.__create_agent_identity(namespace, "agentapp")

    def __create_service_agent_identity(self, namespace: str):
        self.__create_agent_identity(namespace, "agentsvc")

    def __create_agent_identity(self, namespace: str, component: str):
        kubeconfig = self.cluster_provisioner.kubeconfig()
        with tempfile.NamedTemporaryFile(prefix=f"kubeconfig-{self.cluster_name}-") as t:
            t.write(kubeconfig.encode("utf-8"))
            t.flush()

            out, err = Command() \
                .setenv("HOME", os.getenv("HOME")) \
                .setenv("USER", os.getenv("USER")) \
                .setenv("KUBECONFIG", t.name) \
                .setenv("NAMESPACE", namespace) \
                .setenv("GOCACHE", os.getenv("GOCACHE", "/tmp/gocache")) \
                .setenv("GOPATH", os.getenv("GOPATH", "/tmp/go")) \
                .run(f"make {component} deploy-rbac")

            print(out)
            assert err == 0, f"error deploying {component}'s rbac manifests"

    def get_primaza_sa_name(self, tenant: str, cluster_environment: str) -> str:
        user = f"primaza-{tenant}-{cluster_environment}"
        return user

    def get_primaza_sa_kubeconfig(self, tenant: str, cluster_environment: str) -> Dict:
        sec_name = f"tkn-pmz-{tenant}-{cluster_environment}"
        tkn = self.get_secret_token("kube-system", sec_name)
        user = self.get_primaza_sa_name(tenant, cluster_environment)

        kubeconfig = self.cluster_provisioner.kubeconfig(internal=True)
        kcd = yaml.safe_load(kubeconfig)
        kcd["contexts"][0]["context"]["user"] = user
        kcd["users"][0]["name"] = user
        kcd["users"][0]["user"]["token"] = base64.b64decode(tkn.encode("utf-8")).decode("utf-8")
        del kcd["users"][0]["user"]["client-key-data"]
        del kcd["users"][0]["user"]["client-certificate-data"]

        return kcd

    def get_primaza_sa_kubeconfig_yaml(self, tenant: str, cluster_environment: str) -> str:
        """
        Generates the kubeconfig for the Service Account for tenant `tenant`
        and ClusterEnvironment `cluster_environment`.
        The key used when creating the CSR is also needed.
        Returns the YAML string
        """
        kubeconfig = self.get_primaza_sa_kubeconfig(tenant, cluster_environment)
        return yaml.safe_dump(kubeconfig)

    def is_app_agent_deployed(self, namespace: str) -> bool:
        api_client = self.get_api_client()
        appsv1 = client.AppsV1Api(api_client)

        appsv1.read_namespaced_deployment(name="primaza-app-agent", namespace=namespace)
        return True

    def is_svc_agent_deployed(self, namespace: str) -> bool:
        api_client = self.get_api_client()
        appsv1 = client.AppsV1Api(api_client)

        appsv1.read_namespaced_deployment(name="primaza-svc-agent", namespace=namespace)
        return True

    def deploy_agentapp(self, namespace: str):
        """
        Deploys Application Agent into a cluster's namespace
        """

    def deploy_agentsvc(self, namespace: str):
        """
        Deploys the Service Agent into a cluster's namespace
        """

    def read_custom_resource_status(self, group: str, version: str, plural: str, name: str, namespace: str) -> str:
        api_client = self.get_api_client()
        api_instance = client.CustomObjectsApi(api_client)

        try:
            api_response = api_instance.get_namespaced_custom_object_status(group, version, namespace, plural, name)
            return api_response
        except ApiException as e:
            print("Exception when calling CustomObjectsApi->get_namespaced_custom_object_status: %s\n" % e)
            raise e

    def deploy_primaza_kubeconfig(self, primaza_cluster: Cluster, namespace: str):
        api_client = self.get_api_client()
        v1 = client.CoreV1Api(api_client)
        for n in ["kubeconfig-primaza-svc", "kubeconfig-primaza-app"]:
            secret = client.V1Secret(
                metadata=client.V1ObjectMeta(name=n),
                string_data={
                    "kubeconfig": primaza_cluster.get_admin_kubeconfig(internal=True),
                    # for now, assume primaza's default deployment namespace
                    "namespace": "primaza-system"
                })
            v1.create_namespaced_secret(namespace, secret)


# Behave steps
@given('Worker Cluster "{cluster_name}" for ClusterEnvironment "{ce_name}" is running')
@given('Worker Cluster "{cluster_name}" for tenant "{tenant}", and ClusterEnvironment "{ce_name}" is running')
@given('Worker Cluster "{cluster_name}" for ClusterEnvironment "{ce_name}" is running with kubernetes version "{version}"')
def ensure_worker_cluster_running_with_primaza(context, cluster_name: str, ce_name: str, version: str = None, tenant: str = "primaza-system"):
    worker_cluster = context.cluster_provider.create_worker_cluster(cluster_name, version)
    worker_cluster.start()

    worker_cluster.create_primaza_user(tenant=tenant, cluster_environment=ce_name)


@given('On Worker Cluster "{cluster_name}", a ServiceAccount for ClusterEnvironment "{cluster_environment}" exists')
@given('On Worker Cluster "{cluster_name}", a ServiceAccount for tenant "{tenant}" and ClusterEnvironment "{cluster_environment}" exists')
def on_worker_ensure_primaza_user(context, cluster_name: str, cluster_environment: str, version: str = None, tenant: str = "primaza-system"):
    worker_cluster = context.cluster_provider.create_worker_cluster(cluster_name, version)
    worker_cluster.create_primaza_user(tenant=tenant, cluster_environment=cluster_environment)


@step(u'On Worker Cluster "{cluster_name}", application namespace "{namespace}" for ClusterEnvironment "{cluster_environment}" exists')
def ensure_application_namespace_exists(
        context, cluster_name: str, namespace: str, cluster_environment: str, tenant: str = "primaza-system"):
    worker = context.cluster_provider.get_worker_cluster(cluster_name)  # type: WorkerCluster
    worker.create_application_namespace(namespace, tenant, cluster_environment)


@step(u'On Worker Cluster "{cluster_name}", Primaza Application Agent is deployed into namespace "{namespace}"')
def application_agent_is_deployed(context, cluster_name: str, namespace: str):
    worker = context.cluster_provider.get_worker_cluster(cluster_name)  # type: WorkerCluster
    worker.deploy_agentapp(namespace)
    polling2.poll(
        target=lambda: worker.is_app_agent_deployed(namespace),
        step=1,
        timeout=30)


@step(u'On Worker Cluster "{cluster_name}", Primaza Application Agent exists into namespace "{namespace}"')
def application_agent_exists(context, cluster_name: str, namespace: str):
    worker = context.cluster_provider.get_worker_cluster(cluster_name)  # type: WorkerCluster
    polling2.poll(
        target=lambda: worker.is_app_agent_deployed(namespace),
        step=1,
        timeout=30)


@step(u'On Worker Cluster "{cluster_name}", Primaza Application Agent does not exist into namespace "{namespace}"')
def application_agent_does_not_exist(context, cluster_name: str, namespace: str):
    worker = context.cluster_provider.get_worker_cluster(cluster_name)  # type: WorkerCluster

    def is_not_found():
        try:
            worker.is_app_agent_deployed(namespace)
        except ApiException as e:
            return e.reason == "Not Found"
        return False

    polling2.poll(
        target=lambda: is_not_found(),
        step=1,
        timeout=30)


@given(u'On Worker Cluster "{cluster_name}", service namespace "{namespace}" for ClusterEnvironment "{cluster_environment}" exists')
def ensure_service_namespace_exists(context, cluster_name: str, namespace: str, cluster_environment: str, tenant: str = "primaza-system"):
    worker = context.cluster_provider.get_worker_cluster(cluster_name)  # type: WorkerCluster
    worker.create_service_namespace(namespace, tenant, cluster_environment)


@step(u'On Worker Cluster "{cluster_name}", Primaza Service Agent is deployed into namespace "{namespace}"')
def service_agent_is_deployed(context, cluster_name: str, namespace: str):
    worker = context.cluster_provider.get_worker_cluster(cluster_name)  # type: WorkerCluster
    worker.deploy_agentsvc(namespace)
    polling2.poll(
        target=lambda: worker.is_svc_agent_deployed(namespace),
        step=1,
        timeout=30)


@step(u'On Worker Cluster "{cluster_name}", Primaza Service Agent exists into namespace "{namespace}"')
def service_agent_exists(context, cluster_name: str, namespace: str):
    worker = context.cluster_provider.get_worker_cluster(cluster_name)  # type: WorkerCluster
    polling2.poll(
        target=lambda: worker.is_svc_agent_deployed(namespace),
        step=1,
        timeout=30)


@step(u'On Worker Cluster "{cluster_name}", Primaza Service Agent does not exist into namespace "{namespace}"')
def service_agent_does_not_exist(context, cluster_name: str, namespace: str):
    worker = context.cluster_provider.get_worker_cluster(cluster_name)  # type: WorkerCluster

    def is_not_found():
        try:
            worker.is_svc_agent_deployed(namespace)
        except ApiException as e:
            return e.reason == "Not Found"
        return False

    polling2.poll(
        target=lambda: is_not_found(),
        step=1,
        timeout=30)


@given('Worker Cluster "{cluster_name}" is running')
@given('Worker Cluster "{cluster_name}" is running with kubernetes version "{version}"')
def ensure_worker_cluster_is_running(context, cluster_name: str, version: str = None):
    worker_cluster = context.cluster_provider.create_worker_cluster(cluster_name, version)
    worker_cluster.start()


@step(u'Primaza cluster\'s "{primaza_cluster}" kubeconfig is available on "{worker_cluster}" in namespace "{namespace}"')
def deploy_kubeconfig(context, primaza_cluster: str, worker_cluster: str, namespace: str):
    primaza = context.cluster_provider.get_primaza_cluster(primaza_cluster)  # type: Cluster
    worker = context.cluster_provider.get_worker_cluster(worker_cluster)  # type: WorkerCluster
    worker.deploy_primaza_kubeconfig(primaza, namespace)


@step(u'On Worker Cluster "{cluster_name}", the secret "{secret_name}" in namespace "{namespace}" has the key "{key}" with value "{value}"')
def ensure_secret_key_has_the_right_value(context, cluster_name: str, secret_name: str, namespace: str, key: str, value: str):
    primaza_cluster = context.cluster_provider.get_worker_cluster(cluster_name)
    polling2.poll(
        target=lambda: primaza_cluster.read_secret_resource_data(namespace, secret_name, key) == bytes(value, 'utf-8'),
        step=1,
        timeout=30)


@step(u'On Worker Cluster "{cluster_name}", the secret "{secret_name}" does not exist in namespace "{namespace}"')
def ensure_secret_not_exist(context, cluster_name: str, secret_name: str, namespace: str):
    cluster = context.cluster_provider.get_worker_cluster(cluster_name)  # type: WorkerCluster
    try:
        polling2.poll(
            target=lambda: cluster.read_secret(namespace, secret_name),
            step=1,
            timeout=10)
    except Exception:
        return
    raise Exception(f"not expecting secret '{secret_name}' to be found in namespace '{namespace}'")
