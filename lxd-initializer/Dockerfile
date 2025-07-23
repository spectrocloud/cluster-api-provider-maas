FROM ubuntu:22.04

# Install dependencies
RUN apt-get update && apt-get install -y \
    curl \
    jq \
    snapd \
    zfsutils-linux \
    && rm -rf /var/lib/apt/lists/*

# Install LXD
RUN snap install lxd

# Copy our LXD initializer binary
COPY lxd-initializer /usr/local/bin/
RUN chmod +x /usr/local/bin/lxd-initializer

# Set the entrypoint
ENTRYPOINT ["/usr/local/bin/lxd-initializer"] 