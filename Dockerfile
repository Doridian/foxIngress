FROM golang:alpine AS builder
COPY . /go/src/app
WORKDIR /go/src/app
ENV CGO_ENABLED=0
RUN go mod download && go build -o proxy

FROM scratch
COPY --from=builder /go/src/app/proxy /proxy
EXPOSE 80 443
ENV CONFIG_FILE=/etc/config.yml
ENV HTTP_ADDR=:80
ENV HTTPS_ADDR=:443
ENTRYPOINT [ "/proxy" ]
