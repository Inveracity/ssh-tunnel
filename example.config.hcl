tunnel {
    // user: the user to connect to the remote host
    user = "user"
    // name: a name to show in the terminal
    name = "myservice" // Optional
    local {
        port = 8080 // Local port to open

        // Run a command locally after the tunnel is opened.
        // This is useful if the service is a webservice and you want to open a browser.
        cmd = "wslview http://localhost:8080" // Optional
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
