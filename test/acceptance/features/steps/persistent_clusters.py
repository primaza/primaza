import os
import polling2
from steps.command import Command
from steps.clusterprovider import ClusterProvider
from steps.clusterprovisioner import ClusterProvisioner
from steps.primazacluster import PrimazaCluster
from steps.workercluster import WorkerCluster
from steps.util import get_env
from typing import List


class PersistentClusterProvider(ClusterProvider):

    def __init__(self, cluster_kubeconfig: str, worker_kubeconfig: str):
        super().__init__()
        self.__cluster_kubeconfig = cluster_kubeconfig
        self.__worker_kubeconfig = worker_kubeconfig

    def build_primaza_cluster(self, _name, _version):
        return PersistentPrimazaCluster(kubeconfig_path=self.__cluster_kubeconfig)

    def build_worker_cluster(self, _name, _version):
        return PersistentWorkerCluster(kubeconfig_path=self.__worker_kubeconfig)

    def __cluster_name(self, name: str) -> str:
        return f"{self.__prefix}{name}"

    def join_networks(self, _cluster1: str, _cluster2: str):
        # assume for now that the two clusters can talk to each other
        # TODO(sadlerap): actually check that connections can be established, maybe have them ping(8) each other?
        pass


class PersistentClusterProvisioner(ClusterProvisioner):
    def __init__(self, kubeconfig_path: str):
        self.__kubeconfig_path = kubeconfig_path

    def exec(self, command: str) -> (str, int):
        cmd = Command()
        output, exit_code = cmd.run(command)
        return output, exit_code

    def start(self, _timeout_sec: int = 600):
        # the cluster is already started, so nothing to do
        return "", 0

    def delete(self):
        # we're going to let the cluster manage it's own lifecycle, since we
        # don't necessarily know how to delete it.
        pass

    def kubeconfig(self, internal: bool = False) -> str:
        _ = internal
        with open(self.__kubeconfig_path, "r") as config:
            data = config.read()
            return data


class PersistentPrimazaCluster(PrimazaCluster):
    __controller_image: str = get_env("PRIMAZA_CONTROLLER_IMAGE_REF")
    __agentapp_image: str = get_env("PRIMAZA_AGENTAPP_IMAGE_REF")
    __agentsvc_image: str = get_env("PRIMAZA_AGENTSVC_IMAGE_REF")
    __has_controller_install: bool = False
    __has_agentapp_install: bool = False
    __has_agentsvc_install: bool = False

    def __init__(self, kubeconfig_path: str):
        self.__kubeconfig_path = kubeconfig_path
        super().__init__(PersistentClusterProvisioner(kubeconfig_path), "controller")

    def delete(self):
        cmd = Command() \
            .setenv("KUBECONFIG", self.__kubeconfig_path)
        for namespace in self.agentapp_namespaces:
            print(f"deleting application namespace in primaza cluster {self.cluster_name}: {namespace}")
            self.cleanup_application_namespace(namespace)
        for namespace in self.agentsvc_namespaces:
            print(f"deleting service namespace in primaza cluster {self.cluster_name}: {namespace}")
            self.cleanup_service_namespace(namespace)

        if self.__has_controller_install:
            print(f"deleting controller namespace in primaza cluster {self.cluster_name}: {self.primaza_namespace}")
            namespace = self.primaza_namespace
            self.cleanup_controller_namespace(namespace)
            out, err = cmd \
                .run(f"make primaza undeploy ignore-not-found=1 NAMESPACE={namespace}")
            assert err == 0, f"failed to cleanup test namespace {namespace}"

        # there's a potential for crds to overlap between agents and the
        # primaza controller. For instance, the primaza controller requires the
        # registeredservice crd to exist.  To avoid removal conflicts, run
        # agent undeployment after controller cleanup.
        for namespace in self.agentapp_namespaces:
            out, err = cmd \
                .run(f"make agentapp undeploy ignore-not-found=1 NAMESPACE={namespace}")
            assert err == 0, f"failed to cleanup application namespace {namespace}"
        for namespace in self.agentsvc_namespaces:
            out, err = cmd \
                .run(f"make agentsvc undeploy ignore-not-found=1 NAMESPACE={namespace}")
            assert err == 0, f"failed to cleanup service namespace {namespace}"

    def cleanup_controller_namespace(self, namespace: str):
        cmd = Command().setenv("KUBECONFIG", self.__kubeconfig_path)

        # cluster environments need to come last, since the other resource
        # controllers can fall over if cluster environments are removed before
        # they are.
        resource_names = ["registeredservices", "servicecatalogs", "serviceclaims", "serviceclasses", "clusterenvironments"]
        resources = list(map(lambda s: f"{s}.primaza.io", resource_names))
        resources.append("deployments.apps")
        self.delete_resources(cmd, namespace, resources)

    def cleanup_service_namespace(self, namespace: str):
        cmd = Command().setenv("KUBECONFIG", self.__kubeconfig_path)
        self.delete_resources(cmd, namespace, ["serviceclasses.primaza.io", "deployments"])

    def cleanup_application_namespace(self, namespace: str):
        cmd = Command().setenv("KUBECONFIG", self.__kubeconfig_path)
        resource_names = ["servicebindings", "servicecatalogs", "serviceclaims"]
        resources = list(map(lambda s: f"{s}.primaza.io", resource_names))
        resources.append("deployments.apps")
        self.delete_resources(cmd, namespace, resources)

    def delete_resources(self, cmd: Command, namespace: str, resource_types: List[str]):
        # for resiliency, patch out finalizers.  There may be bugs in the
        # controller, and we don't want to rely on finalizers working correctly
        # for testing to work.
        for resource_type in resource_types:
            cmd.run(f"kubectl delete -n {namespace} {resource_type} --all --ignore-not-found --timeout=30s")
            names, _ = cmd.run(f"kubectl get {resource_type} -o name -n {namespace}")
            for resource in names.splitlines():
                patch = '{"metadata": {"finalizers": []}}'
                cmd.run(f'kubectl patch -n {namespace} {resource} -p \'{patch}\' --type=merge')

        resource_list = ",".join(resource_types)
        out, err = cmd.run(f"kubectl delete -n {namespace} {resource_list} --all --ignore-not-found --timeout=30s")
        assert err == 0, "failed to run command!"

        polling2.poll(
            target=lambda: self.are_resources_deleted(cmd, namespace, resource_types),
            step=1,
            timeout=60)

    def are_resources_deleted(self, cmd: Command, namespace: str, resource_types: List[str]) -> bool:
        resource_list = ",".join(resource_types)
        o, ec = cmd.run(f"kubectl get {resource_list} -n {namespace} -o jsonpath='{{.items}}' | jq 'length'")
        return ec == 0 and o.rstrip() == "0"

    def is_resource_deleted(self, cmd: Command, namespace: str, rtype: str, rname: str) -> bool:
        o, ec = cmd.run(f"""kubectl get {rtype} -n {namespace} -o json | jq 'any(.items[]; .metadata.name="{rname}")'""")
        return ec == 0 and o.rstrip() == "0"

    def install_primaza(self):
        img = self.__controller_image
        kubeconfig = self.__kubeconfig_path

        self.__build_load_and_deploy_primaza(kubeconfig, img, self.primaza_namespace)
        self.__has_controller_install = True

    def __build_load_and_deploy_primaza(self, kubeconfig_path: str, img: str, namespace: str):
        self.__install_dependencies(kubeconfig_path)
        self.__deploy_primaza(kubeconfig_path, img, namespace)

    def __install_dependencies(self, kubeconfig_path: str):
        out, err = Command() \
            .setenv("KUBECONFIG", kubeconfig_path) \
            .run("make deploy-cert-manager")
        print(out)
        assert err == 0, "error installing dependencies"

    def __deploy_primaza(self, kubeconfig_path: str, img: str, namespace: str):
        out, err = self.__build_install_base_cmd(kubeconfig_path, img) \
            .setenv("NAMESPACE", namespace) \
            .run("make primaza deploy")
        print(out)
        assert err == 0, f"error deploying Primaza's controller into cluster {self.cluster_name}"

    def __build_install_base_cmd(self, kubeconfig_path: str, img: str) -> Command:
        return Command() \
            .setenv("HOME", os.getenv("HOME")) \
            .setenv("USER", os.getenv("USER")) \
            .setenv("KUBECONFIG", kubeconfig_path) \
            .setenv("GOCACHE", os.getenv("GOCACHE", "/tmp/gocache")) \
            .setenv("GOPATH", os.getenv("GOPATH", "/tmp/go")) \
            .setenv("IMG", img)

    def deploy_agentsvc(self, namespace: str):
        """
        Deploys the Service Agent into a cluster's namespace
        """
        image = self.__agentsvc_image
        self.__install_agentsvc_crd(self.__kubeconfig_path, image)
        self.__deploy_agentsvc(self.__kubeconfig_path, image, namespace)
        self.__has_agentsvc_install = True
        self.agentsvc_namespaces.add(namespace)

    def deploy_agentapp(self, namespace: str):
        """
        Deploys Application Agent into a cluster's namespace
        """
        image = self.__agentapp_image
        self.__install_agentapp_crd(self.__kubeconfig_path, image)
        self.__deploy_agentapp(self.__kubeconfig_path, image, namespace)
        self.__has_agentapp_install = True
        self.agentapp_namespaces.add(namespace)

    def __install_agentapp_crd(self, kubeconfig_path: str, image: str):
        out, err = self.__build_install_base_cmd(kubeconfig_path, image).run("make agentapp install")
        print(out)
        assert err == 0, "error installing manifests and building agent app  controller"

    def __install_agentsvc_crd(self, kubeconfig_path: str, image: str):
        out, err = self.__build_install_base_cmd(kubeconfig_path, image).run("make agentsvc install")
        print(out)
        assert err == 0, "error installing manifests and building agent svc  controller"

    def __deploy_agentapp(self, kubeconfig_path: str, img: str, namespace: str):
        out, err = self.__build_install_base_cmd(kubeconfig_path, img) \
            .setenv("IMG", img) \
            .setenv("NAMESPACE", namespace) \
            .run("make agentapp deploy")
        print(out)
        assert err == 0, f"error deploying Agent app's controller into cluster {self.cluster_name}"

    def __deploy_agentsvc(self, kubeconfig_path: str, img: str, namespace: str):
        out, err = self.__build_install_base_cmd(kubeconfig_path, img) \
            .setenv("IMG", img) \
            .setenv("NAMESPACE", namespace) \
            .run("make agentsvc deploy")
        print(out)
        assert err == 0, f"error deploying Agent app's controller into cluster {self.cluster_name}"


class PersistentWorkerCluster(WorkerCluster):
    __agentapp_loaded: bool = False
    __agentsvc_loaded: bool = False
    __agentapp_image: str = get_env("PRIMAZA_AGENTAPP_IMAGE_REF")
    __agentsvc_image: str = get_env("PRIMAZA_AGENTSVC_IMAGE_REF")

    def __init__(self, kubeconfig_path: str):
        self.__kubeconfig_path = kubeconfig_path
        super().__init__(PersistentClusterProvisioner(kubeconfig_path), "worker")

    def delete(self):
        cmd = Command() \
            .setenv("KUBECONFIG", self.__kubeconfig_path)
        for namespace in self.agentapp_namespaces:
            print(f"deleting application namespace in worker cluster {self.cluster_name}: {namespace}")
            self.cleanup_application_namespace(namespace)
        for namespace in self.agentsvc_namespaces:
            print(f"deleting service namespace in worker cluster {self.cluster_name}: {namespace}")
            self.cleanup_service_namespace(namespace)
        for namespace in self.agentapp_namespaces:
            out, err = cmd \
                .setenv("NAMESPACE", namespace) \
                .run("make agentapp undeploy ignore-not-found=1")
            assert err == 0, f"failed to cleanup application namespace {namespace}"
        for namespace in self.agentsvc_namespaces:
            out, err = cmd \
                .setenv("NAMESPACE", namespace) \
                .run("make agentsvc undeploy ignore-not-found=1")
            assert err == 0, f"failed to cleanup service namespace {namespace}"

    def cleanup_service_namespace(self, namespace: str):
        cmd = Command().setenv("KUBECONFIG", self.__kubeconfig_path)

        resourcelist = "serviceclasses.primaza.io"
        names, _ = cmd.run(f"kubectl get {resourcelist} -o name -n {namespace}")
        for resource in names.splitlines():
            patch = '{"metadata": {"finalizers": []}}'
            cmd.run(f'kubectl delete -n {namespace} {resource} --force --timeout=60s')
            cmd.run(f'kubectl patch -n {namespace} {resource} -p \'{patch}\' --type=merge')
        out, err = cmd.run(f"kubectl delete -n {namespace} {resourcelist} --all --ignore-not-found")
        assert err == 0, "failed to run command!"

        # delete any deployments in the namespace
        cmd.run(f"kubectl delete -n {namespace} deployments --all --ignore-not-found --timeout=30s")
        names, _ = cmd.run(f"kubectl get deployments.apps -o name -n {namespace}")
        for resource in names.splitlines():
            patch = '{"metadata": {"finalizers": []}}'
            cmd.run(f'kubectl patch -n {namespace} {resource} -p \'{patch}\' --type=merge')

    def cleanup_application_namespace(self, namespace: str):
        cmd = Command().setenv("KUBECONFIG", self.__kubeconfig_path)

        resource_names = ["servicebindings", "servicecatalogs", "serviceclaims"]
        resources = list(map(lambda s: f"{s}.primaza.io", resource_names))

        # for resiliency, patch out finalizers.  There may be bugs in the
        # controller, and we don't want to rely on finalizers working correctly
        # for testing to work.
        for crd in resources:
            names, _ = cmd.run(f"kubectl get {crd} -o name -n {namespace}")
            for resource in names.splitlines():
                patch = '{"metadata": {"finalizers": []}}'
                cmd.run(f'kubectl delete -n {namespace} {resource} --force --timeout=60s')
                cmd.run(f'kubectl patch -n {namespace} {resource} -p \'{patch}\' --type=merge')

        resourcelist = ",".join(resources)
        out, err = cmd.run(f"kubectl delete -n {namespace} {resourcelist} --all --ignore-not-found")
        assert err == 0, "failed to run command!"

        # delete any deployments in the namespace
        cmd.run(f"kubectl delete -n {namespace} deployments --all --ignore-not-found --timeout=30s")
        names, _ = cmd.run(f"kubectl get deployments.apps -o name -n {namespace}")
        for resource in names.splitlines():
            patch = '{"metadata": {"finalizers": []}}'
            cmd.run(f'kubectl patch -n {namespace} {resource} -p \'{patch}\' --type=merge')

    def __deploy_agentapp(self, kubeconfig_path: str, image: str, namespace: str):
        out, err = self.__build_install_base_cmd(kubeconfig_path, image).setenv("NAMESPACE", namespace).run("make agentapp deploy")
        print(out)
        assert err == 0, f"error deploying Agent app's controller into cluster {self.cluster_name}"

    def __deploy_agentsvc(self, kubeconfig_path: str, img: str, namespace: str):
        out, err = self.__build_install_base_cmd(kubeconfig_path, img).setenv("NAMESPACE", namespace).run("make agentsvc deploy")
        print(out)
        assert err == 0, f"error deploying Agent app's controller into cluster {self.cluster_name}"

    def __install_agentapp_crd(self, kubeconfig_path: str):
        out, err = self.__build_install_base_cmd(kubeconfig_path, self.__agentapp_image) \
            .run("make agentapp install")
        print(out)
        assert err == 0, "error installing manifests and building agent app controller"

    def __install_agentsvc_crd(self, kubeconfig_path: str):
        out, err = self.__build_install_base_cmd(kubeconfig_path, self.__agentsvc_image) \
            .run("make agentsvc install")
        print(out)
        assert err == 0, "error installing manifests and building agent svc controller"

    def __build_install_base_cmd(self, kubeconfig_path: str, image: str) -> Command:
        return Command() \
            .setenv("KUBECONFIG", kubeconfig_path) \
            .setenv("GOCACHE", os.getenv("GOCACHE", "/tmp/gocache")) \
            .setenv("GOPATH", os.getenv("GOPATH", "/tmp/go")) \
            .setenv("IMG", image)

    def deploy_agentsvc(self, namespace: str):
        """
        Deploys the Service Agent into a cluster's namespace
        """
        image = self.__agentsvc_image
        self.__install_agentsvc_crd(self.__kubeconfig_path)
        self.__deploy_agentsvc(self.__kubeconfig_path, image, namespace)
        self.agentsvc_namespaces.add(namespace)

    def deploy_agentapp(self, namespace: str):
        """
        Deploys Application Agent into a cluster's namespace
        """
        image = self.__agentapp_image
        self.__install_agentapp_crd(self.__kubeconfig_path)
        self.__deploy_agentapp(self.__kubeconfig_path, image, namespace)
        self.agentapp_namespaces.add(namespace)
