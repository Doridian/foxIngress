FROM golang:alpine AS builder
COPY . /go/src/app
WORKDIR /go/src/app
ENV CGO_ENABLED=0
RUN go mod download && go build -o proxy

FROM scratch
COPY --from=builder /go/src/app/proxy /proxy
EXPOSE 80 443
ENV CONFIG_FILE=/config.yml
ENTRYPOINT [ "/proxy" ]
