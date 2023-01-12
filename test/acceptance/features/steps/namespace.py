from behave import step
from command import Command
from environment import ctx
from util import get_env, substitute_scenario_id


class Namespace(object):
    def __init__(self, name):
        self.name = name
        self.cmd = Command()

    def create(self):
        output, exit_code = self.cmd.run(
            f"{ctx.cli} create namespace {self.name}")
        assert exit_code == 0, f"Unexpected output when creating namespace: '{output}'"
        return True

    def is_present(self):
        _, exit_code = self.cmd.run(f'{ctx.cli} get ns {self.name}')
        return exit_code == 0

    def delete(self):
        output, exit_code = self.cmd.run(
            f"{ctx.cli} delete namespace {self.name} --ignore-not-found=true")
        assert exit_code == 0, f"Unexpected output when deleting namespace: '{output}'"
        return True


# Behave steps
@step(u'Namespace "{namespace_name}" exists')
def namespace_maybe_create(context, namespace_name):
    namespace = Namespace(substitute_scenario_id(context, namespace_name))
    if not namespace.is_present():
        print("Namespace is not present, creating namespace: {}...".format(
            namespace_name))
        assert namespace.create(
        ), f"Unable to create namespace '{namespace_name}'"
    print("Namespace {} is created!!!".format(namespace_name))
    return namespace


@step(U'Namespace [{namespace_env}] exists')
def namespace_from_env_maybe_create(context, namespace_env):
    namespace_maybe_create(context, get_env(namespace_env))


@step(u'Namespace "{namespace_name}" is used')
def namespace_is_used(context, namespace_name):
    context.namespace = namespace_maybe_create(context, namespace_name)


@step(u'Namespace [{namespace_env}] is used')
def namespace_from_env_is_used(context, namespace_env):
    namespace_is_used(context, get_env(namespace_env))


@step(u'Namespace "{namespace_name}" is deleted')
def namespace_is_deleted(context, namespace_name):
    namespace = Namespace(substitute_scenario_id(context, namespace_name))
    assert namespace.delete(), "Failed to delete namespace"
