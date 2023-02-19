.PHONY: build clean deploy undeploy

build:
	env GOARCH=amd64 GOOS=linux go build -ldflags="-s -w" -o bin/event/receiver event/receiver/main.go
	env GOARCH=amd64 GOOS=linux go build -ldflags="-s -w" -o bin/event/action event/action/main.go

clean:
	rm -rf ./bin

deploy: clean build
	yarn sls deploy --verbose

undeploy:
	yarn sls remove --verbose
