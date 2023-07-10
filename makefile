#------------------------------------#
#  Build                             #
#    usage: make build VERSION=1.2.3 #
#------------------------------------#

# Overridable version number
VERSION?=dev

# inject the version number into the Version variable
flags=-X 'github.com/inveracity/ssh-tunnel/cmd.Version=$(VERSION)'

.PHONY: build
build:
	@echo "Building..."
	@go mod tidy
	@CGO_ENABLED=0 go build -ldflags "$(flags)" -o ssh-tunnel cmd/ssh-tunnel/main.go

.PHONY: install
install: build
	@echo "Installing..."
	sudo mv -f ssh-tunnel /usr/local/bin/ssh-tunnel

.PHONY: zip
zip: build
	@echo "Zipping..."
	@mkdir -p dist
	@zip -j dist/ssh-tunnel_linux_amd64.zip ssh-tunnel
	@rm ssh-tunnel
