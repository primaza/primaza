ARG MDBOOK_VERSION=v0.4.30

FROM peaceiris/mdbook:${MDBOOK_VERSION}-rust as plugins
RUN apk add musl-dev && \
    rustup target add x86_64-unknown-linux-musl
RUN cargo install mdbook-bash-tutorial --root /tmp/plugins/ --target x86_64-unknown-linux-musl

FROM peaceiris/mdbook:${MDBOOK_VERSION}
COPY --from=plugins /tmp/plugins/bin/mdbook-bash-tutorial /bin/mdbook-bash-tutorial
