#!/bin/env bash

__primaza_get_kind_kubeconfig() {
    kind get kubeconfig --name "$(kind get clusters | grep -E "$1\$" | fzf -1)"
}

__primaza_get_kind_cluster_noninteractive() {
   kind get clusters | grep -E "$1\$" | head -n 1
}

__primaza_get_kind_cluster() {
   kind get clusters | grep -E "$1\$" | fzf -1
}

__primaza_get_kind_kubeconfig_by_name()
(
    if kfg="$(kind get kubeconfig --name "$1")" ; then
        printf '%s' "$kfg"
        return 0
    fi

    return 1
)

__primaza_get_cluster_kubeconfig_path()
(
    kfg="/tmp/kc-primaza-$1"
    printf '%s' "$kfg"
)

__primaza_export_env()
(
    # retrieve the cluster matching provided input
    c=$(__primaza_get_kind_cluster_noninteractive "$1")
    if [ "$c" = "" ]; then
        printf "can not find kind cluster %s\n" "$1" >&2
        return 1
    fi

    # retrieve kubeconfig for the cluster
    if ! kfg="$(__primaza_get_kind_kubeconfig_by_name "$c")" ; then
        printf "can not find kind cluster %s\n" "$c" >&2
        return 1
    fi

    # write the kubeconfig file and print the configuration
    kfgpath=$(__primaza_get_cluster_kubeconfig_path "$c")
    printf '%s' "$kfg" >! "$kfgpath"
    printf "export KUBECONFIG=%s\n" "$kfgpath"

    # if the shell is not interactive inform the user on stderr
    [ ! -t 1 ] && printf "exported context for cluster '%s' (KUBECONFIG=%s)\n" "$c" "$kfgpath" >&2
)

__primaza_kind_export_kubeconfig()
(
    c=$(__primaza_get_kind_cluster "$1")
    kfg=$(__primaza_get_cluster_kubeconfig_path "$c")
    __primaza_get_kind_kubeconfig "$c" >! "$kfg"
)

__primaza_wait_for_controller_logs()
(
    while true; do
        {
        c=$(__primaza_get_kind_cluster_noninteractive "main") && \
            kfg=$(__primaza_get_kind_kubeconfig_by_name "$c") && \
            kubectl wait pods \
                --for=jsonpath='{.status.phase}'=Running \
                -n primaza-system \
                -l control-plane=controller-manager \
                --kubeconfig <(echo "$kfg") && \
            kubectl logs -f \
                -n primaza-system \
                --tail=-1 \
                -l control-plane=controller-manager \
                --kubeconfig <(echo "$kfg")
        } || ( echo "error getting logs: waiting 5 seconds and retrying..." && sleep 5)
    done
)

alias pek="__primaza_export_env"
alias primaza-env="__primaza_export_env"
alias pgc="__primaza_get_kind_cluster"

# main
alias epm='eval $(__primaza_export_env main) && kubectl config set-context --current --namespace primaza-system'
alias pclo="kubectl logs -f -n primaza-system -l control-plane=controller-manager --kubeconfig=<(__primaza_get_kind_kubeconfig main) --tail=-1"
alias pwclo="__primaza_wait_for_controller_logs"
alias pmgpo="kubectl get pods -n primaza-system --kubeconfig=<(__primaza_get_kind_kubeconfig main)"
alias pmgce="kubectl get clusterenvironments -n primaza-system --kubeconfig=<(__primaza_get_kind_kubeconfig main)"
alias pmgrs="kubectl get registeredservices -n primaza-system --kubeconfig <(__primaza_get_kind_kubeconfig main)"
alias pmgsg="kubectl get servicecatalog -n primaza-system --kubeconfig <(__primaza_get_kind_kubeconfig main)"
alias pmgss="kubectl get serviceclasses -n primaza-system --kubeconfig <(__primaza_get_kind_kubeconfig main)"
alias pmgsm="kubectl get serviceclaims -n primaza-system --kubeconfig <(__primaza_get_kind_kubeconfig main)"
alias pmgwrs="kubectl get registeredservices -n primaza-system --kubeconfig <(__primaza_get_kind_kubeconfig main) -w"
alias pmgwsg="kubectl get servicecatalog -n primaza-system --kubeconfig <(__primaza_get_kind_kubeconfig main) -w"
alias pmgwss="kubectl get serviceclasses -n primaza-system --kubeconfig <(__primaza_get_kind_kubeconfig main) -w"
alias pmgwsm="kubectl get serviceclaims -n primaza-system --kubeconfig <(__primaza_get_kind_kubeconfig main) -w"

# worker
alias epw='eval $(__primaza_export_env worker)'
alias epwa='eval $(__primaza_export_env worker) && kubectl config set-context --current --namespace applications'
alias epws='eval $(__primaza_export_env worker) && kubectl config set-context --current --namespace services'
alias pwgpoall="kubectl get pods --all-namespaces --kubeconfig=<(__primaza_get_kind_kubeconfig worker)"
alias pwgpoa="kubectl get pods -n applications --kubeconfig=<(__primaza_get_kind_kubeconfig worker)"
alias pwgpos="kubectl get pods -n services --kubeconfig=<(__primaza_get_kind_kubeconfig worker)"
alias pwgpo="kubectl get pods --kubeconfig=<(__primaza_get_kind_kubeconfig worker)"
