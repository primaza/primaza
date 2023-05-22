from behave import given
from steps.primazacluster import PrimazaCluster
from steps.workercluster import WorkerCluster
from steps.cluster import Cluster
from typing import Dict, Optional


class ClustersStore(object):
    """
    Data store for the ClusterProvider class
    """
    primaza: Dict[str, PrimazaCluster] = {}
    workers: Dict[str, WorkerCluster] = {}


class ClusterProvider(object):
    """
    Manages clusters for a test environment.
    Use this object from context to create clusters and retrieve running ones.
    """
    clusters: ClustersStore = ClustersStore()

    def get_primaza_cluster(self, name: str) -> PrimazaCluster:
        """
        Returns the running Primaza cluster with name `name`
        """
        return self.clusters.primaza[name]

    def get_worker_cluster(self, name) -> WorkerCluster:
        """
        Returns the running Worker cluster with name `name`
        """
        return self.clusters.workers[name]

    def get_cluster(self, name: str) -> Optional[Cluster]:
        cluster = self.clusters.primaza.get(name)
        if cluster is not None:
            return cluster

        return self.clusters.workers.get(name)

    def create_primaza_cluster(self, cluster_name: str, version: str = None) -> PrimazaCluster:
        """
        Creates a new Primaza cluster
        """
        cluster = self.build_primaza_cluster(cluster_name, version)
        self.clusters.primaza[cluster_name] = cluster
        return cluster

    def create_worker_cluster(self, cluster_name: str, version: str = None) -> WorkerCluster:
        """
        Creates a new Worker cluster
        """
        cluster = self.build_worker_cluster(cluster_name, version)
        self.clusters.workers[cluster_name] = cluster
        return cluster

    def delete_cluster(self, cluster_name: str):
        """
        Deletes a running cluster by name
        """
        if cluster_name in self.clusters.primaza.keys():
            self.clusters.primaza.get[cluster_name].delete()

        if cluster_name in self.clusters.workers.keys():
            self.clusters.workers[cluster_name].delete()

    def delete_clusters(self):
        """
        Deletes all the running cluster. Use this on cleanup.
        """
        for name, primaza in self.clusters.primaza.items():
            primaza.delete()

        for name, worker in self.clusters.workers.items():
            worker.delete()

    def build_primaza_cluster(self, _name: str, _version: str = None):
        """
        Builds a Primaza cluster
        """
        pass

    def build_worker_cluster(self, _name: str, _version: str = None):
        """
        Creates a Worker cluster
        """
        pass

    def join_networks(self, _cluster1: str, _cluster2: str):
        """
        Ensure cluster1 and cluster2 can talk each other
        """
        pass


# Behave steps
@given(u'Clusters "{cluster1}" and "{cluster2}" can communicate')
def join_networks(context, cluster1: str, cluster2: str):
    context.cluster_provider.join_networks(cluster1, cluster2)
