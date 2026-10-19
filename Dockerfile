FROM alpine:3.22@sha256:55ae5d250caebc548793f321534bc6a8ef1d116f334f18f4ada1b2daad3251b2

ENV PATH="/app:${PATH}"

COPY ./databricks /app/databricks

ENTRYPOINT ["/app/databricks"]
CMD ["-h"]
