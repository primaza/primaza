"""
before_step(context, step), after_step(context, step)
    These run before and after every step.
    The step passed in is an instance of Step.
before_scenario(context, scenario), after_scenario(context, scenario)
    These run before and after each scenario is run.
    The scenario passed in is an instance of Scenario.
before_feature(context, feature), after_feature(context, feature)
    These run before and after each feature file is exercised.
    The feature passed in is an instance of Feature.
before_all(context), after_all(context)
    These run before and after the whole shooting match.
"""


from behave import fixture, use_fixture
from steps.kind import KindProvider
from steps.persistent_clusters import PersistentClusterProvider
from steps.util import scenario_id, get_env


def is_development(context):
    return "wip" in context.tags and context._config.stop


@fixture
def use_kind(context, _timeout=30, **_kwargs):
    provider = get_env("CLUSTER_PROVIDER")
    if provider == "kind":
        context.cluster_provider = KindProvider(prefix=f"primaza-{scenario_id(context)}-")
    elif provider == "external":
        cluster_kubeconfig_path = get_env("MAIN_KUBECONFIG")
        worker_kubeconfig_path = get_env("WORKER_KUBECONFIG")
        context.cluster_provider = PersistentClusterProvider(
            cluster_kubeconfig=cluster_kubeconfig_path,
            worker_kubeconfig=worker_kubeconfig_path)
    yield context.cluster_provider

    # if development configuration is found and scenario failed, skip cleanup
    if is_development(context) and context.failed:
        print("wip, stop config and context.failed found: not cleaning up")
        return

    context.cluster_provider.delete_clusters()


def before_scenario(context, _scenario):
    use_fixture(use_kind, context, timeout=30)


def before_all(context):
    get_env("HOME")
    get_env("USER")

    for env in ["PRIMAZA_CONTROLLER_IMAGE_REF", "PRIMAZA_AGENTSVC_IMAGE_REF", "PRIMAZA_AGENTAPP_IMAGE_REF", "CLUSTER_PROVIDER"]:
        # assert they exist
        value = get_env(env)
        print(f"{env} = {value}")
