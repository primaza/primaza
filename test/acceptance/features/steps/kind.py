import os
import tempfile
from steps.command import Command
from steps.clusterprovider import ClusterProvider
from steps.clusterprovisioner import ClusterProvisioner
from steps.primazacluster import PrimazaCluster
from steps.workercluster import WorkerCluster


class KindProvider(ClusterProvider):
    __prefix: str | None = None

    def __init__(self, prefix: str | None = None):
        if prefix is not None:
            self.__prefix = prefix

    def build_primaza_cluster(self, name, version):
        return PrimazaKind(self.__cluster_name(name), version)

    def build_worker_cluster(self, name, version):
        return WorkerKind(self.__cluster_name(name), version)

    def __cluster_name(self, name: str) -> str:
        if self.__prefix is None:
            return name
        return f"{self.__prefix}{name}"

    def join_networks(self, _cluster1, _cluster2):
        """
        Kind clusters shares the same docker network, so they can always communicate.
        """
        pass


class KindClusterProvisioner(ClusterProvisioner):
    def exec(self, command: str):
        cmd = Command()
        output, exit_code = cmd.run(command)
        return output, exit_code

    def start(self, timeout_sec: int = 600):
        image = f'image: "kindest/node:v{self.version}"' if self.version is not None else ""

        kind_config = f"""
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: {self.cluster_name}
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: "ClusterConfiguration"
    apiServer:
      extraArgs:
        anonymous-auth: "true"
  {image}
"""
        print(kind_config)
        with tempfile.NamedTemporaryFile() as tf:
            tf.write(kind_config.encode("utf-8"))
            tf.flush()

            cmd = f'kind create cluster --config {tf.name} && kubectl wait --for condition=Ready nodes --all --timeout={timeout_sec}s'
            output, exit_code = self.exec(cmd)
            return output, exit_code

    def delete(self):
        output, exit_code = self.exec(f'kind delete cluster --name {self.cluster_name}')
        return output, exit_code

    def kubeconfig(self, internal: bool = False) -> str:
        # To be used for test client to cluster communication
        cmd = f"kind get kubeconfig --name {self.cluster_name}"

        # To be used for cluster to cluster communication
        if internal:
            cmd = f"""
kind get kubeconfig --name {self.cluster_name} | \
sed "s/server: https:\\/\\/127\\.0\\.0\\.1:[0-9]*$/\
server: https:\\/\\/$(docker container inspect {self.cluster_name}-control-plane --format {{{{.NetworkSettings.Networks.kind.IPAddress}}}}:6443)/g"
            """

        output, exit_code = self.exec(cmd)
        assert exit_code == 0, f"error retrieving kubeconfig for cluster '{self.cluster_name}'"
        return output

    def ipaddress(self):
        output, exit_code = self.exec(f'docker container inspect {self.cluster_name}-control-plane --format {{{{.NetworkSettings.Networks.kind.IPAddress}}}}')
        assert exit_code == 0, f"error retrieving kubeconfig for cluster '{self.cluster_name}'"
        return output


class PrimazaKind(PrimazaCluster):
    def __init__(self, cluster_name: str, version: str = None):
        super().__init__(KindClusterProvisioner(cluster_name, version), cluster_name)

    def install_primaza(self):
        img = "primaza-controller:latest"

        kubeconfig = self.cluster_provisioner.kubeconfig()
        with tempfile.NamedTemporaryFile(prefix=f"kubeconfig-{self.cluster_name}-") as t:
            t.write(kubeconfig.encode("utf-8"))
            self.__build_load_and_deploy_primaza(t.name, img)
            self.__install_servicebinding_crd(t.name, img)

    # TODO: remove this when SBO is no longer needed
    def __install_servicebinding_crd(self, kubeconfig_path: str, img: str):
        sb = os.path.join(os.path.dirname(os.path.dirname(os.path.dirname(__file__))), "resources", "servicebinding.io_servicebindings.yaml")
        out, err = self.__build_install_base_cmd(kubeconfig_path, img).run(f"kubectl apply -f {sb}")
        print(out)
        assert err == 0, f"error installing ServiceBinding CRD: {sb}"

    def __build_load_and_deploy_primaza(self, kubeconfig_path: str, img: str):
        self.__install_crd_and_build_image(kubeconfig_path, img)
        self.__load_image(img)
        self.__deploy_primaza(kubeconfig_path, img)

    def __install_crd_and_build_image(self, kubeconfig_path: str, img: str):
        out, err = self.__build_install_base_cmd(kubeconfig_path, img).run("make primaza install docker-build")
        print(out)
        assert err == 0, "error installing manifests and building primaza controller"

    def __load_image(self, image: str):
        out, err = Command().run(f"kind load docker-image {image} --name {self.cluster_name}")
        print(out)
        assert err == 0, f"error loading image {image} into kind cluster {self.cluster_name}"

    def __deploy_primaza(self, kubeconfig_path: str, img: str):
        out, err = self.__build_install_base_cmd(kubeconfig_path, img).run("make primaza deploy")
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
        image = "agentsvc:latest"
        kubeconfig = self.cluster_provisioner.kubeconfig()
        with tempfile.NamedTemporaryFile(prefix=f"kubeconfig-{self.cluster_name}-") as t:
            t.write(kubeconfig.encode("utf-8"))
            self.__install_crd_and_build_svc_image(t.name, image)
            self.__load_image(image)
            self.__deploy_agentsvc(t.name, image, namespace)

    def deploy_agentapp(self, namespace: str):
        """
        Deploys Application Agent into a cluster's namespace
        """
        image = "agentapp:latest"
        kubeconfig = self.cluster_provisioner.kubeconfig()
        with tempfile.NamedTemporaryFile(prefix=f"kubeconfig-{self.cluster_name}-") as t:
            t.write(kubeconfig.encode("utf-8"))
            self.__install_crd_and_build_app_image(t.name, image)
            self.__load_image(image)
            self.__deploy_agentapp(t.name, image, namespace)

    def __install_crd_and_build_app_image(self, kubeconfig_path: str, image: str):
        out, err = self.__build_install_base_cmd(kubeconfig_path, image).run("make agentapp install docker-build")
        print(out)
        assert err == 0, "error installing manifests and building agent app  controller"

    def __install_crd_and_build_svc_image(self, kubeconfig_path: str, image: str):
        out, err = self.__build_install_base_cmd(kubeconfig_path, image).run("make agentsvc install docker-build")
        print(out)
        assert err == 0, "error installing manifests and building agent svc  controller"

    def __deploy_agentapp(self, kubeconfig_path: str, img: str, namespace: str):
        out, err = self.__build_install_base_cmd(kubeconfig_path, img).setenv("NAMESPACE", namespace).run("make agentapp deploy")
        print(out)
        assert err == 0, f"error deploying Agent app's controller into cluster {self.cluster_name}"

    def __deploy_agentsvc(self, kubeconfig_path: str, img: str, namespace: str):
        out, err = self.__build_install_base_cmd(kubeconfig_path, img).setenv("NAMESPACE", namespace).run("make agentsvc deploy")
        print(out)
        assert err == 0, f"error deploying Agent app's controller into cluster {self.cluster_name}"


class WorkerKind(WorkerCluster):
    __agentapp_loaded: bool = False

    def __init__(self, cluster_name, version=None):
        super().__init__(KindClusterProvisioner(cluster_name, version), cluster_name)

    def create_application_namespace(self, namespace: str):
        self.configure_application_cluster()
        super().create_application_namespace(namespace)

    def configure_application_cluster(self):
        if self.__agentapp_loaded:
            return

        self.__load_agentapp_image()

        self.__agentapp_loaded = True

    def __load_agentapp_image(self):
        cmd = f'make agentapp docker-build && kind load docker-image --name {self.cluster_name} $IMG'
        output, exit_code = Command().setenv("IMG", "agentapp:latest").run(cmd)

    def __load_image(self, image: str):
        cmd = f' kind load docker-image --name {self.cluster_name} $IMG'
        output, exit_code = Command().setenv("IMG", image).run(cmd)
        print(output)
        if exit_code != 0:
            raise Exception(f"error loading image {image}")

    def __deploy_agentapp(self, kubeconfig_path: str, image: str, namespace: str):
        out, err = self.__build_install_base_cmd(kubeconfig_path, image).setenv("NAMESPACE", namespace).run("make agentapp deploy")
        print(out)
        assert err == 0, f"error deploying Agent app's controller into cluster {self.cluster_name}"

    def __deploy_agentsvc(self, kubeconfig_path: str, img: str, namespace: str):
        out, err = self.__build_install_base_cmd(kubeconfig_path, img).setenv("NAMESPACE", namespace).run("make agentsvc deploy")
        print(out)
        assert err == 0, f"error deploying Agent app's controller into cluster {self.cluster_name}"

    def __install_crd_and_build_app_image(self, kubeconfig_path: str, image: str):
        out, err = self.__build_install_base_cmd(kubeconfig_path, image).run("make agentapp install docker-build")
        print(out)
        assert err == 0, "error installing manifests and building agent app  controller"

    def __install_crd_and_build_svc_image(self, kubeconfig_path: str, image: str):
        out, err = self.__build_install_base_cmd(kubeconfig_path, image).run("make agentsvc install docker-build")
        print(out)
        assert err == 0, "error installing manifests and building agent app  controller"

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
        image = "agentsvc:latest"
        kubeconfig = self.cluster_provisioner.kubeconfig()
        with tempfile.NamedTemporaryFile(prefix=f"kubeconfig-{self.cluster_name}-") as t:
            t.write(kubeconfig.encode("utf-8"))
            self.__install_crd_and_build_svc_image(t.name, image)
            self.__load_image(image)
            self.__deploy_agentsvc(t.name, image, namespace)

    def deploy_agentapp(self, namespace: str):
        """
        Deploys Application Agent into a cluster's namespace
        """
        image = "agentapp:latest"
        kubeconfig = self.cluster_provisioner.kubeconfig()
        with tempfile.NamedTemporaryFile(prefix=f"kubeconfig-{self.cluster_name}-") as t:
            t.write(kubeconfig.encode("utf-8"))
            self.__install_crd_and_build_app_image(t.name, image)
            self.__load_image(image)
            self.__deploy_agentapp(t.name, image, namespace)
