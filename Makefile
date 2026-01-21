CC=go build
NAME=schneller-whatsapp
SRCDIR=./src
default: build

build:
	$(CC) -o $(NAME) $(SRCDIR)

run:
	go run $(SRCDIR)

arm:
	env CGO_ENABLED=1 GOOS=linux GOARCH=arm64 $(CC) -o $(NAME)-arm64

update:
	go get -u go.mau.fi/whatsmeow
	go mod tidy

update-all:
	go get -u
	go mod tidy

deploy:
	rsync $(NAME) jojo:$(NAME)

test-remote:
	ssh jojo 'cd $(NAME) && ./test.sh'

upgrade: update build deploy

cicd: upgrade test-remote

clean: 
	git clean -fX
