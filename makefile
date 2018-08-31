build:
	env GOOS=linux GOARCH=amd64 go build -o simpleproxy proxy/proxy.go