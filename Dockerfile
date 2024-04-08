FROM golang:alpine as builder

RUN apk add jq

WORKDIR /build

# # Copy repo to wd
# COPY . .

# # Build CLI binary
# RUN go mod download && go mod verify
# RUN CGO_ENABLED=0 go build -o /build/databricks

COPY ./docker/setup.sh ./docker/setup.sh
COPY ./databricks ./databricks

ENV BUILD_ARCH "arm64"
RUN ./docker/setup.sh

# Construct final image
FROM alpine:3.19



# # COPY --from=builder /build/databricks /app/databricks
# # COPY --from=builder /build/bundle/internal/tf/codegen/tmp/bin/terraform /app/terraform/terraform
# # COPY --from=builder /build/bundle/internal/tf/codegen/tmp/

# # CONTINUE: stop using the Go script. It does not work. Instead rely on a new shell
# # script that I write.
