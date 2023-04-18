import base64
import os
import polling2
import tempfile
import yaml
from kubernetes import client
from kubernetes.client.rest import ApiException
from steps.clusterprovisioner import ClusterProvisioner
from steps.util import get_api_client_from_kubeconfig
from steps.command import Command
from typing import Dict


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

    def get_secret_token(self, namespace: str, secret_name: str) -> str:
        api_client = self.get_api_client()
        corev1 = client.CoreV1Api(api_client)
        return corev1.read_namespaced_secret(name=secret_name, namespace=namespace).data["token"]

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
                    verbs=["delete", "get"],
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
        self.__prepare_agent_namespace(namespace, "agentapp")

    def __create_service_agent_identity(self, namespace: str):
        self.__prepare_agent_namespace(namespace, "agentsvc")

    def __prepare_agent_namespace(self, namespace: str, component: str):
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
                .run(f"make {component} prepare-namespace")

            print(out)
            assert err == 0, f"error deploying {component}'s rbac manifests"

    def get_primaza_sa_kubeconfig(self, tenant: str, cluster_environment: str) -> Dict:
        sec_name = f"tkn-pmz-{tenant}-{cluster_environment}"
        tkn = self.get_secret_token("kube-system", sec_name)
        user = f"primaza-{tenant}-{cluster_environment}"

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
        return self.agent_is_running(namespace, "app")

    def is_svc_agent_deployed(self, namespace: str) -> bool:
        return self.agent_is_running(namespace, "svc")

    def agent_is_running(self, namespace: str, agent_type: str) -> bool:
        api_client = self.get_api_client()
        appsv1 = client.AppsV1Api(api_client)

        dp = appsv1.read_namespaced_deployment_status(f"primaza-{agent_type}-agent", namespace)
        ar = dp.status.available_replicas
        er = dp.status.replicas

        return ar == er

    def read_custom_resource_status(self, group: str, version: str, plural: str, name: str, namespace: str) -> str:
        api_client = self.get_api_client()
        api_instance = client.CustomObjectsApi(api_client)

        try:
            api_response = api_instance.get_namespaced_custom_object_status(group, version, namespace, plural, name)
            return api_response
        except ApiException as e:
            print("Exception when calling CustomObjectsApi->get_namespaced_custom_object_status: %s\n" % e)
            raise e

    def deploy_primaza_kubeconfig(self, kubeconfig: str, namespace: str):
        api_client = self.get_api_client()
        v1 = client.CoreV1Api(api_client)
        for n in ["kubeconfig-primaza-svc", "kubeconfig-primaza-app"]:
            secret = client.V1Secret(
                metadata=client.V1ObjectMeta(name=n),
                string_data={
                    "kubeconfig": kubeconfig,
                    # for now, assume primaza's default deployment namespace
                    "namespace": "primaza-system"
                })
            v1.create_namespaced_secret(namespace, secret)

    def has_agent_ownership_set(self, resource_plural: str, resource_name: str, namespace: str, group: str, version: str):
        agent_names = ["primaza-app-agent", "primaza-svc-agent"]
        api_client = self.get_api_client()
        api_instance = client.CustomObjectsApi(api_client)

        api_response = api_instance.get_namespaced_custom_object(group, version, namespace, resource_plural, resource_name)
        print(api_response)
        owner_ref = api_response.get("metadata", {}).get("ownerReferences", [])
        if len(owner_ref) == 0:
            return False

        for r in owner_ref:
            if r["kind"] == "Deployment" and (r["name"] in agent_names):
                return True
        return False
