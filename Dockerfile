# Use Ubuntu as base image for better DevBox compatibility
FROM ubuntu:24.04

# Avoid prompts from apt
ENV DEBIAN_FRONTEND=noninteractive

# Install dependencies required by DevBox and Nix
RUN apt-get update && apt-get install -y \
    curl \
    git \
    xz-utils \
    ca-certificates \
    sudo \
    && rm -rf /var/lib/apt/lists/*

# Create a non-root user for DevBox
RUN useradd -m -s /bin/bash devbox && \
    echo "devbox ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# Switch to devbox user
USER devbox
WORKDIR /home/devbox

# Install Nix (required by DevBox)
RUN curl -L https://nixos.org/nix/install | sh -s -- --no-daemon

# Add Nix to PATH for subsequent commands
ENV PATH="/home/devbox/.nix-profile/bin:${PATH}"

# Install DevBox
RUN curl -fsSL https://get.jetify.com/devbox | bash

# Add DevBox to PATH
ENV PATH="/home/devbox/.local/bin:${PATH}"

# Copy DevBox configuration files
COPY --chown=devbox:devbox devbox.json devbox.lock ./

# Initialize DevBox and install all packages
# This will download and cache all the packages from your devbox.json
RUN devbox install

# Set up shell environment
RUN echo 'eval "$(devbox shellenv)"' >> ~/.bashrc

# Default command to enter DevBox shell
CMD ["devbox", "shell"]
