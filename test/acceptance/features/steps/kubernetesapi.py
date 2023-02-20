import polling2
from behave import then
from datetime import datetime
from kubernetes import client


@then(u'On Primaza Cluster "{cluster}", ClusterEnvironment "{ce_name}" state will eventually move to "{state}"')
@then(u'On Primaza Cluster "{cluster}", ClusterEnvironment "{ce_name}" state will move to "{state}" in "{timeout}" seconds')
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
def on_primaza_cluster_check_status_condition(context, cluster, ce_name, ctype):
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


@then(u'On Primaza Cluster "{cluster}", RegisteredService "{rs_name}" state will eventually move to "{state}"')
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
