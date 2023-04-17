import polling2
from behave import given, step
from kubernetes.client.rest import ApiException
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
    kubeconfig = primaza.get_admin_kubeconfig(internal=True)

    worker = context.cluster_provider.get_worker_cluster(worker_cluster)  # type: WorkerCluster
    worker.deploy_primaza_kubeconfig(kubeconfig, namespace)


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
