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


from steps.command import Command
from steps.cluster import Kubernetes
from steps.environment import ctx

cmd = Command()


def before_all(_context):
    _context.kubernetes = Kubernetes()


def before_scenario(_context, _scenario):
    output, code = cmd.run(
        f'{ctx.cli} get ns default -o jsonpath="{{.metadata.name}}"')
    assert code == 0, f"Checking connection to OS cluster by getting the 'default' project failed: {output}"
