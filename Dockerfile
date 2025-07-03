# Build stage
FROM golang:latest AS build

RUN apt-get update && \
  apt-get upgrade -y ca-certificates

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . /src

RUN CGO_ENABLED=0 go test ./...

# Build static
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -v \
    -ldflags "-extldflags '-static' -s -w" \
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

COPY --from=build /etc/ssl/certs /etc/ssl/certs
COPY --from=build /src/mappings /mappings
COPY --from=upx   /koralmapper      /koralmapper

ENTRYPOINT [ "/termmapper" ]

LABEL maintainer="korap@ids-mannheim.de"
LABEL description="Docker Image for Koral-Mapper"
LABEL repository="https://github.com/KorAP/Koral-Mapper"

# docker build -f Dockerfile -t korap/koral-mapper:latest .
# docker run --rm --network host korap/koral-mapper:latest -m /mappings/*.yaml
# docker save -o korap-koral-mapper-latest.tar korap/koral-mapper:latest