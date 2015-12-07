tftp2httpd
==========
tftp2httpd is a daemon written in Go which acts as a TFTP server and proxies
the requests to an HTTP server. This can be used to serve configuration files
requested by embedded devices on a dynamic basis, by taking advantage of the
huge HTTP ecosystem for serving requests dynamically.

For example, maybe you have a model of IP phone which requests a configuration
file from a TFTP server based on its MAC address. With tftp2httpd, you can
proxy that request to an HTTP handler which generates an appropriate
configuration based on the filename.

  1. An embedded device with MAC address 0000.2222.4444 requests configuration
     file p000022224444.cfg from the TFTP server specified by DHCP.
  2. The TFTP server, which is running tftp2httpd, proxies the request to an
     HTTP server setup to handle the request dynamically.
  3. The HTTP server parses the filename to deduce the MAC address and
     generates an appropriate configuration, which it returns via HTTP.
  4. The TFTP server returns the generated configuration to the embedded device
     via TFTP.

The requestor IP address is passed to the HTTP server in the X-Forwarded-For
header.

Note that valid filenames to TFTP are currently constrained by a fairly strict
regex. If you use strange characters in your filenames you may need to alter
the regex in the source code. The current rules are:

  - No element of the path must start with a .
  - Valid characters are: alphanumerics, space, colon, dot, underscore, dash

Currently only read requests are supported, which are converted to GET
requests.

Status
------
This is early alpha software. It appears to work nicely, but I don't at this
point guarantee there isn't some memory leak somewhere.

Requirements
------------
Besides the Go compiler and Go standard library, github.com/hlandau/degoutils
is required. See the build instructions below.

Security
--------
tftp2httpd is designed to be highly secure. Besides being written in Go, which
eliminates many classes of possible security vulnerabilities, the daemon can
chroot itself into an empty directory and drop privileges.

For this reason, tftp2httpd cannot currently reload its configuration file.
However, the configuration is so simple I don't think this is a significant
problem.

Building
--------
You need the go compiler installed. Clone the repository and run 'make'. The
daemon will be placed in the bin directory.

    # Clone.
    git clone https://github.com/hlandau/tftp2httpd
    cd tftp2httpd

    # Build.
    make

    # Configure.
    mkdir etc; cp doc/tftp2httpd.conf.example etc/tftp2httpd.conf

    # Run.
    ./bin/tftp2httpd -conf=etc/tftp2httpd.conf

Running
-------
The daemon requires a TOML configuration file. The path to this file should be
specified with the `-conf` option to the daemon. An example is in
`etc/tftp2httpd.conf`.

The following options are supported in the configuration file:

  `http_url`: The URL to proxy TFTP file requests to. The TFTP filename will be
    appended to this URL. A slash will not be added to the end of this URL
    automatically, so add one yourself if you want one.

  `tftp_listen`: A string specifying what interface to bind on. Default: ":69".
    You could bind on a specific interface using something like "127.0.0.1:69".

The following additional options can be specified as flags only:

  `-uid=UID`: UID of user to drop to if run as root. Ignored if not run as root. Must
    be an integer. Usernames currently not supported because go doesn't support
    getpwnam.

  `-gid=GID`: GID of group to drop to if run as root. Ignored if not run as root. Must
    be an integer. Group names not currently supported.

  `-pidfile=path`: Path to a PID file to create.

  `-daemon`: Daemonize (doesn't fork).

  `-chroot`: Specify custom chroot dir.

You can either:

  1. Run as a normal user with CAP_NET_BIND_SERVICE set on the binary so as to
     allow the daemon to bind to port 69.

  2. Run as a normal user on an unprivileged port.

  3. Run as root with "uid" and "gid" set in the configuration file. The daemon
     will drop privileges and chroot automatically after binding.

If you pass `-service.daemon=1`, the daemon will close stdin/stdout/stderr and setsid
if possible. The daemon does not currently fork due to limitations with Go.

tftp2httpd will log to syslog when daemonized using the `daemon` facility.

TODO
----

  - Support TFTP option negotiation.

  - Write support.

  - Miscellanea.

Licence
-------

    © 2013—2014 Hugo Landau <hlandau@devever.net>  GPLv3+ License

