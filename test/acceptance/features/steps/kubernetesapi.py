import polling2
from behave import then, step
from datetime import datetime
from kubernetes import client
from kubernetes.client.rest import ApiException
from steps.util import substitute_scenario_id


@step(u'On Cluster "{cluster_name}", {r_type} "{r_name}" in namespace "{ns}" state will eventually move to "{state}"')
@step(u'On Cluster "{cluster_name}", {r_type} "{r_name}" in namespace "{ns}" state will move to "{state}" in "{timeout:g}" seconds')
@step(u'On Primaza Cluster "{cluster_name}", {r_type} "{r_name}" state will eventually move to "{state}"')
@step(u'On Primaza Cluster "{cluster_name}", {r_type} "{r_name}" state will move to "{state}" in "{timeout:g}" seconds')
@step(u'On Primaza Cluster "{cluster_name}", {r_type} "{r_name}" in namespace "{ns}" state will eventually move to "{state}"')
@step(u'On Primaza Cluster "{cluster_name}", {r_type} "{r_name}" in namespace "{ns}" state will move to "{state}" in "{timeout:g}" seconds')
@step(u'On Worker Cluster "{cluster_name}", {r_type} "{r_name}" in namespace "{ns}" state will eventually move to "{state}"')
@step(u'On Worker Cluster "{cluster_name}", {r_type} "{r_name}" in namespace "{ns}" state will move to "{state}" in "{timeout:g}" seconds')
def on_cluster_check_state(context, cluster_name: str, r_type: str, r_name: str, state: str, ns: str = "primaza-system", timeout: float = 60):
    api_client = context.cluster_provider.get_cluster(cluster_name).get_api_client()
    cobj = client.CustomObjectsApi(api_client)
    name = substitute_scenario_id(context, r_name)
    namespace = substitute_scenario_id(context, ns)

    polling2.poll(
        target=lambda: cobj.get_namespaced_custom_object_status(
            group="primaza.io",
            version="v1alpha1",
            namespace=namespace,
            plural=get_resource(r_type),
            name=name).get("status", {}).get("state", None),
        ignore_exceptions=(ApiException,),
        check_success=lambda x: x is not None and x == state,
        step=1,
        timeout=float(timeout))


@then(u'On {_type} Cluster "{cluster_name}", {r_type} "{r_name}" last status condition has Type "{ctype}"')
@then(u'On {_type} Cluster "{cluster_name}", {r_type} "{r_name}" last status condition has Type "{ctype}"')
@then(u'On {_type} Cluster "{cluster_name}", {r_type} "{r_name}" in namespace "{ns}" last status condition has Type "{ctype}"')
def on_cluster_check_last_status_condition(context, _type: str, cluster_name: str, r_type: str, r_name: str, ctype: str, ns: str = "primaza-system"):
    api_client = context.cluster_provider.get_cluster(cluster_name).get_api_client()
    cobj = client.CustomObjectsApi(api_client)
    name = substitute_scenario_id(context, r_name)
    namespace = substitute_scenario_id(context, ns)

    ce_status = cobj.get_namespaced_custom_object_status(
        group="primaza.io",
        version="v1alpha1",
        namespace=namespace,
        plural=get_resource(r_type),
        name=name)
    ce_conditions = ce_status.get("status", {}).get("conditions", None)
    assert ce_conditions is not None and len(ce_conditions) > 0, "Cluster Environment status conditions are empty or not defined"

    last_applied = ce_conditions[0]
    for condition in ce_conditions[1:]:
        lat = datetime.fromisoformat(last_applied["last_transition_time"])
        cct = datetime.fromisoformat(condition["last_transition_time"])
        if cct > lat:
            last_applied = condition

    assert last_applied["type"] == ctype, f'Cluster Environment last condition type is not matching: wanted "{ctype}", found "{last_applied["type"]}"'


@step(u'On {_type} Cluster "{cluster_name}", {r_type} "{r_name}" status condition with Type "{ctype}" has Status "{cstatus}"')
@step(u'On {_type} Cluster "{cluster_name}", {r_type} "{r_name}" in namespace "{ns}" status condition with Type "{ctype}" has Status "{cstatus}"')
def on_cluster_check_status_condition(context, _type: str, cluster_name: str, r_type: str, r_name: str, ctype: str, cstatus: str, ns: str = "primaza-system"):
    api_client = context.cluster_provider.get_cluster(cluster_name).get_api_client()
    cobj = client.CustomObjectsApi(api_client)
    name = substitute_scenario_id(context, r_name)
    namespace = substitute_scenario_id(context, ns)

    ce_status = cobj.get_namespaced_custom_object_status(
        group="primaza.io",
        version="v1alpha1",
        namespace=namespace,
        plural=get_resource(r_type),
        name=name)
    ce_conditions = ce_status.get("status", {}).get("conditions", None)
    assert ce_conditions is not None and len(ce_conditions) > 0, "Cluster Environment status conditions are empty or not defined"

    for condition in ce_conditions:
        if condition["type"] == ctype:
            assert condition["status"] == cstatus, f'Cluster Environment condition with type "{ctype}" has status {condition["status"]}, wanted {cstatus}'
            return

    assert False, f"Cluster Environment does not have a condition with type {ctype}: {ce_conditions}"


@step(u'On {_type} Cluster "{cluster_name}", {r_type} "{r_name}" status condition with Type "{ctype}" has Reason "{creason}"')
@step(u'On {_type} Cluster "{cluster_name}", {r_type} "{r_name}" in namespace "{ns}" status condition with Type "{ctype}" has Reason "{creason}"')
def on_cluster_check_status_condition_reason(
        context, _type: str, cluster_name: str, r_type: str, r_name: str, ctype: str, creason: str, ns: str = "primaza-system"):
    api_client = context.cluster_provider.get_cluster(cluster_name).get_api_client()
    cobj = client.CustomObjectsApi(api_client)
    name = substitute_scenario_id(context, r_name)
    namespace = substitute_scenario_id(context, ns)

    try:
        polling2.poll(
            target=lambda: cobj.get_namespaced_custom_object_status(
                group="primaza.io",
                version="v1alpha1",
                namespace=namespace,
                plural=get_resource(r_type),
                name=name),
            check_success=lambda x: len([i for i in x.get("status", {}).get("conditions", {}) if i.get("type")
                                        and i["type"] == ctype and i.get("reason") and i["reason"] == creason]) == 1,
            step=1,
            timeout=60)
    except polling2.TimeoutException:
        pass


def get_resource(res: str) -> str:
    if res == "ClusterEnvironment":
        return "clusterenvironments"
    if res == "ServiceClaim":
        return "serviceclaims"
    if res == "ServiceClass":
        return "serviceclasses"
    if res == "ServiceBinding":
        return "servicebindings"
    if res == "RegisteredService":
        return "registeredservices"

    raise Exception(f"Resource {res} not known")


@then(u'On Primaza Cluster "{cluster_name}", ClusterEnvironment "{ce_name}" in namespace "{namespace}" state remains not present')
def on_primaza_cluster_check_state_not_change(context, cluster_name, ce_name, namespace="primaza-system", timeout=60):
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


def registered_service_in_catalog(rs_name, catalog):
    if "spec" in catalog and "services" in catalog["spec"]:
        for service in catalog["spec"]["services"]:
            if service["name"] == rs_name:
                return True
    return False


def registered_service_claimed_labels_in_catalog(rs_name, catalog):
    if "spec" in catalog and "claimedByLabels" in catalog["spec"]:
        for service in catalog["spec"]["claimedByLabels"]:
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


@step(u'On {_type} Cluster "{cluster_name}", {r_type} "{r_name}" state moves to "{state}"')
@step(u'On {_type} Cluster "{cluster_name}", {r_type} "{r_name}" in namespace "{ns}" state moves to "{state}"')
def on_cluster_claim_registered_service(
        context, _type: str, cluster_name: str, r_type: str, r_name: str, state: str, ns: str = "primaza-system"):
    api_client = context.cluster_provider.get_cluster(cluster_name).get_api_client()
    cobj = client.CustomObjectsApi(api_client)
    name = substitute_scenario_id(context, r_name)
    namespace = substitute_scenario_id(context, ns)

    cobj.patch_namespaced_custom_object_status(
        group="primaza.io",
        version="v1alpha1",
        namespace=namespace,
        plural=get_resource(r_type),
        name=name,
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


@step(u'On Primaza Cluster "{cluster_name}", ServiceCatalog "{catalog_name}" contain RegisteredService "{rs_name}" in claimedByLabels')
def on_primaza_cluster_check_service_catalog_claimed_by_labels_exists(context, cluster_name, catalog_name, rs_name):
    cluster = context.cluster_provider.get_primaza_cluster(cluster_name)

    polling2.poll(
        target=lambda: cluster.read_primaza_custom_object(
            version="v1alpha1",
            namespace="primaza-system",
            plural="servicecatalogs",
            name=catalog_name),
        check_success=lambda x: x is not None and registered_service_claimed_labels_in_catalog(rs_name, x),
        ignore_exceptions=(ApiException,),
        step=1,
        timeout=60)


@step(u'On {_type} Cluster "{cluster_name}", Service Class "{sc_class}" exists in "{serviceclass_namespace}"')
def on_cluster_check_service_class_exists_in_serviceclass_namespace(context, _type: str, cluster_name: str, sc_class: str, serviceclass_namespace: str):
    cluster = context.cluster_provider.get_cluster(cluster_name)
    name = substitute_scenario_id(context, sc_class)

    polling2.poll(
        target=lambda: cluster.primaza_custom_object_exists(
            version="v1alpha1",
            namespace=serviceclass_namespace,
            plural="serviceclasses",
            name=name),
        step=1,
        timeout=60)


@step(u'On {_type} Cluster "{cluster_name}", Service Binding "{service_binding}" exists in "{app_namespace}"')
def on_cluster_check_service_binding_exists_in_application_namespace(context, _type: str, cluster_name: str, service_binding: str, app_namespace: str):
    cluster = context.cluster_provider.get_cluster(cluster_name)
    polling2.poll(
        target=lambda: cluster.primaza_custom_object_exists(
            version="v1alpha1",
            namespace=app_namespace,
            plural="servicebindings",
            name=service_binding),
        step=1,
        timeout=60)


@step(u'On {_type} Cluster "{cluster_name}", Service Binding "{service_binding}" does not exist in "{namespace}"')
def on_cluster_check_service_bindings_not_exists_in_application_namespace(context, _type: str, cluster_name: str, service_binding: str, namespace: str):
    cluster = context.cluster_provider.get_cluster(cluster_name)

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
