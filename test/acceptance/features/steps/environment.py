import os
from steps.command import Command


class Environment(object):
    cli = "kubectl"

    def __init__(self, cli):
        self.cli = cli


global ctx
ctx = Environment(os.getenv("TEST_ACCEPTANCE_CLI", "kubectl"))
