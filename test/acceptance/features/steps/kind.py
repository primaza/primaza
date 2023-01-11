import os
import tempfile
from steps.command import Command
from steps.clusterprovider import ClusterProvider
from steps.clusterprovisioner import ClusterProvisioner
from steps.primazacluster import PrimazaCluster
from steps.workercluster import WorkerCluster


class KindProvider(ClusterProvider):
    def __init__(self):
        pass

    def build_primaza_cluster(self, name, version):
        return PrimazaKind(name, version)

    def build_worker_cluster(self, name, version):
        return WorkerKind(name, version)

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
        anonymous-auth: "false"
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

    def kubeconfig(self):
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
        with tempfile.NamedTemporaryFile(prefix=f"kubeconfig-primaza-{self.cluster_name}-") as t:
            t.write(kubeconfig.encode("utf-8"))
            self.__build_load_and_deploy_primaza(t.name, img)

    def __build_load_and_deploy_primaza(self, kubeconfig_path: str, img: str):
        self.__install_crd_and_build_image(kubeconfig_path, img)
        self.__load_image(img)
        self.__deploy_primaza(kubeconfig_path, img)

    def __install_crd_and_build_image(self, kubeconfig_path: str, img: str):
        out, err = self.__build_install_primaza_base_cmd(kubeconfig_path, img).run("make install docker-build")
        print(out)
        assert err == 0, "error installing manifests and building primaza controller"

    def __load_image(self, img: str):
        out, err = Command().run(f"kind load docker-image {img} --name {self.cluster_name}")
        print(out)
        assert err == 0, f"error loading Primaza's controller into kind cluster {self.cluster_name}"

    def __deploy_primaza(self, kubeconfig_path: str, img: str):
        out, err = self.__build_install_primaza_base_cmd(kubeconfig_path, img).run("make deploy")
        print(out)
        assert err == 0, f"error deploying Primaza's controller into cluster {self.cluster_name}"

    def __build_install_primaza_base_cmd(self, kubeconfig_path: str, img: str) -> Command:
        return Command() \
            .setenv("KUBECONFIG", kubeconfig_path) \
            .setenv("GOCACHE", os.getenv("GOCACHE", "/tmp/gocache")) \
            .setenv("GOPATH", os.getenv("GOPATH", "/tmp/go")) \
            .setenv("IMG", img)


class WorkerKind(WorkerCluster):
    def __init__(self, cluster_name, version=None):
        super().__init__(KindClusterProvisioner(cluster_name, version), cluster_name)
