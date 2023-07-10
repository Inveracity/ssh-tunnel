tunnel {
    user = "user"

    local {
        port = 8080 // Local port to open
    }

    remote {
        host = "server:22" // Remote host to connect to
        port = 8080 // The port on the remote host to tunnel back to the local port
    }
}

// Multiple tunnels can be defined
/*
tunnel {
    user = "user"

    local {
        port = 8081 // another port to open
    }

    remote {
        host   = "server:22" // same server
        port = 8081 // Another port on the remote host to tunnel back to the local port
    }
}
*/
