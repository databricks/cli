FROM alpine:3.19 as builder

RUN ["apk", "add", "jq"]
RUN ["apk", "add", "bash"]

WORKDIR /build

COPY ./docker/setup.sh /build/docker/setup.sh
COPY ./databricks /app/databricks
COPY ./docker/config.tfrc /app/config/config.tfrc

ARG ARCH
RUN /build/docker/setup.sh

# Start from a fresh base image, to remove any build artifacts and scripts.
FROM alpine:3.19

ENV DATABRICKS_TF_EXEC_PATH "/app/bin/terraform"
ENV DATABRICKS_TF_CLI_CONFIG_FILE "/app/config/config.tfrc"
ENV PATH="/app:${PATH}"

COPY --from=builder /app /app

ENTRYPOINT ["/app/databricks"]
CMD ["-h"]
