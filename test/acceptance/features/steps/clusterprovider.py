from behave import given
from steps.primazacluster import PrimazaCluster
from steps.workercluster import WorkerCluster
from typing import Dict


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

    def create_primaza_cluster(self, cluster_name: str, version: str = None) -> PrimazaCluster:
        """
        Creates a new Worker cluster
        """
        cluster = self.build_primaza_cluster(cluster_name, version)
        self.clusters.primaza[cluster_name] = cluster
        return cluster

    def create_worker_cluster(self, cluster_name: str, version: str = None) -> WorkerCluster:
        """
        Creates a new Primaza cluster
        """
        cluster = self.build_worker_cluster(cluster_name, version)
        self.clusters.workers[cluster_name] = cluster
        return cluster

    def delete_clusters(self):
        """
        Deletes all the running cluster. Use this on cleanup.
        """
        for name, primaza in self.clusters.primaza.items():
            print(f"deleting primaza cluster '{name}'")
            primaza.delete()

        for name, worker in self.clusters.workers.items():
            print(f"deleting worker cluster '{name}'")
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
