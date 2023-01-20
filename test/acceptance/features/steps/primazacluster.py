import yaml
from behave import given
from cryptography import x509
from cryptography.x509.oid import NameOID
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import rsa
from cryptography.hazmat.primitives.asymmetric.rsa import RSAPrivateKey
from kubernetes import client
from kubernetes.client.rest import ApiException
from steps.cluster import Cluster


class PrimazaCluster(Cluster):
    """
    Base class for instances of Primaza clusters.
    Implements functionalities for configuration of kubernetes clusters that will host Primaza,
    like CertificateSigningRequest or ClusterContext creation.

    Concrete implementations built on this class will have to implement the `install_primaza` method,
    as the installation procedure usually varies with respect to the Cluster Provisioner
    (i.e., kind, minikube, openshift)
    """
    certificate_private_key: bytes = None
    certificate: RSAPrivateKey = None
    primaza_namespace: str = None

    def __init__(self, cluster_provisioner: str, cluster_name: str, namespace: str = "primaza-system"):
        super().__init__(cluster_provisioner, cluster_name)
        self.certificate = rsa.generate_private_key(public_exponent=65537, key_size=2048)
        self.certificate_private_key = self.certificate.private_bytes(
            format=serialization.PrivateFormat.PKCS8,
            encoding=serialization.Encoding.PEM,
            encryption_algorithm=serialization.NoEncryption()).decode("utf-8")
        self.primaza_namespace = namespace

    def start(self):
        super().start()
        self.install_primaza()

    def __create_certificate_signing_request(self):
        # Generate RSA Key and CertificateSignignRequest
        return x509.CertificateSigningRequestBuilder().subject_name(x509.Name([
            x509.NameAttribute(NameOID.COUNTRY_NAME, u"US"),
            x509.NameAttribute(NameOID.STATE_OR_PROVINCE_NAME, u""),
            x509.NameAttribute(NameOID.LOCALITY_NAME, u""),
            x509.NameAttribute(NameOID.ORGANIZATION_NAME, u'primaza'),
            x509.NameAttribute(NameOID.COMMON_NAME, u'primaza'),
        ])).add_extension(
            x509.SubjectAlternativeName([x509.DNSName(u"primaza.io")]),
            critical=False,
        ).sign(self.certificate, hashes.SHA256())

    def create_certificate_signing_request_pem(self) -> bytes:
        """
        Creates the V1CertificateSigningRequest needed for registration on a worker cluster
        """
        c = self.__create_certificate_signing_request()
        return c.public_bytes(serialization.Encoding.PEM)

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


# Behave steps
@given('Primaza Cluster "{cluster_name}" is running')
@given('Primaza Cluster "{cluster_name}" is running with kubernetes version "{version}"')
def ensure_primaza_cluster_running(context, cluster_name: str, version: str = None):
    cluster = context.cluster_provider.create_primaza_cluster(cluster_name, version)
    cluster.start()


@given('On Primaza Cluster "{primaza_cluster_name}", Worker "{worker_cluster_name}"\'s ClusterContext secret "{secret_name}" is published')
def ensure_primaza_cluster_has_worker_clustercontext(context, primaza_cluster_name: str, worker_cluster_name: str, secret_name: str):
    primaza_cluster = context.cluster_provider.get_primaza_cluster(primaza_cluster_name)
    worker_cluster = context.cluster_provider.get_worker_cluster(worker_cluster_name)

    cc_kubeconfig = worker_cluster.get_primaza_user_kubeconfig_yaml(primaza_cluster.certificate_private_key)
    primaza_cluster.create_clustercontext_secret(secret_name, cc_kubeconfig)


@given('On Primaza Cluster "{primaza_cluster_name}", an invalid Worker "{worker_cluster_name}"\'s ClusterContext secret "{secret_name}" is published')
def ensure_primaza_cluster_has_invalid_worker_clustercontext(context, primaza_cluster_name: str, worker_cluster_name: str, secret_name: str):
    primaza_cluster = context.cluster_provider.get_primaza_cluster(primaza_cluster_name)
    worker_cluster = context.cluster_provider.get_worker_cluster(worker_cluster_name)

    cc_kubeconfig = worker_cluster.get_primaza_user_kubeconfig(primaza_cluster.certificate_private_key)
    cc_kubeconfig["users"][0]["user"]["client-key-data"] = ''
    cc_kubeconfig["users"][0]["user"]["client-certificate-data"] = ''

    cc_kubeconfig_yaml = yaml.safe_dump(cc_kubeconfig)
    primaza_cluster.create_clustercontext_secret(secret_name, cc_kubeconfig_yaml)
