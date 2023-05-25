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
from typing import Dict, Tuple


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
        return None

    def read_secret(self, namespace: str, secret_name: str) -> client.V1Secret:
        api_client = self.get_api_client()

        corev1 = client.CoreV1Api(api_client)
        try:
            return corev1.read_namespaced_secret(name=secret_name, namespace=namespace)
        except ApiException as e:
            if e.reason != "Not Found":
                raise e
        return None

    def read_custom_object(self, namespace: str, group: str, version: str, plural: str, name: str) -> Dict:
        api_client = self.get_api_client()
        cobj = client.CustomObjectsApi(api_client)

        return cobj.get_namespaced_custom_object(
            namespace=namespace,
            group=group,
            version=version,
            plural=plural,
            name=name)

    def custom_object_exists(self, namespace: str, group: str, version: str, plural: str, name: str) -> bool:
        try:
            self.read_custom_object(
                namespace=namespace,
                group=group,
                version=version,
                plural=plural,
                name=name)
        except ApiException as e:
            if e.reason == "Not Found":
                return False
            raise e
        return True

    def read_primaza_custom_object(self, namespace: str, plural: str, name: str, version: str = "v1alpha1") -> Dict:
        return self.read_custom_object(
            namespace=namespace,
            group="primaza.io",
            version=version,
            plural=plural,
            name=name)

    def primaza_custom_object_exists(self, namespace: str, plural: str, name: str, version: str = "v1alpha1") -> bool:
        return self.custom_object_exists(
                namespace=namespace,
                group="primaza.io",
                version=version,
                plural=plural,
                name=name)

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

    def create_primaza_user(self, tenant: str, cluster_environment: str, timeout: int = 60) -> client.V1Secret:
        """
        Creates a ServiceAccount for `tenant` and `cluster_environment`,
        the secret with the JWT token, and creates the needed roles and role bindings.

        This operation is usually performed via `primazactl`.
        """
        sa_name = f"primaza-{tenant}-{cluster_environment}"
        sa_ns = "kube-system"
        sec_name = f"primaza-tkn-{tenant}-{cluster_environment}"

        return self.create_identity(sa_ns, sa_name, sec_name, timeout)

    def create_identity(self, namespace: str, service_account: str, secret: str, timeout: int) -> client.V1Secret:
        api_client = self.get_api_client()
        corev1 = client.CoreV1Api(api_client)

        body = client.V1ServiceAccount(metadata=client.V1ObjectMeta(name=service_account))
        sa = corev1.create_namespaced_service_account(namespace=namespace, body=body)

        tkn = client.V1Secret(
            metadata=client.V1ObjectMeta(
                name=secret,
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

        corev1.create_namespaced_secret(namespace=namespace, body=tkn)
        sec = polling2.poll(
            target=lambda: corev1.read_namespaced_secret(name=secret, namespace=namespace),
            check_success=lambda s: s is not None and s.data is not None and s.data.get("token", "").startswith("ZX"),
            ignore_exceptions=(ApiException,),
            step=1,
            timeout=timeout)
        return sec

    def create_application_namespace(self, namespace: str, tenant: str, cluster_environment: str, kubeconfig: str):
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

        self.__prepare_agent_namespace(namespace, "agentapp")
        self.__allow_primaza_access_to_namespace(namespace, "app", tenant, cluster_environment)
        self.__create_agent_kubeconfig_secret(namespace, "app", tenant, kubeconfig)

    def create_service_namespace(self, namespace: str, tenant: str, cluster_environment: str, kubeconfig: str):
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

        self.__prepare_agent_namespace(namespace, "agentsvc")
        self.__allow_primaza_access_to_namespace(namespace, "svc", tenant, cluster_environment)
        self.__create_agent_kubeconfig_secret(namespace, "svc", tenant, kubeconfig)

    def __create_agent_kubeconfig_secret(self, namespace: str, agent_type: str, tenant: str, kubeconfig: str):
        api_client = self.get_api_client()
        corev1 = client.CoreV1Api(api_client)

        secret = client.V1Secret(
            metadata=client.V1ObjectMeta(name=f"primaza-{agent_type}-kubeconfig"),
            string_data={
                "kubeconfig": kubeconfig,
                "namespace": tenant,
            })
        corev1.create_namespaced_secret(namespace, secret)

    def __allow_primaza_access_to_namespace(self, namespace: str, nstype: str, tenant: str, cluster_environment: str):
        api_client = self.get_api_client()
        rbacv1 = client.RbacAuthorizationV1Api(api_client)

        sa_name = f"primaza-{tenant}-{cluster_environment}"
        sa_namespace = "kube-system"
        role_name = f"primaza:controlplane:{nstype}"
        pmz_rules = [client.V1PolicyRule(
            api_groups=["primaza.io"],
            resources=["serviceclasses", "registeredservices"] if nstype == "svc" else ["servicebindings", "servicecatalogs", "serviceclaims"],
            verbs=["get", "list", "watch", "create", "update", "patch", "delete"])]

        r = client.V1Role(
            metadata=client.V1ObjectMeta(name=role_name, namespace=namespace),
            rules=[
                client.V1PolicyRule(
                    api_groups=[""],
                    resources=["secrets"],
                    verbs=["create", "get", "update"]),
                client.V1PolicyRule(
                    api_groups=[""],
                    resources=["configmaps"],
                    verbs=["create"]),
                client.V1PolicyRule(
                    api_groups=[""],
                    resources=["secrets"],
                    verbs=["update", "patch"],
                    resource_names=[f"primaza-{nstype}-kubeconfig"]),
                client.V1PolicyRule(
                    api_groups=["apps"],
                    resources=["deployments"],
                    verbs=["create"]),
                client.V1PolicyRule(
                    api_groups=["apps"],
                    resources=["deployments"],
                    verbs=["delete", "get"],
                    resource_names=[f"primaza-{nstype}-agent"]),
                client.V1PolicyRule(
                    api_groups=[""],
                    resources=["configmaps"],
                    verbs=["delete", "get"],
                    resource_names=[f"primaza-agent{nstype}-config"]),
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

    def bake_sa_kubeconfig(self, user: str, tkn: str) -> Dict:
        kubeconfig = self.cluster_provisioner.kubeconfig(internal=True)
        kcd = yaml.safe_load(kubeconfig)
        kcd["contexts"][0]["context"]["user"] = user
        kcd["users"][0]["name"] = user
        kcd["users"][0]["user"]["token"] = base64.b64decode(tkn.encode("utf-8")).decode("utf-8")
        del kcd["users"][0]["user"]["client-key-data"]
        del kcd["users"][0]["user"]["client-certificate-data"]

        return kcd

    def bake_sa_kubeconfig_yaml(self, user: str, tkn: str) -> str:
        kubeconfig = self.bake_sa_kubeconfig(user, tkn)
        return yaml.safe_dump(kubeconfig)

    def get_primaza_sa_kubeconfig(self, tenant: str, cluster_environment: str) -> Dict:
        sec_name = f"primaza-tkn-{tenant}-{cluster_environment}"
        tkn = self.get_secret_token("kube-system", sec_name)
        user = f"primaza-{tenant}-{cluster_environment}"

        return self.bake_sa_kubeconfig(user, tkn)

    def get_primaza_sa_kubeconfig_yaml(self, tenant: str, cluster_environment: str) -> str:
        """
        Generates the kubeconfig for the Service Account for tenant `tenant`
        and ClusterEnvironment `cluster_environment`.
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
        return dp.status.available_replicas == dp.status.replicas

    def read_custom_resource_status(self, group: str, version: str, plural: str, name: str, namespace: str) -> str:
        api_client = self.get_api_client()
        api_instance = client.CustomObjectsApi(api_client)

        try:
            api_response = api_instance.get_namespaced_custom_object_status(group, version, namespace, plural, name)
            return api_response
        except ApiException as e:
            print("Exception when calling CustomObjectsApi->get_namespaced_custom_object_status: %s\n" % e)
            raise e

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

    def create_agent_identity(self, agent_type: str, tenant: str, namespace: str, cluster_environment: str, timeout: int = 60) -> Tuple[client.V1Secret, str]:
        sa_name = f"primaza-{agent_type}-{cluster_environment}-{namespace}"
        sec_name = f"primaza-tkn-{agent_type}-{cluster_environment}-{namespace}"
        sec = self.create_identity(tenant, sa_name, sec_name, timeout)
        return (sec, sa_name)
