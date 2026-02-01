BINARY=email-server.exe

all: build

build:
	go build -o $(BINARY) ./cmd/email-server

clean:
	rm -f $(BINARY)
