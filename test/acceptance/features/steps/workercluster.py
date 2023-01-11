import base64
import os
import tempfile
import time
import yaml
from behave import given
from typing import Dict
from datetime import datetime, timezone, timedelta
from kubernetes import client
from kubernetes.client.rest import ApiException
from steps.command import Command
from steps.cluster import Cluster


class WorkerCluster(Cluster):
    """
    Base class for instances of Worker clusters.
    Implements functionalities for configuration of kubernetes clusters that will act as Primaza workers,
    like CertificateSigningRequest approval and Primaza CRD installation.
    """

    def __init__(self, cluster_provisioner: str, cluster_name: str):
        super().__init__(cluster_provisioner, cluster_name)

    def start(self):
        super().start()
        self.install_primaza_crd()

    def install_primaza_crd(self):
        """
        Installs Primaza's CRD into the cluster
        """
        kubeconfig = self.cluster_provisioner.kubeconfig()
        with tempfile.NamedTemporaryFile(prefix=f"kubeconfig-primaza-{self.cluster_name}-") as t:
            t.write(kubeconfig.encode("utf-8"))

            out, err = Command() \
                .setenv("KUBECONFIG", t.name) \
                .setenv("GOCACHE", os.getenv("GOCACHE", "/tmp/gocache")) \
                .setenv("GOPATH", os.getenv("GOPATH", "/tmp/go")) \
                .run("make install")

            print(out)
            assert err == 0, "error installing Primaza's manifests"

    def create_primaza_user(self, csr_pem: bytes, timeout: int = 60):
        """
        Creates a CertificateSigningRequest for user primaza, approves it,
        and creates the needed roles and role bindings.
        """
        csr = "primaza"
        api_client = self.get_api_client()
        certs = client.CertificatesV1Api(api_client)

        # Check if CertificateSigningRequest has yet been created and approved
        try:
            s = certs.read_certificate_signing_request_status(name=csr)
            if s == "Approved":
                print(f"cluster '{self.cluster_name}' already has an approved CertificateSigningRequest '{csr}'")
                return
        except ApiException as e:
            if e.reason != "Not Found":
                raise e

        # Create CertificateSigningRequest
        v1csr = client.V1CertificateSigningRequest(
            metadata=client.V1ObjectMeta(name="primaza"),
            spec=client.V1CertificateSigningRequestSpec(
                signer_name="kubernetes.io/kube-apiserver-client",
                request=base64.b64encode(csr_pem).decode("utf-8"),
                expiration_seconds=86400,
                usages=["client auth"]))
        certs.create_certificate_signing_request(v1csr)

        # Approve CertificateSigningRequest
        v1csr = certs.read_certificate_signing_request(name=csr)
        approval_condition = client.V1CertificateSigningRequestCondition(
            last_update_time=datetime.now(timezone.utc).astimezone(),
            message='This certificate was approved by Acceptance tests',
            reason='Acceptance tests',
            type='Approved',
            status='True')
        v1csr.status.conditions = [approval_condition]
        certs.replace_certificate_signing_request_approval(name="primaza", body=v1csr)

        # Configure primaza user permissions
        self.__configure_primaza_user_permissions()

        # Wait for certificate emission
        tend = datetime.now() + timedelta(seconds=timeout)
        while datetime.now() < tend:
            v1csr = certs.read_certificate_signing_request(name=csr)
            status = v1csr.status
            if hasattr(status, 'certificate') and status.certificate is not None:
                print(f"CertificateSignignRequest '{csr}' certificate is ready")
                return
            print(f"CertificateSignignRequest '{csr}' certificate is not ready")
            time.sleep(5)
        assert False, f"Timed-out waiting CertificateSignignRequest '{csr}' certificate to become ready"

    def __configure_primaza_user_permissions(self):
        api_client = self.get_api_client()
        rbac = client.RbacAuthorizationV1Api(api_client)
        try:
            rbac.read_cluster_role_binding(name="primaza-primaza")
            return
        except ApiException as e:
            if e.reason != "Not Found":
                raise e

        role = client.V1ClusterRole(
            metadata=client.V1ObjectMeta(name="primaza"),
            rules=[
                client.V1PolicyRule(
                    api_groups=[""],
                    resources=["pods"],
                    verbs=["list", "get", "create"]),
            ])
        rbac.create_cluster_role(role)

        role_binding = client.V1ClusterRoleBinding(
            metadata=client.V1ObjectMeta(name="primaza-primaza"),
            role_ref=client.V1RoleRef(api_group="rbac.authorization.k8s.io", kind="ClusterRole", name="primaza"),
            subjects=[
                client.V1Subject(api_group="rbac.authorization.k8s.io", kind="User", name="primaza"),
            ])
        rbac.create_cluster_role_binding(role_binding)

    def get_csr_kubeconfig(self, certificate_key: str, csr: str) -> Dict:
        """
        Generates the kubeconfig for the CertificateSignignRequest `csr`.
        The key used when creating the CSR is also needed.
        """

        # retrieve primaza's certificate
        api_client = self.get_api_client()
        certs = client.CertificatesV1Api(api_client)
        v1csr = certs.read_certificate_signing_request(name=csr)
        certificate = v1csr.status.certificate

        # building kubeconfig
        kubeconfig = self.cluster_provisioner.kubeconfig()
        kcd = yaml.safe_load(kubeconfig)
        kcd["contexts"][0]["context"]["user"] = csr
        kcd["users"][0]["name"] = csr
        kcd["users"][0]["user"]["client-key-data"] = base64.b64encode(certificate_key.encode("utf-8")).decode("utf-8")
        kcd["users"][0]["user"]["client-certificate-data"] = certificate  # yet in base64 encoding

        return kcd

    def get_primaza_user_kubeconfig(self, certificate_key: str) -> Dict:
        """
        Generates the kubeconfig for the CertificateSignignRequest "primaza".
        The key used when creating the CSR is also needed.
        """
        return self.get_csr_kubeconfig(certificate_key, "primaza")

    def get_user_kubeconfig_yaml(self, certificate_key: str, csr: str) -> str:
        """
        Generates the kubeconfig for the CertificateSignignRequest `csr`.
        The key used when creating the CSR is also needed.

        Returns the YAML string
        """
        kubeconfig = self.get_csr_kubeconfig(certificate_key, csr)
        return yaml.safe_dump(kubeconfig)

    def get_primaza_user_kubeconfig_yaml(self, certificate_key: str) -> str:
        """
        Generates the kubeconfig for the CertificateSignignRequest "primaza".
        The key used when creating the CSR is also needed.

        Returns the YAML string
        """
        return self.get_user_kubeconfig_yaml(certificate_key, "primaza")


# Behave steps
@given('Worker Cluster "{cluster_name}" for "{primaza_cluster_name}" is running')
@given('Worker Cluster "{cluster_name}" for "{primaza_cluster_name}" is running with kubernetes version "{version}"')
def ensure_worker_cluster_running(context, cluster_name: str, primaza_cluster_name: str, version: str = None):
    worker_cluster = context.cluster_provider.create_worker_cluster(cluster_name, version)
    worker_cluster.start()

    primaza_cluster = context.cluster_provider.get_primaza_cluster(primaza_cluster_name)
    p_csr_pem = primaza_cluster.create_certificate_signing_request_pem()
    worker_cluster.create_primaza_user(p_csr_pem)
