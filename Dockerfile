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
    -o /src/termmapper ./cmd/termmapper/

FROM gruebel/upx:latest AS upx

COPY --from=build /src/termmapper /termmapper-big

# Compress the binary and copy it to final image
RUN upx --best --lzma -o /termmapper /termmapper-big

# Main stage
FROM scratch AS final

WORKDIR /

EXPOSE 5725

COPY --from=build /etc/ssl/certs /etc/ssl/certs
COPY --from=build /src/mappings /mappings
COPY --from=upx   /termmapper      /termmapper

ENTRYPOINT [ "/termmapper" ]

LABEL maintainer="korap@ids-mannheim.de"
LABEL description="Docker Image for KoralPipe-TermMapper"
LABEL repository="https://github.com/KorAP/KoralPipe-TermMapper"

# docker build -f Dockerfile -t korap/koralpipe-termmapper:latest .
# docker run --rm --network host korap/koralpipe-termmapper:latest -m /mappings/*.yaml
# docker save -o korap-koralpipe-termmapper-latest.tar korap/koralpipe-termmapper:latest