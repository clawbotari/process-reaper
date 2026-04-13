FROM rockylinux:9

ARG GO_VERSION=1.26.2
ARG TARGETARCH

RUN set -eux; \
    case "${TARGETARCH}" in \
        amd64) GO_ARCH=amd64 ;; \
        arm64) GO_ARCH=arm64 ;; \
        *) echo "Unsupported TARGETARCH: ${TARGETARCH}"; exit 1 ;; \
    esac; \
    dnf -y install epel-release; \
    dnf -y install bash ca-certificates curl-minimal dpkg findutils git gzip procps-ng python3 rpm-build tar which; \
    update-ca-trust; \
    curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz" -o /tmp/go.tgz; \
    tar -C /usr/local -xzf /tmp/go.tgz; \
    /usr/local/go/bin/go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest; \
    cp /root/go/bin/nfpm /usr/local/bin/nfpm; \
    ln -sf /usr/local/go/bin/go /usr/local/bin/go; \
    ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt; \
    dnf clean all; \
    rm -rf /var/cache/dnf /tmp/go.tgz

ENV PATH="/usr/local/go/bin:/usr/local/bin:${PATH}"

WORKDIR /workspace
