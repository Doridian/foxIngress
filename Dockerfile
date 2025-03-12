FROM golang:alpine AS builder

RUN apk --no-cache add upx

COPY . /go/src/app
WORKDIR /go/src/app
ENV CGO_ENABLED=0
RUN go mod download && go build -ldflags='-s -w' -trimpath -o proxy
RUN upx -9 proxy -o proxy-compressed

FROM scratch AS base
EXPOSE 80 443
ENV CONFIG_FILE=/config/config.yml
ENV PUID=1000
ENV PGID=1000
ENTRYPOINT [ "/proxy" ]

FROM base AS compressed
COPY --from=builder /go/src/app/proxy-compressed /proxy

FROM base AS uncompressed
COPY --from=builder /go/src/app/proxy /proxy
