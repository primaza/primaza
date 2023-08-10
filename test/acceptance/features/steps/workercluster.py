import polling2
from behave import given, step, when
from kubernetes.client.rest import ApiException
from steps.cluster import Cluster
from steps.util import substitute_scenario_id


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
@given('Worker Cluster "{cluster_name}" for tenant "{tenant}" and ClusterEnvironment "{ce_name}" is running')
@given('Worker Cluster "{cluster_name}" for ClusterEnvironment "{ce_name}" is running with kubernetes version "{version}"')
def ensure_worker_cluster_running_with_primaza(
        context, cluster_name: str, ce_name: str, version: str = None, tenant: str = "primaza-system", timeout: int = 60):
    worker_cluster = context.cluster_provider.create_worker_cluster(cluster_name, version)
    worker_cluster.start()

    worker_cluster.create_primaza_user(tenant=tenant, cluster_environment=ce_name, timeout=timeout)


@when('Primaza Cluster "{cluster_name}" is deleted')
@when('Worker Cluster "{cluster_name}" is deleted')
def delete_cluster(context, cluster_name: str):
    context.cluster_provider.delete_cluster(cluster_name)


@given('On Worker Cluster "{cluster_name}", a ServiceAccount for ClusterEnvironment "{cluster_environment}" exists')
@given('On Worker Cluster "{cluster_name}", a ServiceAccount for tenant "{tenant}" and ClusterEnvironment "{cluster_environment}" exists')
def on_worker_ensure_primaza_user(context, cluster_name: str, cluster_environment: str, version: str = None, tenant: str = "primaza-system"):
    worker_cluster = context.cluster_provider.create_worker_cluster(cluster_name, version)
    worker_cluster.create_primaza_user(tenant=tenant, cluster_environment=cluster_environment)


@step(u'On Worker Cluster "{cluster_name}", application namespace "{namespace}" for ClusterEnvironment "{cluster_environment}" exists')
@step(u'On Worker Cluster "{cluster_name}", application namespace "{namespace}" for ClusterEnvironment "{cluster_environment}" in Cluster "{primaza_cluster_name}" exists')  # noqa: E501
@step(u'On Worker Cluster "{cluster_name}", application namespace "{namespace}" for ClusterEnvironment "{cluster_environment}" and tenant "{tenant}" in Cluster "{primaza_cluster_name}" exists')  # noqa: E501
def ensure_application_namespace_exists(
        context, cluster_name: str, namespace: str, cluster_environment: str, tenant: str = "primaza-system", primaza_cluster_name: str = "main"):
    worker = context.cluster_provider.get_worker_cluster(cluster_name)  # type: WorkerCluster
    primaza = context.cluster_provider.get_cluster(primaza_cluster_name)

    (secret,  name) = primaza.create_agent_identity(
        agent_type="app",
        tenant=tenant,
        cluster_environment=cluster_environment,
        namespace=namespace)
    kubeconfig = primaza.bake_sa_kubeconfig_yaml(name, secret.data["token"])
    worker.create_application_namespace(
            namespace=namespace,
            tenant=tenant,
            cluster_environment=cluster_environment,
            kubeconfig=kubeconfig)


@step(u'On Worker Cluster "{cluster_name}", Primaza Application Agent is deployed into namespace "{namespace}"')
def application_agent_is_deployed(context, cluster_name: str, namespace: str):
    worker = context.cluster_provider.get_worker_cluster(cluster_name)  # type: WorkerCluster
    worker.deploy_agentapp(namespace)
    polling2.poll(
        target=lambda: worker.is_app_agent_deployed(namespace),
        ignore_exceptions=(ApiException,),
        step=1,
        timeout=60)


@step(u'On Worker Cluster "{cluster_name}", Primaza Application Agent exists into namespace "{namespace}"')
def application_agent_exists(context, cluster_name: str, namespace: str):
    worker = context.cluster_provider.get_worker_cluster(cluster_name)  # type: WorkerCluster
    polling2.poll(
        target=lambda: worker.is_app_agent_deployed(namespace),
        ignore_exceptions=(ApiException,),
        step=1,
        timeout=60)


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
        timeout=60)


@step(u'On Worker Cluster "{cluster_name}", service namespace "{namespace}" for ClusterEnvironment "{cluster_environment}" exists')
@step(u'On Worker Cluster "{cluster_name}", service namespace "{namespace}" for ClusterEnvironment "{cluster_environment}" and tenant "{tenant}" in Primaza Cluster "{primaza_cluster_name}" exists')  # noqa: E501
def ensure_service_namespace_exists(
        context, cluster_name: str, namespace: str, cluster_environment: str, tenant: str = "primaza-system", primaza_cluster_name: str = "main"):
    worker = context.cluster_provider.get_worker_cluster(cluster_name)  # type: WorkerCluster
    primaza = context.cluster_provider.get_primaza_cluster(primaza_cluster_name)

    (secret,  name) = primaza.create_agent_identity(
        agent_type="svc",
        tenant=tenant,
        cluster_environment=cluster_environment,
        namespace=namespace)
    kubeconfig = primaza.bake_sa_kubeconfig_yaml(name, secret.data["token"])
    worker.create_service_namespace(namespace, tenant, cluster_environment, kubeconfig)


@step(u'On Worker Cluster "{cluster_name}", Primaza Service Agent exists into namespace "{namespace}"')
def service_agent_exists(context, cluster_name: str, namespace: str):
    worker = context.cluster_provider.get_worker_cluster(cluster_name)  # type: WorkerCluster
    polling2.poll(
        target=lambda: worker.is_svc_agent_deployed(namespace),
        ignore_exceptions=(ApiException,),
        step=1,
        timeout=60)


@step(u'On Worker Cluster "{cluster_name}", Primaza resource "{resource_type_name}" in namespace "{namespace}" has agent\'s ownership set')
def resource_has_agent_ownership_set(context, cluster_name: str, resource_type_name: str, namespace: str):
    worker = context.cluster_provider.get_worker_cluster(cluster_name)  # type: WorkerCluster
    rtn = resource_type_name.split("/")
    if len(rtn) != 2:
        raise Exception(f"invalid input: 'resource' should be in the format 'resource_type_plural/resource_name' (given {resource_type_name})")

    name = substitute_scenario_id(context, rtn[1])
    polling2.poll(
        target=lambda: worker.has_agent_ownership_set(rtn[0], name, namespace, "primaza.io", "v1alpha1"),
        ignore_exceptions=(ApiException,),
        step=1,
        timeout=60)


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
        timeout=60)


@given('Worker Cluster "{cluster_name}" is running')
@given('Worker Cluster "{cluster_name}" is running with kubernetes version "{version}"')
def ensure_worker_cluster_is_running(context, cluster_name: str, version: str = None):
    worker_cluster = context.cluster_provider.create_worker_cluster(cluster_name, version)
    worker_cluster.start()


@step(u'On Worker Cluster "{cluster_name}", the secret "{secret_name}" in namespace "{namespace}" has the key "{key}" with value "{value}"')
def ensure_secret_key_has_the_right_value(context, cluster_name: str, secret_name: str, namespace: str, key: str, value: str):
    primaza_cluster = context.cluster_provider.get_worker_cluster(cluster_name)
    polling2.poll(
        target=lambda: primaza_cluster.read_secret_resource_data(namespace, secret_name, key) == bytes(value, 'utf-8'),
        step=1,
        timeout=60)


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


@step(u'On Worker Cluster "{cluster_name}", the status of ServiceClaim "{service_claim_name}" is "{status}"')
def ensure_status_of_service_claim(context, cluster_name: str, service_claim_name: str, status: str):
    cluster = context.cluster_provider.get_worker_cluster(cluster_name)
    group = "primaza.io"
    version = "v1alpha1"
    plural = "serviceclaims"
    namespace = "applications"
    polling2.poll(
        target=lambda: cluster.read_custom_resource_status(
            group,
            version,
            plural,
            service_claim_name,
            namespace).get("status", {}).get("state", None) == status,
        step=1,
        timeout=60)


@step(u'On Worker Cluster "{cluster_name}", the RegisteredService bound to the ServiceClaim "{service_claim_name}" is "{registered_service}"')
def ensure_registered_service_of_service_claim(context, cluster_name: str, service_claim_name: str, registered_service: str):
    cluster = context.cluster_provider.get_worker_cluster(cluster_name)
    group = "primaza.io"
    version = "v1alpha1"
    plural = "serviceclaims"
    namespace = "applications"
    polling2.poll(
        target=lambda: cluster.read_custom_resource_status(
            group,
            version,
            plural,
            service_claim_name,
            namespace).get("status", {}).get("registeredService", None).get("name", None) == registered_service,
        step=1,
        timeout=60)


@step(u'On Worker Cluster "{cluster_name}", the status of RegisteredService "{name}" is "{status}"')
def ensure_status_of_registered_service(context, cluster_name: str, name: str, status: str):
    cluster = context.cluster_provider.get_worker_cluster(cluster_name)
    group = "primaza.io"
    version = "v1alpha1"
    plural = "registeredservices"
    namespace = "applications"
    polling2.poll(
        target=lambda: cluster.read_custom_resource_status(
            group,
            version,
            plural,
            name,
            namespace).get("status", {}).get("state", None) == status,
        step=1,
        timeout=60)
