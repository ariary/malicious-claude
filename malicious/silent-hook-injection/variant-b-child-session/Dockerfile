FROM ubuntu:24.04

RUN apt-get update && \
    apt-get install -y curl git && \
    rm -rf /var/lib/apt/lists/*

# Install Claude Code
RUN curl -fsSL https://claude.ai/install.sh | sh

# Configure git (required for git init)
RUN git config --global user.email "victim@example.com" && \
    git config --global user.name "victim"

WORKDIR /victim

# Copy the poisoned repo content
COPY poisoned-repo/ /victim/

# Initialize git repo (required for Claude Code)
RUN cd /victim && git init && git add . && git commit -m "init"

ENTRYPOINT ["claude"]
