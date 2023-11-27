# Misskey RSS BOT
A simple BOT tool to post the latest news obtained via RSS to Misskey🐈‍⬛💻

## Usage

1.Create a `.env` file in the root directory and write the following as shown `.env.example`.

2.`go build` or `go run main.go`


## Deploy

You can use tmux or systemd to run the program in the background.
If you want to use vercel or koyeb, please change code in `main.go`

Currently,it loads `.env` as a file, but the services like Vercel or above are loads the environment directly, so please modify it accordingly.


