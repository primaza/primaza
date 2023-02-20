import re
import base64
import time
import json
import polling2
import ipaddress
import parse
import binascii
import tempfile
import yaml
from steps.environment import ctx
from steps.command import Command
from steps.util import substitute_scenario_id
from behave import register_type, step


class Kubernetes(object):

    deployment_template = '''
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: '{name}'
  namespace: {namespace}
  labels:
    app: myapp
spec:
  replicas: 1
  selector:
    matchLabels:
      app: myapp
  template:
    metadata:
      labels:
        app: myapp
    spec:
      containers:
      - name: myapp
        image: {image_name}
        env:
        - name: SERVICE_BINDING_ROOT
          value: {bindingRoot}
'''

    def __init__(self, kubeconfig=None):
        self.cmd = Command()
        if kubeconfig is not None:
            self.cmd.setenv("KUBECONFIG", kubeconfig)

    def get_pod_lst(self, namespace):
        return self.get_resource_lst("pods", namespace)

    def get_resource_lst(self, resource_plural, namespace):
        output, exit_code = self.cmd.run(
            f'{ctx.cli} get {resource_plural} -n {namespace} -o "jsonpath={{.items[*].metadata.name}}"')
        assert exit_code == 0, f"Getting resource list failed as the exit code is not 0 with output - {output}"
        if len(output.strip()) == 0:
            return list()
        else:
            return output.split(" ")

    def search_item_in_lst(self, lst, search_pattern):
        for item in lst:
            if re.fullmatch(search_pattern, item) is not None:
                print(f"item matched {item}")
                return item
        print("Given item not matched from the list of pods")
        return None

    def search_pod_in_namespace(self, pod_name_pattern, namespace):
        return self.search_resource_in_namespace("pods", pod_name_pattern, namespace)

    def search_resource_in_namespace(self, resource_plural, name_pattern, namespace):
        print(
            f"Searching for {resource_plural} that matches {name_pattern} in {namespace} namespace")
        lst = self.get_resource_lst(resource_plural, namespace)
        if len(lst) != 0:
            print("Resource list is {}".format(lst))
            return self.search_item_in_lst(lst, name_pattern)
        else:
            print('Resource list is empty under namespace - {}'.format(namespace))
            return None

    def is_resource_in(self, resource_type, resource_name=None):
        if resource_name is None:
            _, exit_code = self.cmd.run(f'{ctx.cli} get {resource_type}')
        else:
            _, exit_code = self.cmd.run(
                f'{ctx.cli} get {resource_type} {resource_name}')

        return exit_code == 0

    def wait_for_pod(self, pod_name_pattern, namespace, interval=5, timeout=600):
        pod = self.search_pod_in_namespace(pod_name_pattern, namespace)
        start = 0
        if pod is not None:
            return pod
        else:
            while ((start + interval) <= timeout):
                pod = self.search_pod_in_namespace(pod_name_pattern, namespace)
                if pod is not None:
                    return pod
                time.sleep(interval)
                start += interval
        return None

    def check_pod_status(self, pod_name, namespace, wait_for_status="Running"):
        cmd = f'{ctx.cli} get pod {pod_name} -n {namespace} -o "jsonpath={{.status.phase}}"'
        status_found, output, exit_status = self.cmd.run_wait_for_status(
            cmd, wait_for_status)
        return status_found

    def get_pod_status(self, pod_name, namespace):
        cmd = f'{ctx.cli} get pod {pod_name} -n {namespace} -o "jsonpath={{.status.phase}}"'
        output, exit_status = self.cmd.run(cmd)
        print(f"Get pod status: {output}, {exit_status}")
        if exit_status == 0:
            return output
        return None

    def apply(self, yaml, namespace=None, user=None):
        if namespace is not None:
            ns_arg = f"-n {namespace}"
        else:
            ns_arg = ""
        if user is not None:
            user_arg = f"--user={user}"
        else:
            user_arg = ""
        (output, exit_code) = self.cmd.run(
            f"{ctx.cli} apply {ns_arg} {user_arg} --validate=false -f -", yaml)
        assert exit_code == 0, f"Non-zero exit code ({exit_code}) while applying a YAML: {output}"
        return output

    def apply_invalid(self, yaml, namespace=None):
        if namespace is not None:
            ns_arg = f"-n {namespace}"
        else:
            ns_arg = ""
        (output, exit_code) = self.cmd.run(
            f"{ctx.cli} apply {ns_arg} -f -", yaml)
        assert exit_code != 0, f"the command should fail but it did not, output: {output}"
        return output

    def delete(self, yaml, namespace=None):
        if namespace is not None:
            ns_arg = f"-n {namespace}"
        else:
            ns_arg = ""
        (output, exit_code) = self.cmd.run(
            f"{ctx.cli} delete {ns_arg} -f -", yaml)
        assert exit_code == 0, f"Non-zero exit code ({exit_code}) while deleting a YAML: {output}"
        return output

    def delete_by_name(self, res_type, res_name, namespace=None):
        if namespace is not None:
            ns_arg = f"-n {namespace}"
        else:
            ns_arg = ""
        (output, exit_code) = self.cmd.run(
            f"{ctx.cli} delete {res_type} {res_name} {ns_arg}")
        assert exit_code == 0, f"Non-zero exit code ({exit_code}) while deleting a Custom Resource: {output}"
        return output

    def expose_service_route(self, name, namespace, port=""):
        output, exit_code = self.cmd.run(
            f'{ctx.cli} expose deployment {name} -n {namespace} --port={port} --type=NodePort')
        assert exit_code == 0, f"Could not expose deployment: {output}"

    def get_service_host(self, name, namespace):
        addr = self.get_node_address()
        output, exit_code = self.cmd.run(
            f'{ctx.cli} get service {name} -n {namespace} -o "jsonpath={{.spec.ports[0].nodePort}}"')
        host = f"{addr}:{output}"

        assert exit_code == 0, f"Getting route host failed as the exit code is not 0 with output - {output}"

        return host

    def get_node_address(self):
        output, exit_code = self.cmd.run(
            f'{ctx.cli} get nodes -o "jsonpath={{.items[0].status.addresses}}"')
        assert exit_code == 0, f"Error accessing Node resources - {output}"
        addresses = json.loads(output)
        for addr in addresses:
            if addr['type'] in ["InternalIP", "ExternalIP"]:
                return addr['address']
        assert False, f"No IP addresses found in {output}"

    def get_deployment_status(self, deployment_name, namespace, wait_for_status=None, interval=5, timeout=400):
        deployment_status_cmd = f'{ctx.cli} get deployment {deployment_name} -n {namespace} -o json' \
            + ' | jq -rc \'.status.conditions[] | select(.type=="Available").status\''
        output = None
        exit_code = -1
        if wait_for_status is not None:
            status_found, output, exit_code = self.cmd.run_wait_for_status(
                deployment_status_cmd, wait_for_status, interval, timeout)
            if exit_code == 0:
                assert status_found is True, f"Deployment {deployment_name} result after waiting for status is {status_found}"
        else:
            output, exit_code = self.cmd.run(deployment_status_cmd)
        assert exit_code == 0, "Getting deployment status failed as the exit code is not 0"

        return output

    def get_deployment_env_info(self, name, namespace):
        env_cmd = f'{ctx.cli} get deploy {name} -n {namespace} -o "jsonpath={{.spec.template.spec.containers[0].env}}"'
        env, exit_code = self.cmd.run(env_cmd)
        assert exit_code == 0, f"Non-zero exit code ({exit_code}) returned while getting deployment's env: {env}"
        return env

    def get_deployment_envFrom_info(self, name, namespace):
        env_from_cmd = f'{ctx.cli} get deploy {name} -n {namespace} -o "jsonpath={{.spec.template.spec.containers[0].envFrom}}"'
        env_from, exit_code = self.cmd.run(env_from_cmd)
        assert exit_code == 0, f"Non-zero exit code ({exit_code}) returned while getting deployment's envFrom: {env_from}"
        return env_from

    def get_resource_info_by_jsonpath(self, resource_type, name, namespace=None, json_path="{.*}", user=None):
        oc_cmd = f'{ctx.cli} get {resource_type} {name} -o "jsonpath={json_path}"'
        if namespace is not None:
            oc_cmd += f" -n {namespace}"
        if user:
            oc_cmd += f" --user={user}"
        output, exit_code = self.cmd.run(oc_cmd)
        if exit_code == 0:
            if resource_type == "secrets":
                return base64.decodebytes(bytes(output, 'utf-8')).decode('utf-8')
            else:
                return output
        else:
            print(
                f'Error getting value for {resource_type}/{name} in {namespace} path={json_path}: {output}')
            return None

    def get_json_resource(self, resource_type, name, namespace=None):
        error_msg = f"Error in getting resource: '{resource_type}' '{name}' namespace: '{namespace}'"
        cmd = f'{ctx.cli} get {resource_type} {name} -o json'
        if namespace is not None:
            cmd += f' -n {namespace}'
        output, exit_code = self.cmd.run(cmd)
        assert exit_code == 0, error_msg
        json_output = json.loads(output)
        assert json_output is not None, "Error in parsing JSON"
        return json_output

    def get_resource_info_by_jq(self, resource_type, name, namespace, jq_expression, wait=False, interval=5, timeout=120):
        output, exit_code = self.cmd.run(
            f'{ctx.cli} get {resource_type} {name} -n {namespace} -o json | jq  \'{jq_expression}\'')
        return output

    def get_docker_image_repository(self, name, namespace):
        cmd = f'{ctx.cli} get is {name} -n {namespace} -o "jsonpath={{.status.dockerImageRepository}}"'
        (output, exit_code) = self.cmd.run(cmd)
        assert exit_code == 0, f"cmd-{cmd} result for getting docker image repository is {output} with exit code-{exit_code} not equal to 0"
        return output

    def wait_for_build_pod_status(self, build_pod_name, namespace, wait_for_status="Succeeded", timeout=780):
        cmd = f'{ctx.cli} get pod {build_pod_name} -n {namespace} -o "jsonpath={{.status.phase}}"'
        status_found, output, exit_status = self.cmd.run_wait_for_status(
            cmd, wait_for_status, timeout=timeout)
        return status_found, output

    def get_deployment_name_in_namespace(self, deployment_name_pattern, namespace, wait=False, interval=5, timeout=120, resource="deployment"):
        if wait:
            start = 0
            while ((start + interval) <= timeout):
                deployment = self.search_resource_in_namespace(
                    resource, deployment_name_pattern, namespace)
                if deployment is not None:
                    return deployment
                time.sleep(interval)
                start += interval
            return None
        else:
            return self.search_resource_in_namespace(resource, deployment_name_pattern, namespace)

    def get_resource_list_in_namespace(self, resource_plural, name_pattern, namespace):
        print(
            f"Searching for {resource_plural} that matches {name_pattern} in {namespace} namespace")
        lst = self.get_resource_lst(resource_plural, namespace)
        if len(lst) != 0:
            print("Resource list is {}".format(lst))
            return self.get_all_matched_pattern_from_lst(lst, name_pattern)
        else:
            print('Resource list is empty under namespace - {}'.format(namespace))
            return []

    def get_all_matched_pattern_from_lst(self, lst, search_pattern):
        output_arr = []
        for item in lst:
            if re.fullmatch(search_pattern, item) is not None:
                print(f"item matched {item}")
                output_arr.append(item)
        if not output_arr:
            print("Given item not matched from the list of pods")
            return []
        else:
            return output_arr

    def search_resource_lst_in_namespace(self, resource_plural, name_pattern, namespace):
        print(
            f"Searching for {resource_plural} that matches {name_pattern} in {namespace} namespace")
        lst = self.get_resource_list_in_namespace(
            resource_plural, name_pattern, namespace)
        if len(lst) != 0:
            print("Resource list is {}".format(lst))
            return lst
        print('Resource list is empty under namespace - {}'.format(namespace))
        return None

    def lookup_namespace_for_resource(self, resource_plural, name):
        output, code = self.cmd.run(
            f"{ctx.cli} get {resource_plural} --all-namespaces -o json"
            + f" | jq -rc '.items[] | select(.metadata.name == \"{name}\").metadata.namespace'")
        assert code == 0, f"Non-zero return code while trying to detect namespace for {resource_plural} '{name}': {output}"
        output = str(output).strip()
        if output != "":
            return output
        else:
            return None

    def apply_yaml_file(self, yaml, namespace=None, validate=False):
        if namespace is not None:
            ns_arg = f"-n {namespace}"
        else:
            ns_arg = ""
        (output, exit_code) = self.cmd.run(
            f"{ctx.cli} apply {ns_arg} --validate={validate} -f " + yaml)
        assert exit_code == 0, "Applying yaml file failed as the exit code is not 0"
        return output

    def get_deployment_names_of_given_pattern(self, deployment_name_pattern, namespace):
        return self.search_resource_lst_in_namespace("deployment", deployment_name_pattern, namespace)

    def new_app(self, name, image_name, namespace, bindingRoot=None):
        cmd = f"{ctx.cli} create deployment {name} -n {namespace} --image={image_name}"
        if bindingRoot:
            (output, exit_code) = self.cmd.run(f"{ctx.cli} apply -f -",
                                               self.deployment_template.format(name=name, image_name=image_name,
                                                                               namespace=namespace, bindingRoot=bindingRoot))
        else:
            (output, exit_code) = self.cmd.run(cmd)
        assert exit_code == 0, f"Non-zero exit code ({exit_code}) returned when attempting to create a new app using following command line {cmd}\n: {output}"

    def set_label(self, name, label, namespace):
        cmd = f"{ctx.cli} label deployments {name} '{label}' -n {namespace}"
        (output, exit_code) = self.cmd.run(cmd)
        assert exit_code == 0, f"Non-zero exit code ({exit_code}) returned when attempting set label: {cmd}\n: {output}"

    def cli(self, cmd, namespace=None):
        bcmd = f'{ctx.cli} {cmd}'
        if namespace is not None:
            bcmd += f' -n {namespace}'

        output, exit_status = self.cmd.run(bcmd)
        assert exit_status == 0, "Exit should be zero"
        return output

    def check_for_condition(self, resource, name, namespace, condition, value, timeout=0):
        output, exit_code = self.cmd.run(
            f'{ctx.cli} wait --for=condition={condition}={value} {resource}/{name} --timeout={timeout}s -n {namespace}')
        assert exit_code == 0, f"Condition {condition}={value} for {resource}/{name} in {namespace} namespace was not met\n: {output}"

# Behave steps


@parse.with_pattern(r'.*')
def parse_nullable_string(text):
    return text


register_type(NullableString=parse_nullable_string)


@step(u'Secret contains "{secret_key}" key with value "{secret_value:NullableString}"')
def check_secret_key_value(context, secret_key, secret_value):
    sb = list(context.bindings.values())[0]
    kubernetes = context.kubernetes
    secret = polling2.poll(lambda: sb.get_secret_name(), step=100, timeout=1000, ignore_exceptions=(
        ValueError,), check_success=lambda v: v is not None)
    json_path = f'{{.data.{secret_key}}}'
    polling2.poll(lambda: kubernetes.get_resource_info_by_jsonpath("secrets", secret, context.namespace.name,
                  json_path) == secret_value, step=5, timeout=120, ignore_exceptions=(binascii.Error,))


@step(u'Secret contains "{secret_key}" key with dynamic IP addess as the value')
def check_secret_key_with_ip_value(context, secret_key):
    sb = list(context.bindings.values())[0]
    kubernetes = context.kubernetes
    secret = polling2.poll(lambda: sb.get_secret_name(), step=100, timeout=1000, ignore_exceptions=(
        ValueError,), check_success=lambda v: v is not None)
    json_path = f'{{.data.{secret_key}}}'
    polling2.poll(lambda: ipaddress.ip_address(
        kubernetes.get_resource_info_by_jsonpath("secrets", secret, context.namespace.name, json_path)),
        step=5, timeout=120, ignore_exceptions=(ValueError,))


@step(u'The Custom Resource is present')
def apply_yaml(context, user=None):
    kubernetes = context.kubernetes
    resource = substitute_scenario_id(context, context.text)
    metadata = yaml.full_load(resource)["metadata"]
    metadata_name = metadata["name"]
    if "namespace" in metadata:
        ns = metadata["namespace"]
    else:
        if "namespace" in context:
            ns = context.namespace.name
        else:
            ns = None
    output = kubernetes.apply(resource, ns, user)
    result = re.search(
        rf'.*{metadata_name}.*(created|unchanged|configured)', output)
    assert result is not None, f"Unable to apply YAML for CR '{metadata_name}': {output}"
    return metadata


@step(u'The Custom Resource is deleted')
def delete_yaml(context):
    kubernetes = context.kubernetes
    text = substitute_scenario_id(context, context.text)
    metadata = yaml.full_load(text)["metadata"]
    metadata_name = metadata["name"]
    if "namespace" in metadata:
        ns = metadata["namespace"]
    else:
        if "namespace" in context:
            ns = context.namespace.name
        else:
            ns = None
    output = kubernetes.delete(text, ns)
    result = re.search(rf'.*{metadata_name}.*(deleted)', output)
    assert result is not None, f"Unable to delete CR '{metadata_name}': {output}"


@step(u'Secret has been injected into CR "{cr_name}" of kind "{crd_name}" at path "{json_path}"')
def verify_injected_secretRef(context, cr_name, crd_name, json_path):
    sb = list(context.bindings.values())[0]
    name = substitute_scenario_id(context, cr_name)
    kubernetes = Kubernetes()
    secret = polling2.poll(lambda: sb.get_secret_name(), step=100, timeout=1000, ignore_exceptions=(
        ValueError,), check_success=lambda v: v is not None)
    polling2.poll(lambda: kubernetes.get_resource_info_by_jsonpath(crd_name, name, context.namespace.name, json_path) == secret,
                  step=5, timeout=400)


@step(u'Error message is thrown')
@step(u'Error message "{err_msg}" is thrown')
def validate_error(context, err_msg=None):
    if err_msg is None:
        assert context.expected_error is not None, "An error message should happen"
    else:
        search = re.search(rf'.*{err_msg}.*', context.expected_error)
        assert search is not None, f"Actual error: '{context.expected_error}', Expected error: '{err_msg}'"


@step(u'Secret does not contain "{key}"')
def check_secret_key(context, key):
    sb = list(context.bindings.values())[0]
    kubernetes = context.kubernetes
    secret = polling2.poll(lambda: sb.get_secret_name(context), step=100, timeout=1000,
                           ignore_exceptions=(ValueError,), check_success=lambda v: v is not None)
    json_path = f'{{.data.{key}}}'
    polling2.poll(lambda: kubernetes.get_resource_info_by_jsonpath("secrets", secret, context.namespace.name,
                                                                   json_path) == "",
                  step=5, timeout=120, ignore_exceptions=(binascii.Error,))


def assert_generation(context, count):
    context.latest_application_generation = context.application.get_generation()
    return context.latest_application_generation - context.original_application_generation == int(count)


@step(u'{type}/{resource_name} is available in the cluster')
def check_resource_exists(context, type, resource_name):
    kubernetes = context.kubernetes
    assert kubernetes.is_resource_in(
        type, resource_name), f"Could not find the resource: '{type}/{resource_name}' in the cluster"


@step(u"Condition {condition}={value} for {resource}/{name} resource is met")
@step(u"Condition {condition}={value} for {resource}/{name} resource is met in less then {timeout} seconds")
def condition_is_met_for_resource(context, condition, value, resource, name, timeout=600):
    kubernetes = Kubernetes()
    kubernetes.check_for_condition(
        resource, name, context.namespace.name, condition, value, timeout=timeout)


@step(u'On Primaza Cluster "{primaza_cluster}", Resource is created')
@step(u'On Primaza Cluster "{primaza_cluster}", Resource is updated')
def on_primaza_cluster_apply_yaml(context, primaza_cluster):
    resource = substitute_scenario_id(context, context.text)

    with tempfile.NamedTemporaryFile() as tf:
        kubeconfig = context.cluster_provider.get_primaza_cluster(primaza_cluster).get_admin_kubeconfig()
        tf.write(kubeconfig.encode("utf-8"))
        tf.flush()

        Kubernetes(kubeconfig=tf.name).apply(resource)


@step(u'On Primaza Cluster "{primaza_cluster}", RegisteredService "{primaza_rs}" is deleted')
def on_primaza_cluster_delete_registered_service(context, primaza_cluster, primaza_rs):

    with tempfile.NamedTemporaryFile() as tf:
        kubeconfig = context.cluster_provider.get_primaza_cluster(primaza_cluster).get_admin_kubeconfig()
        tf.write(kubeconfig.encode("utf-8"))
        tf.flush()

        Kubernetes(kubeconfig=tf.name).delete_by_name("registeredservice", primaza_rs, "primaza-system")
