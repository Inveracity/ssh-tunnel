# SSH Tunnel

https://user-images.githubusercontent.com/8753040/252415448-7dafba47-13cc-4669-b891-1b7f9e9935ed.mp4

> this could've been a small bash script 💩

If you are like me and can't remember the ssh tunnel command to reach your services behind a firewall

as in:

```sh
ssh -L 8080:localhost:8080 user@server -N
```

Then this highly overengineered tool will help you to worry about it again.

Write a config specifying the tunnel(s) needed:

> ℹ️ **TIP** see the `example.config.hcl` file

```hcl
// tunnel 1
tunnel {
    user = "user"
    local {
        port = 8080
    }
    remote {
        host = "server:22"
        port = 8080
    }
}

// tunnel 2
tunnel {
    user = "user"
    local {
        port = 8081
    }
    remote {
        host = "server:22"
        port = 8081
    }
}
```

and run

```sh
ssh-tunnel

# Remote Host     Tunnel
# user@server:22  127.0.0.1:8080 -> 127.0.0.1:8080
# user@server:22  127.0.0.1:8081 -> 127.0.0.1:8081

```

```sh
$ ssh-tunnel --help

#  -config string
#        The config file to use (default "config.hcl")
#  -no-color
#        Disable color output
#  -version
#        Print the version and exit
```

# Prerequisites

`ssh-tunnel` uses your SSH Agent (_SSH_AUTH_SOCK_) to create the tunnels, it does not work without it!

Here's a quick guide to setting up an SSH Agent on linux (and WSL2)

```sh
sudo apt update
sudo apt install keychain
echo "eval `keychain --eval --agents ssh id_rsa`" >> ~/.profile
source ~/.profile
```

# Installation

Either download it from the [releases page](https://github.com/Inveracity/ssh-tunnel/releases)

or install it with go

```sh
go install github.com/inveracity/ssh-tunnel/cmd/ssh-tunnel@latest
ssh-tunnel --version
```

# Build

```sh
make ssh-tunnel
# ./ssh-tunnel --help
```

or `make install` to install the binary directly into `/usr/local/bin`

# Limitations

It does not work on Windows except WSL2, and is not tested on macOS.
