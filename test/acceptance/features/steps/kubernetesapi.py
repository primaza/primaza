import polling2
from behave import then, step
from datetime import datetime
from kubernetes import client
from kubernetes.client.rest import ApiException
from steps.util import substitute_scenario_id


@step(u'On Primaza Cluster "{cluster_name}", ClusterEnvironment "{ce_name}" state will eventually move to "{state}"')
@step(u'On Primaza Cluster "{cluster_name}", ClusterEnvironment "{ce_name}" state will move to "{state}" in "{timeout}" seconds')
def on_primaza_cluster_check_state(context, cluster_name, ce_name, state, timeout=60):
    api_client = context.cluster_provider.get_primaza_cluster(cluster_name).get_api_client()
    cobj = client.CustomObjectsApi(api_client)

    polling2.poll(
        target=lambda: cobj.get_namespaced_custom_object_status(
            group="primaza.io",
            version="v1alpha1",
            namespace="primaza-system",
            plural="clusterenvironments",
            name=ce_name).get("status", {}).get("state", None),
        check_success=lambda x: x is not None and x == state,
        step=1,
        timeout=timeout)


@then(u'On Primaza Cluster "{cluster_name}", ClusterEnvironment "{ce_name}" last status condition has Type "{ctype}"')
def on_primaza_cluster_check_last_status_condition(context, cluster_name, ce_name, ctype):
    api_client = context.cluster_provider.get_primaza_cluster(cluster_name).get_api_client()
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


@step(u'On Primaza Cluster "{cluster_name}", ClusterEnvironment "{ce_name}" status condition with Type "{ctype}" has Status "{cstatus}"')
def on_primaza_cluster_check_status_condition(context, cluster_name: str, ce_name: str, ctype: str, cstatus: str):
    api_client = context.cluster_provider.get_primaza_cluster(cluster_name).get_api_client()
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


@then(u'On Primaza Cluster "{cluster_name}", ClusterEnvironment "{ce_name}" in namespace "{namespace}" state remains not present')
def on_primaza_cluster_check_state_not_change(context, cluster_name, ce_name,  namespace="primaza-system", timeout=60):
    api_client = context.cluster_provider.get_primaza_cluster(cluster_name).get_api_client()
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
            step=1,
            timeout=timeout)
        assert state is not None, f'Cluster Environment state is defined {state}, wanted undefined'
    except polling2.TimeoutException:
        pass


@step(u'On Primaza Cluster "{cluster_name}", RegisteredService "{rs_name}" state will eventually move to "{state}"')
def on_primaza_cluster_check_registered_service_status(context, cluster_name, rs_name, state):
    api_client = context.cluster_provider.get_primaza_cluster(cluster_name).get_api_client()
    cobj = client.CustomObjectsApi(api_client)

    polling2.poll(
        target=lambda: cobj.get_namespaced_custom_object_status(
            group="primaza.io",
            version="v1alpha1",
            namespace="primaza-system",
            plural="registeredservices",
            name=rs_name).get("status", {}).get("state", None),
        check_success=lambda x: x is not None and x == state,
        step=1,
        timeout=60)


def registered_service_in_catalog(rs_name, catalog):
    if "spec" in catalog and "services" in catalog["spec"]:
        for service in catalog["spec"]["services"]:
            if service["name"] == rs_name:
                return True
    return False


def catalog_is_empty(catalog):
    return "spec" not in catalog or catalog["spec"] == {}


def registered_service_in_catalog_contains_service_class_identity(rs_name, sci, catalog):
    if "spec" in catalog and "services" in catalog["spec"]:
        for service in catalog["spec"]["services"]:
            if service["name"] == rs_name:
                service_class_identity = sci.split('=')
                for serviceclassidentity in service["serviceClassIdentity"]:
                    if serviceclassidentity["name"] == service_class_identity[0] and serviceclassidentity["value"] == service_class_identity[1]:
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


@step(u'On Primaza Cluster "{cluster_name}", ServiceCatalog "{catalog_name}" will contain RegisteredService "{rs_name}"')
def on_primaza_cluster_check_service_catalog_augmented(context, cluster_name, catalog_name, rs_name):
    cluster = context.cluster_provider.get_primaza_cluster(cluster_name)

    polling2.poll(
        target=lambda: cluster.read_primaza_custom_object(
            version="v1alpha1",
            namespace="primaza-system",
            plural="servicecatalogs",
            name=catalog_name),
        check_success=lambda x: x is not None and registered_service_in_catalog(rs_name, x),
        ignore_exceptions=(ApiException,),
        step=1,
        timeout=60)


@step(u'On Primaza Cluster "{cluster_name}", ServiceCatalog "{catalog_name}" will not contain RegisteredService "{rs_name}"')
def on_primaza_cluster_check_service_catalog_reduced(context, cluster_name, catalog_name, rs_name):
    cluster = context.cluster_provider.get_primaza_cluster(cluster_name)

    polling2.poll(
        target=lambda: cluster.read_primaza_custom_object(
            version="v1alpha1",
            namespace="primaza-system",
            plural="servicecatalogs",
            name=catalog_name),
        check_success=lambda x: x is not None and not registered_service_in_catalog(rs_name, x),
        ignore_exceptions=(ApiException,),
        step=1,
        timeout=60)


@step(u'On Primaza Cluster "{cluster_name}", ServiceBinding "{name}" on namespace "{namespace}" state will eventually move to "{state}"')
def on_primaza_cluster_check_service_binding_status(context, cluster_name, name, namespace, state, timeout=120):
    cluster = context.cluster_provider.get_primaza_cluster(cluster_name)
    return on_cluster_cluster_check_service_binding_status(cluster, name, namespace, state, timeout)


@step(u'On Worker Cluster "{cluster_name}", ServiceBinding "{name}" on namespace "{namespace}" state will eventually move to "{state}"')
def on_worker_cluster_check_service_binding_status(context, cluster_name, name, namespace, state, timeout=60):
    cluster = context.cluster_provider.get_worker_cluster(cluster_name)
    return on_cluster_cluster_check_service_binding_status(cluster, name, namespace, state, timeout)


def on_cluster_cluster_check_service_binding_status(cluster, name, namespace, state, timeout):
    polling2.poll(
        target=lambda: cluster.read_primaza_custom_object(
            version="v1alpha1",
            namespace=namespace,
            plural="servicebindings",
            name=name).get("status", {}).get("state", None),
        check_success=lambda x: x is not None and x == state,
        ignore_exceptions=(ApiException,),
        step=1,
        timeout=timeout)


@step(u'On Worker Cluster "{cluster_name}", Service Class "{sc_class}" exists in "{serviceclass_namespace}"')
def on_worker_cluster_check_service_class_exists_on_serviceclass_namespace(context, cluster_name, sc_class, serviceclass_namespace):
    cluster = context.cluster_provider.get_worker_cluster(cluster_name)
    name = substitute_scenario_id(context, sc_class)

    polling2.poll(
        target=lambda: cluster.primaza_custom_object_exists(
            version="v1alpha1",
            namespace=serviceclass_namespace,
            plural="serviceclasses",
            name=name),
        step=1,
        timeout=60)


@step(u'On Worker Cluster "{cluster_name}", Service Binding "{service_binding}" exists in "{application_namespace}"')
def on_worker_cluster_check_service_bindings_exists_on_application_namespace(context, cluster_name, service_binding, application_namespace):
    cluster = context.cluster_provider.get_worker_cluster(cluster_name)
    polling2.poll(
        target=lambda: cluster.primaza_custom_object_exists(
            version="v1alpha1",
            namespace=application_namespace,
            plural="servicebindings",
            name=service_binding),
        step=1,
        timeout=60)


@step(u'On Worker Cluster "{cluster_name}", Service Binding "{service_binding}" does not exist in "{namespace}"')
def on_worker_cluster_check_service_bindings_not_exists_in_application_namespace(context, cluster_name, service_binding, namespace):
    cluster = context.cluster_provider.get_worker_cluster(cluster_name)

    polling2.poll(
        target=lambda: not cluster.primaza_custom_object_exists(
            version="v1alpha1",
            namespace=namespace,
            plural="servicebindings",
            name=service_binding),
        step=1,
        timeout=180)


@step(u'On Worker Cluster "{cluster_name}", ServiceCatalog "{catalog}" exists in "{application_namespace}"')
def on_worker_cluster_check_service_catalog_exists_on_application_namespace(context, cluster_name, catalog, application_namespace):
    cluster = context.cluster_provider.get_worker_cluster(cluster_name)

    polling2.poll(
        target=lambda: cluster.primaza_custom_object_exists(
            version="v1alpha1",
            namespace=application_namespace,
            plural="servicecatalogs",
            name=catalog),
        step=1,
        timeout=60)


@then(u'On Worker Cluster "{cluster_name}", Service Class "{service_class}" does not exist in "{namespace}"')
def on_worker_cluster_check_service_class_not_exists_in_service_namespace(context, cluster_name, service_class, namespace):
    cluster = context.cluster_provider.get_worker_cluster(cluster_name)
    name = substitute_scenario_id(context, service_class)

    polling2.poll(
        target=lambda: not cluster.primaza_custom_object_exists(
            version="v1alpha1",
            namespace=namespace,
            plural="serviceclasses",
            name=name),
        step=1,
        timeout=60)


@step(u'On Primaza Cluster "{cluster_name}", ServiceCatalog "{catalog}" exists')
def on_primaza_cluster_check_service_catalog_exists(context, cluster_name, catalog):
    cluster = context.cluster_provider.get_primaza_cluster(cluster_name)

    polling2.poll(
        target=lambda: cluster.primaza_custom_object_exists(
            version="v1alpha1",
            namespace="primaza-system",
            plural="servicecatalogs",
            name=catalog),
        step=1,
        timeout=60)


@step(u'On Primaza Cluster "{cluster_name}", ServiceCatalog "{catalog_name}" does not exist')
def on_primaza_cluster_check_service_catalog_does_not_exists(context, cluster_name, catalog_name):
    cluster = context.cluster_provider.get_primaza_cluster(cluster_name)

    polling2.poll(
        target=lambda: not cluster.primaza_custom_object_exists(
            version="v1alpha1",
            namespace="primaza-system",
            plural="servicecatalogs",
            name=catalog_name),
        step=1,
        timeout=60)


@then(u'On Primaza Cluster "{cluster_name}", ServiceCatalog "{catalog_name}" is empty')
def on_primaza_cluster_check_service_catalog_empty(context, cluster_name, catalog_name):
    cluster = context.cluster_provider.get_primaza_cluster(cluster_name)

    polling2.poll(
        target=lambda: cluster.read_primaza_custom_object(
            version="v1alpha1",
            namespace="primaza-system",
            plural="servicecatalogs",
            name=catalog_name),
        check_success=lambda x: x is not None and catalog_is_empty(x),
        ignore_exceptions=(ApiException,),
        step=1,
        timeout=60)


@step(u'On Worker Cluster "{cluster_name}", ServiceCatalog "{catalog_name}" in application namespace "{namespace}" will contain RegisteredService "{rs_name}"')
def on_worker_cluster_check_service_catalog_augmented(context, cluster_name, catalog_name, rs_name, namespace):
    cluster = context.cluster_provider.get_worker_cluster(cluster_name)

    polling2.poll(
        target=lambda: cluster.read_primaza_custom_object(
            version="v1alpha1",
            namespace=namespace,
            plural="servicecatalogs",
            name=catalog_name),
        check_success=lambda x: x is not None and registered_service_in_catalog(rs_name, x),
        ignore_exceptions=(ApiException,),
        step=1,
        timeout=60)


@step(u'On Worker Cluster "{cluster_name}", ServiceCatalog "{catalog_name}" in application namespace "{namespace}" will not contain RegisteredService "{rs_name}"')  # noqa: E501
def on_worker_cluster_check_service_catalog_reduced(context, cluster_name, catalog_name, rs_name, namespace):
    cluster = context.cluster_provider.get_worker_cluster(cluster_name)

    polling2.poll(
        target=lambda: cluster.read_primaza_custom_object(
            version="v1alpha1",
            namespace=namespace,
            plural="servicecatalogs",
            name=catalog_name),
        check_success=lambda x: x is not None and not registered_service_in_catalog(rs_name, x),
        ignore_exceptions=(ApiException,),
        step=1,
        timeout=60)


@step(u'On Primaza Cluster "{cluster_name}", ServiceCatalog "{catalog_name}" has RegisteredService "{registered_service}" with Service Class Identity "{sci}"')
def check_on_primaza_cluster_registered_service_contains_identity(context, cluster_name, catalog_name, registered_service, sci):
    cluster = context.cluster_provider.get_primaza_cluster(cluster_name)

    polling2.poll(
        target=lambda: cluster.read_primaza_custom_object(
            version="v1alpha1",
            namespace="primaza-system",
            plural="servicecatalogs",
            name=catalog_name),
        check_success=lambda x: x is not None and registered_service_in_catalog_contains_service_class_identity(registered_service, sci, x),
        ignore_exceptions=(ApiException,),
        step=1,
        timeout=60)


@step(u'On Worker Cluster "{cluster_name}", ServiceCatalog "{catalog_name}" in application namespace "{namespace}" has "{registered_service}" with "{sci}"')
def check_on_worker_cluster_registered_service_contains_identity(context, cluster_name, catalog_name, registered_service, sci, namespace):
    cluster = context.cluster_provider.get_worker_cluster(cluster_name)

    polling2.poll(
        target=lambda: cluster.read_primaza_custom_object(
            version="v1alpha1",
            namespace=namespace,
            plural="servicecatalogs",
            name=catalog_name),
        check_success=lambda x: x is not None and registered_service_in_catalog_contains_service_class_identity(registered_service, sci, x),
        ignore_exceptions=(ApiException,),
        step=1,
        timeout=60)


@step(u'On Worker Cluster "{cluster_name}", Service Catalog "{catalog}" does not exist in "{namespace}"')
def on_worker_cluster_check_service_catalog_not_exists_in_application_namespace(context, cluster_name: str, catalog: str, namespace: str):
    cluster = context.cluster_provider.get_worker_cluster(cluster_name)
    polling2.poll(
        target=lambda: not cluster.primaza_custom_object_exists(namespace=namespace, plural="servicecatalogs", name=catalog),
        step=1,
        timeout=180)


@then(u'On Primaza Cluster "{cluster_name}", there are no ServiceCatalogs')
def on_primaza_cluster_no_service_catalogs_found(context, cluster_name):
    api_client = context.cluster_provider.get_primaza_cluster(cluster_name).get_api_client()
    cobj = client.CustomObjectsApi(api_client)

    response = cobj.list_namespaced_custom_object(
        group="primaza.io",
        version="v1alpha1",
        namespace="primaza-system",
        plural="servicecatalogs"
    )

    assert not response["items"]


@step(u'On Primaza Cluster "{cluster_name}", RegisteredService "{rs}" is available')
def on_primaza_cluster_check_registered_service_exists(context, cluster_name, rs):
    cluster = context.cluster_provider.get_primaza_cluster(cluster_name)
    name = substitute_scenario_id(context, rs)
    polling2.poll(
        target=lambda: cluster.primaza_custom_object_exists(
            version="v1alpha1",
            namespace="primaza-system",
            plural="registeredservices",
            name=name),
        step=1,
        timeout=60)


@step(u'On Primaza Cluster "{cluster_name}", RegisteredService "{rs}" is not available')
def on_primaza_cluster_check_registered_service_does_not_exists(context, cluster_name, rs):
    cluster = context.cluster_provider.get_primaza_cluster(cluster_name)
    name = substitute_scenario_id(context, rs)
    polling2.poll(
        target=lambda: not cluster.primaza_custom_object_exists(
            version="v1alpha1",
            namespace="primaza-system",
            plural="registeredservices",
            name=name),
        step=1,
        timeout=60)
