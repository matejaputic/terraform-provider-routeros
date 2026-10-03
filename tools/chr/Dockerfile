# QEMU runs inside OrbStack Linux; no privileged container or nested KVM.
FROM debian@sha256:3783cc01769c7b2b1b83a5c5ad96c815348e28ed7da68e2e3687004faa906251
RUN apt-get update && apt-get install -y --no-install-recommends qemu-system-x86=1:7.2+dfsg-7+deb12u18+b3 && rm -rf /var/lib/apt/lists/*
USER 65534:65534
ENTRYPOINT ["qemu-system-x86_64"]
