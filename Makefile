CC=go build
NAME=schneller_whatsapp
default: build

build:
	$(CC)

run:
	go run .

arm:
	env CGO_ENABLED=1 GOOS=linux GOARCH=arm64 $(CC) -o $(NAME)-arm64

update:
	go get -u go.mau.fi/whatsmeow
	go mod tidy

update-all:
	go get -u
	go mod tidy

clean: 
	git clean -fX
