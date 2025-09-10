# WebRat
WebRat is a light-weight BurpSuite-inspired web proxy built in Go using the Fyne GUI framework. It was developed as a learning project to learn about web security, networking, and Go. 
I made this project a while ago, and although I no longer program in Go, WebRat shows that I can take on ambitious, self-directed projects and see them through to a working prototype, even if it means stepping out of my comfort zone. 
This project is not production-ready, but it demonstrates ambition, self-motivation, and hands-on learning. 

## Features
- Intercept and inspect HTTP requests/responses  
- Modify traffic on the fly
- Bruteforce directories or username/password fields

## Notes
- Only tested on Kali Linux and Linux Mint  
- GUI requires X11 libraries
- The compilation process takes a while (around 5 minutes) because it needs to compile and link the GUI library, which includes C bindings for graphical functionality.
- May contain bugs, I haven't had the time to thoroughly test the program. 

## Technical Setup

### 1. Install Go
```
sudo apt update
sudo apt install golang
```

### 2. Install dependencies
```
sudo apt install xorg-dev libx11-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev libgl1-mesa-dev libglu1-mesa-dev
```

### 3. Clone the project
```
git clone https://github.com/Sviatko124/WebRat
cd WebRat
```

### 4. Compile into a single binary
```
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o WebRat cmd/main.go
```

The compilation process may take a long time due to the Fyne GUI framework, but after that, you should be left with a single binary. 
The usage of the program should be intuitive enough, as it isn't nearly as complex as Burp Suite. 
To use the proxy, use a browser extension to easily manage proxies (like FoxyProxy) and add the following proxy:
```
IP: localhost
PORT: 9090
```
