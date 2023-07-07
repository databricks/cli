ARG BASE_IMAGE_ARCH=base
FROM gcr.io/distroless/$BASE_IMAGE_ARCH
COPY databricks /
ENTRYPOINT ["/databricks"]
