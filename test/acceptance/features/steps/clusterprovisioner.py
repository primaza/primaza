class ClusterProvisioner(object):
    """
    Represents a tool to use to create a kubernetes cluster.
    This class is meant to be inherited by concrete classes.
    Examples of concrete classes are KindClusterProvisioner, MinikubeClusterProvisioner.
    """

    def __init__(self, cluster_name: str, version: str = None):
        assert cluster_name is not None, "Kind cluster name is needed"
        self.cluster_name = cluster_name
        self.version = version

    def start(self, _timeout_sec: int = 600):
        """
        start the kubernetes cluster
        """
        pass

    def delete(self):
        """
        Deletes the kubernetes cluster
        """
        pass

    def kubeconfig(self, _internal: bool = False) -> str:
        """
        returns the admin kubeconfig
        """
        pass
