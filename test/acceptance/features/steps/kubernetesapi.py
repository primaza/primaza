import polling2
from behave import then, step
from datetime import datetime
from kubernetes import client


@step(u'On Primaza Cluster "{cluster}", ClusterEnvironment "{ce_name}" state will eventually move to "{state}"')
@step(u'On Primaza Cluster "{cluster}", ClusterEnvironment "{ce_name}" state will move to "{state}" in "{timeout}" seconds')
def on_primaza_cluster_check_state(context, cluster, ce_name, state, timeout=60):
    api_client = context.cluster_provider.get_primaza_cluster(cluster).get_api_client()
    cobj = client.CustomObjectsApi(api_client)

    polling2.poll(
        target=lambda: cobj.get_namespaced_custom_object_status(
            group="primaza.io",
            version="v1alpha1",
            namespace="primaza-system",
            plural="clusterenvironments",
            name=ce_name).get("status", {}).get("state", None),
        check_success=lambda x: x is not None and x == state,
        step=5,
        timeout=timeout)


@then(u'On Primaza Cluster "{cluster}", ClusterEnvironment "{ce_name}" last status condition has Type "{ctype}"')
def on_primaza_cluster_check_last_status_condition(context, cluster, ce_name, ctype):
    api_client = context.cluster_provider.get_primaza_cluster(cluster).get_api_client()
    cobj = client.CustomObjectsApi(api_client)

    ce_status = cobj.get_namespaced_custom_object_status(
        group="primaza.io",
        version="v1alpha1",
        namespace="primaza-system",
        plural="clusterenvironments",
        name=ce_name)
    ce_conditions = ce_status.get("status", {}).get("conditions", None)
    assert ce_conditions is not None and len(ce_conditions) > 0, "Cluster Environment status conditions are empty or not defined"

    last_applied = ce_conditions[0]
    for condition in ce_conditions[1:]:
        lat = datetime.fromisoformat(last_applied["last_transition_time"])
        cct = datetime.fromisoformat(condition["last_transition_time"])
        if cct > lat:
            last_applied = condition

    assert last_applied["type"] == ctype, f'Cluster Environment last condition type is not matching: wanted "{ctype}", found "{last_applied["type"]}"'


@step(u'On Primaza Cluster "{cluster}", ClusterEnvironment "{ce_name}" status condition with Type "{ctype}" has Status "{cstatus}"')
def on_primaza_cluster_check_status_condition(context, cluster: str, ce_name: str, ctype: str, cstatus: str):
    api_client = context.cluster_provider.get_primaza_cluster(cluster).get_api_client()
    cobj = client.CustomObjectsApi(api_client)

    ce_status = cobj.get_namespaced_custom_object_status(
        group="primaza.io",
        version="v1alpha1",
        namespace="primaza-system",
        plural="clusterenvironments",
        name=ce_name)
    ce_conditions = ce_status.get("status", {}).get("conditions", None)
    assert ce_conditions is not None and len(ce_conditions) > 0, "Cluster Environment status conditions are empty or not defined"

    for condition in ce_conditions:
        if condition["type"] == ctype:
            assert condition["status"] == cstatus, f'Cluster Environment condition with type "{ctype}" has status {condition["status"]}, wanted {cstatus}'
            return

    assert False, f"Cluster Environment does not have a condition with type {ctype}: {ce_conditions}"


@then(u'On Primaza Cluster "{cluster}", ClusterEnvironment "{ce_name}" in namespace "{namespace}" state remains not present')
def on_primaza_cluster_check_state_not_change(context, cluster, ce_name,  namespace="primaza-system", timeout=60):
    api_client = context.cluster_provider.get_primaza_cluster(cluster).get_api_client()
    cobj = client.CustomObjectsApi(api_client)

    try:
        state = polling2.poll(
            target=lambda: cobj.get_namespaced_custom_object_status(
                group="primaza.io",
                version="v1alpha1",
                namespace=namespace,
                plural="clusterenvironments",
                name=ce_name).get("status", {}).get("state", None),
            check_success=lambda x: x is not None,
            step=5,
            timeout=timeout)
        assert state is not None, f'Cluster Environment state is defined {state}, wanted undefined'
    except polling2.TimeoutException:
        pass


@step(u'On Primaza Cluster "{cluster}", RegisteredService "{rs_name}" state will eventually move to "{state}"')
def on_primaza_cluster_check_registered_service_status(context, cluster, rs_name, state):
    api_client = context.cluster_provider.get_primaza_cluster(cluster).get_api_client()
    cobj = client.CustomObjectsApi(api_client)

    polling2.poll(
        target=lambda: cobj.get_namespaced_custom_object_status(
            group="primaza.io",
            version="v1alpha1",
            namespace="primaza-system",
            plural="registeredservices",
            name=rs_name).get("status", {}).get("state", None),
        check_success=lambda x: x is not None and x == state,
        step=5,
        timeout=30)


def registered_service_in_catalog(rs_name, catalog):
    if "spec" in catalog and "services" in catalog["spec"]:
        for service in catalog["spec"]["services"]:
            if service["name"] == rs_name:
                return True
    return False


@step(u'On Primaza Cluster "{primaza_cluster}", RegisteredService "{primaza_rs}" state moves to "{state}"')
def on_primaza_cluster_claim_registered_service(context, primaza_cluster, primaza_rs, state):
    api_client = context.cluster_provider.get_primaza_cluster(primaza_cluster).get_api_client()
    cobj = client.CustomObjectsApi(api_client)
    cobj.patch_namespaced_custom_object_status(
            group="primaza.io",
            version="v1alpha1",
            namespace="primaza-system",
            plural="registeredservices",
            name=primaza_rs,
            body={"status": {"state": state}})


@then(u'On Primaza Cluster "{cluster}", ServiceCatalog "{catalog_name}" will contain RegisteredService "{rs_name}"')
def on_primaza_cluster_check_service_catalog_augmented(context, cluster, catalog_name, rs_name):
    api_client = context.cluster_provider.get_primaza_cluster(cluster).get_api_client()
    cobj = client.CustomObjectsApi(api_client)

    polling2.poll(
        target=lambda: cobj.get_namespaced_custom_object(
            group="primaza.io",
            version="v1alpha1",
            namespace="primaza-system",
            plural="servicecatalogs",
            name=catalog_name),
        check_success=lambda x: x is not None and registered_service_in_catalog(rs_name, x),
        step=5,
        timeout=20)


@then(u'On Primaza Cluster "{cluster}", ServiceCatalog "{catalog_name}" will not contain RegisteredService "{rs_name}"')
def on_primaza_cluster_check_service_catalog_reduced(context, cluster, catalog_name, rs_name):
    api_client = context.cluster_provider.get_primaza_cluster(cluster).get_api_client()
    cobj = client.CustomObjectsApi(api_client)

    polling2.poll(
        target=lambda: cobj.get_namespaced_custom_object(
            group="primaza.io",
            version="v1alpha1",
            namespace="primaza-system",
            plural="servicecatalogs",
            name=catalog_name),
        check_success=lambda x: x is not None and not registered_service_in_catalog(rs_name, x),
        step=5,
        timeout=20)


@step(u'On Primaza Cluster "{cluster}", ServiceBinding "{name}" on namespace "{namespace}" state will eventually move to "{state}"')
def on_primaza_cluster_check_service_binding_status(context, cluster, name, namespace, state, timeout=60):
    api_client = context.cluster_provider.get_primaza_cluster(cluster).get_api_client()
    cobj = client.CustomObjectsApi(api_client)

    polling2.poll(
        target=lambda: cobj.get_namespaced_custom_object_status(
            group="primaza.io",
            version="v1alpha1",
            namespace=namespace,
            plural="servicebindings",
            name=name).get("status", {}).get("state", None),
        check_success=lambda x: x is not None and x == state,
        step=5,
        timeout=timeout)


@step(u'On Worker Cluster "{cluster}", ServiceBinding "{name}" on namespace "{namespace}" state will eventually move to "{state}"')
def on_worker_cluster_check_service_binding_status(context, cluster, name, namespace, state, timeout=60):
    api_client = context.cluster_provider.get_worker_cluster(cluster).get_api_client()
    cobj = client.CustomObjectsApi(api_client)

    polling2.poll(
        target=lambda: cobj.get_namespaced_custom_object_status(
            group="primaza.io",
            version="v1alpha1",
            namespace=namespace,
            plural="servicebindings",
            name=name).get("status", {}).get("state", None),
        check_success=lambda x: x is not None and x == state,
        step=5,
        timeout=timeout)


@then(u'On Worker Cluster "{cluster}", Service Class "{sc_class}" exists in "{serviceclass_namespace}"')
def on_worker_cluster_check_service_class_exists_on_serviceclass_namespace(context, cluster, sc_class, serviceclass_namespace):
    api_client = context.cluster_provider.get_worker_cluster(cluster).get_api_client()
    cobj = client.CustomObjectsApi(api_client)
    polling2.poll(
        target=lambda: cobj.get_namespaced_custom_object(
            group="primaza.io",
            version="v1alpha1",
            namespace=serviceclass_namespace,
            plural="serviceclasses",
            name=sc_class),
        check_success=lambda x: x is not None,
        step=5,
        timeout=60)


@then(u'On Worker Cluster "{cluster}", Service Binding "{service_binding}" exists in "{application_namespace}"')
def on_worker_cluster_check_service_bindings_exists_on_application_namespace(context, cluster, service_binding, application_namespace):
    api_client = context.cluster_provider.get_worker_cluster(cluster).get_api_client()
    cobj = client.CustomObjectsApi(api_client)
    polling2.poll(
        target=lambda: cobj.get_namespaced_custom_object(
            group="primaza.io",
            version="v1alpha1",
            namespace=application_namespace,
            plural="servicebindings",
            name=service_binding),
        check_success=lambda x: x is not None,
        step=5,
        timeout=60)


@then(u'On Worker Cluster "{cluster}", Service Binding "{service_binding}" does not exist in "{namespace}"')
def on_worker_cluster_check_service_bindings_not_exists_in_application_namespace(context, cluster, service_binding, namespace):
    api_client = context.cluster_provider.get_worker_cluster(cluster).get_api_client()
    cobj = client.CustomObjectsApi(api_client)
    try:
        polling2.poll(
            target=lambda: cobj.get_namespaced_custom_object(
                group="primaza.io",
                version="v1alpha1",
                namespace=namespace,
                plural="servicebindings",
                name=service_binding),
            check_success=lambda x: x is not None,
            step=1,
            timeout=10)
        raise Exception(f"not expecting service binding '{service_binding}' to be found in namespace '{namespace}'")
    except Exception:
        pass
