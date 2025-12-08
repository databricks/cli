FROM alpine:3.22 as builder

RUN ["apk", "add", "jq"]
RUN ["apk", "add", "bash"]

WORKDIR /build

COPY ./setup.sh /build/docker/setup.sh
COPY ./config.tfrc /app/config/config.tfrc

ARG ARCH
ARG CLI_VERSION=0.279.0
RUN /build/docker/setup.sh

# Start from a fresh base image, to remove any build artifacts and scripts.
FROM alpine:3.22

ENV DATABRICKS_TF_EXEC_PATH "/app/bin/terraform"
ENV DATABRICKS_TF_CLI_CONFIG_FILE "/app/config/config.tfrc"
ENV DATABRICKS_TF_VERSION "1.14.0"
ENV PATH="/app:${PATH}"

COPY --from=builder /app /app

ENTRYPOINT ["/app/databricks"]
CMD ["-h"]
