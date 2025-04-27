CC=go build
NAME=whatsup-poll
default: build

build:
	$(CC)

arm:
	env CGO_ENABLED=1 GOOS=linux GOARCH=arm64 $(CC) -o $(NAME)-arm64

clean: 
	git clean -fX
