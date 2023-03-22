import base64
from kubernetes import client
from kubernetes.client.rest import ApiException
from steps.clusterprovisioner import ClusterProvisioner
from steps.util import get_api_client_from_kubeconfig


class Cluster(object):
    """
    Base class for managing a kubernetes cluster provisioned through a ClusterProvisioner
    """
    cluster_name: str = None
    cluster_provisioner: ClusterProvisioner = None

    def __init__(self, cluster_provisioner: ClusterProvisioner, cluster_name: str):
        self.cluster_provisioner = cluster_provisioner
        self.cluster_name = cluster_name

    def start(self):
        """
        Starts the cluster via the cluster provisioner
        """
        output, ec = self.cluster_provisioner.start()
        assert ec == 0, f'Worker Cluster "{self.cluster_name}" failed to start: {output}'
        print(f'Worker "{self.cluster_name}" started')

    def get_api_client(self):
        """
        Build and returns a client for the kubernetes API server of the cluster
        using the administrator user
        """
        kubeconfig = self.cluster_provisioner.kubeconfig()
        api_client = get_api_client_from_kubeconfig(kubeconfig)
        return api_client

    def delete(self):
        """
        Deletes the cluster via the cluster provisioner
        """
        self.cluster_provisioner.delete()

    def get_admin_kubeconfig(self, internal=False):
        """
        Returns the cluster admin kubeconfig
        """
        return self.cluster_provisioner.kubeconfig(internal)

    def read_secret_resource_data(self, namespace: str, secret_name: str, key: str) -> str:
        api_client = self.get_api_client()

        corev1 = client.CoreV1Api(api_client)
        try:
            secret = corev1.read_namespaced_secret(name=secret_name, namespace=namespace)
            b64value = secret.data[key]
            return base64.b64decode(b64value)
        except ApiException as e:
            if e.reason != "Not Found":
                raise e

    def read_secret(self, namespace: str, secret_name: str) -> client.V1Secret:
        api_client = self.get_api_client()

        corev1 = client.CoreV1Api(api_client)
        try:
            return corev1.read_namespaced_secret(name=secret_name, namespace=namespace)
        except ApiException as e:
            if e.reason != "Not Found":
                raise e
