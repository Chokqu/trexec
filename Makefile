.PHONY: all test race bench fuzz cross-compile build-examples clean

all: test race cross-compile

test:
	go test -v -count=1 ./...

race:
	go test -v -race -count=1 ./...

bench:
	go test -bench=. -benchmem -count=1 .

fuzz:
	go test -fuzz=FuzzContextCancellation -fuzztime=5s .
	go test -fuzz=FuzzCommandExecution -fuzztime=5s .

cross-compile:
	GOOS=windows go test -c -o /dev/null .
	GOOS=windows go test -c -o /dev/null ./supervisor
	GOOS=windows go test -c -o /dev/null ./telemetry
	GOOS=windows go test -c -o /dev/null ./watcher
	GOOS=windows go test -c -o /dev/null ./cobraexec
	GOOS=linux go test -c -o /dev/null .
	GOOS=linux go test -c -o /dev/null ./supervisor
	GOOS=linux go test -c -o /dev/null ./telemetry
	GOOS=linux go test -c -o /dev/null ./watcher
	GOOS=linux go test -c -o /dev/null ./cobraexec
	GOOS=darwin go test -c -o /dev/null .

build-examples:
	mkdir -p bin
	go build -o bin/example_basic ./examples/basic
	go build -o bin/example_devserver ./examples/devserver
	go build -o bin/example_supervisor ./examples/supervisor
	go build -o bin/example_watcher ./examples/watcher
	go build -o bin/example_telemetry ./examples/telemetry
	go build -o bin/example_cobra ./examples/cobra

clean:
	rm -rf bin
