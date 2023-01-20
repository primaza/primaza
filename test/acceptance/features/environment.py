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
from steps.command import Command
from steps.kind import KindProvider

cmd = Command()


@fixture
def use_kind(context, _timeout=30, **_kwargs):
    context.cluster_provider = KindProvider()
    yield context.cluster_provider
    context.cluster_provider.delete_clusters()


def before_scenario(context, _scenario):
    use_fixture(use_kind, context, timeout=30)
