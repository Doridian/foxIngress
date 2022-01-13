FROM golang:latest AS builder
COPY . /go/src/app
WORKDIR /go/src/app
RUN go mod download && go build -o proxy

FROM scratch
COPY --from=builder /go/src/app/proxy /proxy
EXPOSE 80 443
ENTRYPOINT [ "/proxy" ]
