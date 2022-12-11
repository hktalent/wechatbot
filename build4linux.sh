docker run --rm -it -v $PWD:/app golang:latest /bin/bash -c "cd /app/; CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags \"-linkmode external -extldflags '-static' -s -w\" -o wechatbot_linux_amd64 main.go ; exit"

