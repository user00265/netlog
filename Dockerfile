# syntax=docker/dockerfile:1

# NetLog container image.
#
# This Dockerfile consumes a *prebuilt* netlog binary laid out per platform at
# $TARGETPLATFORM/netlog (the embedded SPA is already inside it). GoReleaser's
# dockers_v2 arranges the build context that way for the release images; the
# `testing` workflow stages the snapshot binary into the same layout. There is
# deliberately no in-image compilation, which keeps multi-arch builds
# emulation-free and fast.
#
# Runtime base is a Docker Hardened Image (dhi.io/static): a minimal,
# continuously-patched base for static binaries that ships CA certificates
# (needed for the outbound HTTPS calls to QRZ, HamQTH, the cty.xml mirror, and
# the OIDC issuer) and a nonroot user (uid 65532), with no shell or package
# manager. Pulling from dhi.io requires `docker login dhi.io` (a free Docker
# account); end users only pull the finished image from ghcr.io and need no DHI
# access.

# Throwaway helper, pinned to the build platform so it runs natively rather than
# under QEMU. It only produces an empty directory we copy in with the right
# ownership; nothing from this stage ships in the final image (which is 100%
# DHI), so it intentionally uses a plain upstream busybox to avoid a second
# registry login for a discarded layer.
FROM --platform=$BUILDPLATFORM busybox:stable AS prep
RUN mkdir -p /seed/data

# Bump this dated tag via your image-update automation (Renovate/Dependabot);
# DHI publishes immutable date-stamped tags rather than a rolling `latest`.
FROM dhi.io/static:20260611-alpine3.24
ARG TARGETPLATFORM

# Pre-create /data owned by the nonroot user (uid 65532) so a named volume
# mounted here is writable on first use — Docker seeds a fresh volume from the
# ownership of the image path, so NetLog can create its SQLite database and
# cty.xml cache without a manual chown.
COPY --from=prep --chown=65532:65532 /seed/data /data
COPY --chown=65532:65532 --chmod=0755 $TARGETPLATFORM/netlog /app/netlog

WORKDIR /app
USER 65532:65532
EXPOSE 8080
VOLUME ["/data"]

# If /app/config.yaml is mounted it is used; otherwise NetLog falls back to its
# built-in defaults overridden by NETLOG_* environment variables.
ENTRYPOINT ["/app/netlog", "-config", "/app/config.yaml"]
