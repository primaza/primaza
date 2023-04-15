import yaml
from behave import given, step
from kubernetes import client
from kubernetes.client.rest import ApiException
from steps.cluster import Cluster
import polling2


class PrimazaCluster(Cluster):
    """
    Base class for instances of Primaza clusters.
    Implements functionalities for configuration of kubernetes clusters that will host Primaza,
    like Service Account management, and ClusterContext creation.

    Concrete implementations built on this class will have to implement the `install_primaza` method,
    as the installation procedure usually varies with respect to the Cluster Provisioner
    (i.e., kind, minikube, openshift)
    """
    primaza_namespace: str = None

    def __init__(self, cluster_provisioner: str, cluster_name: str, namespace: str = "primaza-system"):
        super().__init__(cluster_provisioner, cluster_name)
        self.primaza_namespace = namespace

    def start(self):
        super().start()
        self.install_primaza()

    def create_clustercontext_secret(self, secret_name: str, kubeconfig: str):
        """
        Creates the Primaza's ClusterContext secret
        """
        api_client = self.get_api_client()
        namespace = self.primaza_namespace

        corev1 = client.CoreV1Api(api_client)
        try:
            corev1.read_namespaced_secret(name=secret_name, namespace=namespace)
            corev1.delete_namespaced_secret(name=secret_name, namespace=namespace)
        except ApiException as e:
            if e.reason != "Not Found":
                raise e

        secret = client.V1Secret(
            metadata=client.V1ObjectMeta(name=secret_name),
            string_data={"kubeconfig": kubeconfig})
        corev1.create_namespaced_secret(namespace=namespace, body=secret)

    def install_primaza(self):
        """
        Installs Primaza on the cluster. This method MUST be implemented by inheriting classes.
        """
        pass

    def read_custom_resource_status(self, group: str, version: str, plural: str, name: str) -> str:
        api_client = self.get_api_client()
        namespace = self.primaza_namespace
        api_instance = client.CustomObjectsApi(api_client)

        try:
            api_response = api_instance.get_namespaced_custom_object_status(group, version, namespace, plural, name)
            return api_response
        except ApiException as e:
            print("Exception when calling CustomObjectsApi->get_namespaced_custom_object_status: %s\n" % e)
            raise e

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


# Behave steps
@given('Primaza Cluster "{cluster_name}" is running')
@given('Primaza Cluster "{cluster_name}" is running with kubernetes version "{version}"')
def ensure_primaza_cluster_running(context, cluster_name: str, version: str = None):
    cluster = context.cluster_provider.create_primaza_cluster(cluster_name, version)
    cluster.start()


@given('On Primaza Cluster "{primaza_cluster_name}", Worker "{worker_cluster_name}"\'s ClusterContext secret "{secret_name}" for ClusterEnvironment "{ce_name}" is published')  # noqa: E501
def ensure_primaza_cluster_has_worker_clustercontext(
        context, primaza_cluster_name: str, worker_cluster_name: str, secret_name: str, ce_name: str, tenant: str = "primaza-system"):
    primaza_cluster = context.cluster_provider.get_primaza_cluster(primaza_cluster_name)
    worker_cluster = context.cluster_provider.get_worker_cluster(worker_cluster_name)

    cc_kubeconfig = worker_cluster.get_primaza_sa_kubeconfig_yaml(tenant, ce_name)
    primaza_cluster.create_clustercontext_secret(secret_name, cc_kubeconfig)


@given('On Primaza Cluster "{primaza_cluster_name}", an invalid Worker "{worker_cluster_name}"\'s ClusterContext secret "{secret_name}" for ClusterEnvironment "{ce_name}" is published')  # noqa: e501
def ensure_primaza_cluster_has_invalid_worker_clustercontext(
        context, primaza_cluster_name: str, worker_cluster_name: str, secret_name: str, ce_name: str, tenant: str = "primaza-system"):
    primaza_cluster = context.cluster_provider.get_primaza_cluster(primaza_cluster_name)
    worker_cluster = context.cluster_provider.get_worker_cluster(worker_cluster_name)

    cc_kubeconfig = worker_cluster.get_primaza_sa_kubeconfig(tenant, ce_name)
    cc_kubeconfig["users"][0]["user"]["token"] = ''

    cc_kubeconfig_yaml = yaml.safe_dump(cc_kubeconfig)
    primaza_cluster.create_clustercontext_secret(secret_name, cc_kubeconfig_yaml)


@step(u'On Primaza Cluster "{primaza_cluster_name}", Primaza ClusterContext secret "{secret_name}" is published')
def ensure_primaza_cluster_has_clustercontext(context, primaza_cluster_name: str, secret_name: str):
    primaza_cluster = context.cluster_provider.get_primaza_cluster(primaza_cluster_name)

    cc_kubeconfig = primaza_cluster.get_admin_kubeconfig(True)
    primaza_cluster.create_clustercontext_secret(secret_name, cc_kubeconfig)


@step(u'On Primaza Cluster "{primaza_cluster_name}", the status of ServiceClaim "{service_claim_name}" is "{status}"')
def ensure_status_of_service_claim(context, primaza_cluster_name: str, service_claim_name: str, status: str):
    primaza_cluster = context.cluster_provider.get_primaza_cluster(primaza_cluster_name)
    group = "primaza.io"
    version = "v1alpha1"
    plural = "serviceclaims"
    response = primaza_cluster.read_custom_resource_status(group, version, plural, service_claim_name)
    assert response["status"]["state"] == status


@step(u'On Primaza Cluster "{primaza_cluster_name}", the secret "{secret_name}" in namespace "{namespace}" has the key "{key}" with value "{value}"')
def ensure_secret_key_has_the_right_value(context, primaza_cluster_name: str, secret_name: str, namespace: str, key: str, value: str):
    primaza_cluster = context.cluster_provider.get_primaza_cluster(primaza_cluster_name)
    polling2.poll(
        target=lambda: primaza_cluster.read_secret_resource_data(namespace, secret_name, key) == bytes(value, 'utf-8'),
        step=1,
        timeout=30)


@step(u'On Primaza Cluster "{cluster_name}", Primaza Service Agent is deployed into namespace "{namespace}"')
def service_agent_is_deployed(context, cluster_name: str, namespace: str):
    primaza_cluster = context.cluster_provider.get_primaza_cluster(cluster_name)  # type: PrimazaCluster
    primaza_cluster.deploy_agentsvc(namespace)
    polling2.poll(
        target=lambda: primaza_cluster.is_svc_agent_deployed(namespace),
        step=1,
        timeout=30)


@step(u'On Primaza Cluster "{cluster_name}", Primaza Application Agent is deployed into namespace "{namespace}"')
def application_agent_is_deployed(context, cluster_name: str, namespace: str):
    primaza_cluster = context.cluster_provider.get_primaza_cluster(cluster_name)  # type: PrimazaCluster
    primaza_cluster.deploy_agentapp(namespace)
    polling2.poll(
        target=lambda: primaza_cluster.is_app_agent_deployed(namespace),
        step=1,
        timeout=30)
