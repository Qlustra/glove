ARG GO_VERSION=1.26
FROM golang:${GO_VERSION}-bookworm

ARG TASK_VERSION=v3.44.1
ARG YQ_VERSION=v4.44.6
ARG USER_NAME=dev
ARG USER_UID=1000
ARG USER_GID=1000

ARG TOOLING_DIR=/tooling

ENV TOOLING_DIR=${TOOLING_DIR}
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        gpg \
        jq \
        ripgrep \
        net-tools \
        tar \
        tree \
        gzip

RUN ARCH_RAW="$(dpkg --print-architecture)" \
    && case "${ARCH_RAW}" in \
         amd64) ARCH_DL=amd64 ;; \
         arm64) ARCH_DL=arm64 ;; \
         *) echo "Unsupported architecture: ${ARCH_RAW}" >&2; exit 1 ;; \
       esac \
    && curl -fsSL "https://github.com/go-task/task/releases/download/${TASK_VERSION}/task_linux_${ARCH_DL}.tar.gz" \
        | tar -xz -C /usr/local/bin task \
    && curl -fsSL "https://github.com/mikefarah/yq/releases/download/${YQ_VERSION}/yq_linux_${ARCH_DL}" \
        -o /usr/local/bin/yq \
    && chmod +x /usr/local/bin/yq

RUN set -eux; \
    install_root="/usr/local/share/opencode"; \
    mkdir -p "${install_root}"; \
    export HOME="${install_root}"; \
    curl -fsSL https://opencode.ai/install | bash; \
    opencode_bin=""; \
    if [ -x "${install_root}/.opencode/bin/opencode" ]; then \
      opencode_bin="${install_root}/.opencode/bin/opencode"; \
    elif [ -x "${install_root}/.local/bin/opencode" ]; then \
      opencode_bin="${install_root}/.local/bin/opencode"; \
    elif [ -x "${install_root}/bin/opencode" ]; then \
      opencode_bin="${install_root}/bin/opencode"; \
    fi; \
    if [ -z "${opencode_bin}" ]; then \
      echo "opencode install failed: binary not found under ${install_root}" >&2; \
      exit 1; \
    fi; \
    ln -sf "${opencode_bin}" /usr/local/bin/opencode; \
    chmod -R a+rX "${install_root}"

RUN GOBIN=/usr/local/bin go install golang.org/x/tools/cmd/goimports@latest \
    && GOBIN=/usr/local/bin go install honnef.co/go/tools/cmd/staticcheck@latest \
    && GOBIN=/usr/local/bin go install golang.org/x/vuln/cmd/govulncheck@latest \
    && GOBIN=/usr/local/bin go install gotest.tools/gotestsum@latest

RUN if ! getent group "${USER_GID}" >/dev/null; then groupadd --gid "${USER_GID}" "${USER_NAME}"; fi \
    && if ! id -u "${USER_NAME}" >/dev/null 2>&1; then \
         useradd --uid "${USER_UID}" --gid "${USER_GID}" --create-home --shell /bin/bash "${USER_NAME}"; \
       fi \
    && install -d -m 0755 -o "${USER_UID}" -g "${USER_GID}" /workspace

WORKDIR /workspace
USER ${USER_NAME}

CMD ["bash"]
