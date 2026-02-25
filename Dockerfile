# Build stage
FROM golang:latest AS build

RUN apt-get update && \
  apt-get upgrade -y ca-certificates

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . /src

RUN CGO_ENABLED=0 go test ./...

ARG BUILDDATE="[unset]"
ARG BUILDCOMMIT="[unset]"
ARG BUILDVERSION="EARLY"

# Build static
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -v \
    -ldflags "-X github.com/KorAP/Koral-Mapper/config.Buildtime=${BUILDDATE} -X github.com/KorAP/Koral-Mapper/config.Buildhash=${BUILDCOMMIT} -X github.com/KorAP/Koral-Mapper/config.Version=${BUILDVERSION} -extldflags '-static' -s -w" \
    --trimpath \
    -o /src/koralmapper ./cmd/koralmapper/

FROM gruebel/upx:latest AS upx

COPY --from=build /src/koralmapper /koralmapper-big

# Compress the binary and copy it to final image
RUN upx --best --lzma -o /koralmapper /koralmapper-big

# Main stage
FROM scratch AS final

WORKDIR /

EXPOSE 5725

ARG BUILDVERSION="EARLY"
ARG BUILDCOMMIT="[unset]"
ARG BUILDDATE="1970-01-01T00:00:00Z"

COPY --from=build /etc/ssl/certs /etc/ssl/certs
COPY --from=build /src/mappings /mappings
COPY --from=upx   /koralmapper      /koralmapper

ENTRYPOINT [ "/koralmapper" ]

LABEL maintainer="korap@ids-mannheim.de"
LABEL description="Docker Image for Koral-Mapper"
LABEL repository="https://github.com/KorAP/Koral-Mapper"
LABEL org.opencontainers.image.title="Koral-Mapper"
LABEL org.opencontainers.image.description="Docker Image for Koral-Mapper"
LABEL org.opencontainers.image.url="https://github.com/KorAP/Koral-Mapper"
LABEL org.opencontainers.image.source="https://github.com/KorAP/Koral-Mapper"
LABEL org.opencontainers.image.created="${BUILDDATE}"
LABEL org.opencontainers.image.revision="${BUILDCOMMIT}"
LABEL org.opencontainers.image.version="${BUILDVERSION}"

# docker build -f Dockerfile -t korap/koral-mapper:latest .
# docker run --rm --network host korap/koral-mapper:latest -m /mappings/*.yaml
# docker save -o korap-koral-mapper-latest.tar korap/koral-mapper:latest