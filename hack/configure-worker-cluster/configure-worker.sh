#!/bin/env sh

## References
## cfr. https://kubernetes.io/docs/reference/access-authn-authz/certificate-signing-requests/#normal-user

[ -z "$DIR" ] && DIR=$(dirname "$(readlink -f "$0")")
CERTS_DIR="$DIR/certs"

generate_csr() {
    mkdir "$CERTS_DIR"

    ( cd "$CERTS_DIR" && \
        openssl req \
            -nodes \
            -newkey rsa:2048 \
            -keyout primaza.key \
            -out primaza.csr \
            -subj "/CN=Primaza/O=Primaza" )
}

register_primaza_user() {
    cat <<EOF | kubectl apply -f -
apiVersion: certificates.k8s.io/v1
kind: CertificateSigningRequest
metadata:
  name: primaza
spec:
  request: $(base64 -w0 < "$CERTS_DIR"/primaza.csr)
  signerName: kubernetes.io/kube-apiserver-client
  expirationSeconds: 864000  # ten days
  usages:
  - client auth
EOF

    kubectl certificate approve primaza
    kubectl get csr primaza -o jsonpath='{.status.certificate}'| base64 -d > "$CERTS_DIR"/primaza.crt
}

configure_primaza_user() {
    kubectl create role primaza-pods \
        --verb=create \
        --verb=get \
        --verb=list \
        --verb=update \
        --verb=delete \
        --resource=pods

    kubectl create rolebinding primaza-pods-primaza \
        --role=primaza-pods \
        --user=primaza
}

bake_kubeconfig() {
    kubectl config view --flatten > "$KUBECONFIG""-primaza"

    kubectl config set-credentials primaza \
        --client-key="$CERTS_DIR"/primaza.key \
        --client-certificate="$CERTS_DIR"/primaza.crt \
        --embed-certs=true \
        --kubeconfig "$KUBECONFIG""-primaza"
}

bake_secret() {
    secret_file="$DIR"/secret-config.yaml
    config_file="$KUBECONFIG""-primaza"

    cat << EOF > "$secret_file"
apiVersion: v1
kind: Secret
metadata:
    name: primaza-worker-kubeconfig
data:
    kubeconfig: $(base64 -w0 < "$config_file")
EOF
}

test_connection() {
    kubectl get all > /dev/null
}

main() {
    test_connection || exit 1
    [ ! -f "$CERTS_DIR/primaza.csr" ] && [ ! -f "$CERTS_DIR/primaza.key" ] && generate_csr

    register_primaza_user && \
        configure_primaza_user && \
        bake_kubeconfig && \
        bake_secret
}

main
