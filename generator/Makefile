all: run
run: build
	bin/generator

build:
	go build -o bin/generator -mod=vendor .

tidy:
	go mod tidy
	go mod vendor