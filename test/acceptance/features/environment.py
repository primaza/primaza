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
import os

cmd = Command()


class Runner(object):
    def __init__(self, num_runners: int = 1, runner_id: int = 0):
        assert num_runners >= 1
        assert runner_id < num_runners
        assert runner_id >= 0

        self.num_runners = num_runners
        self.runner_id = runner_id
        self.job_id = 0

    def should_skip(self) -> bool:
        return (self.job_id + self.runner_id) % self.num_runners != 0

    def next_job(self):
        self.job_id += 1


def get_envvar_int(name: str, default: int = 0) -> int:
    n = os.environ.get(name)
    if n is None:
        return default
    try:
        x = int(n)
    except ValueError:
        print(f"Failed to parse envvar {name} (value: {n}) as an integer")
    return x


@fixture
def use_kind(context, _timeout=30, **_kwargs):
    context.cluster_provider = KindProvider()
    yield context.cluster_provider
    context.cluster_provider.delete_clusters()


def before_all(context):
    context.runner = Runner(
        get_envvar_int("RUNS", 1),
        get_envvar_int("RUN_ID"))


def before_scenario(context, scenario):
    if context.runner.should_skip():
        scenario.skip("Scenario {scenario.name} not assigned to this runner!")
        return
    use_fixture(use_kind, context, timeout=30)


def after_scenario(context, _scenario):
    context.runner.next_job()
