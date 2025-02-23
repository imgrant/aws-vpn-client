# aws-vpn-client

This project provides a wrapper for connecting to an [AWS Client VPN](https://aws.amazon.com/vpn/client-vpn/) endpoint using [OpenVPN](https://openvpn.net/community/) with SAML authentication.
It is primarily tested on macOS but should also work on Linux.

## Introduction

AWS’s implementation of SAML authentication via an external IdP is ~~an ugly hack~~ a proprietary extension to OpenVPN, and otherwise requires the use of the AWS Client VPN, uh, client, which comes with some limitations.

`aws-vpn-client` is a Go-based CLI tool that simplifies the process of connecting to AWS Client VPN with SAML authentication, without having to use the official client. It leverages a patched version of [OpenVPN](https://openvpn.net/source-code/) to support the necessary authentication mechanisms.

Use cases include, being able to customise your OpenVPN client configuration (eg leverage `up` and `down` scripts), or to use with Linux distributions other than those [supported by the official client](https://docs.aws.amazon.com/vpn/latest/clientvpn-user/client-vpn-connect-linux.html#client-vpn-connect-linux-install).

## Building the client

1. Ensure you have [Go](https://go.dev/) installed on your system.
2. Clone this repository.
3. Run the following command to build the binary:

   ```sh
   go build -o aws-vpn-client
   ```

## How to use

> [!IMPORTANT]
> Ensure you have a ~~butchered~~ patched version of OpenVPN which supports the AWS authentication mechanism. Because [OpenVPN is licensed under the GPL](https://github.com/OpenVPN/openvpn/blob/master/COPYRIGHT.GPL), AWS do make [their source code](https://amazon-source-code-downloads.s3.amazonaws.com/aws/clientvpn/osx-v1.2.5/openvpn-2.4.5-aws-2.tar.gz) available. You can either compile that yourself, or use a [patch](https://github.com/samm-git/aws-vpn-client/blob/master/openvpn-v2.5.1-aws.patch) provided in the original proof-of-concept repository (or other forks), or you can use the Nix derivations and flake provided in this repository (see below).

The client accepts two command line arguments, to specify the path to an OpenVPN config file, and to the OpenVPN client binary to use. If they are not specified, defaults apply:

```bash
$ ./$aws-vpn-client -h
Usage of ./aws-vpn-client:
  -config string
        Path to the OpenVPN config file (default "./config.ovpn")
  -openvpn string
        Path to the AWS-patched OpenVPN binary (default "openvpn")
```

Run the Go binary to initiate authentication with your IdP in a browser window. The client will then launch OpenVPN with the authenticated credentials from the SAML response:

> [!NOTE]
> Save your AWS Client VPN configuration file as `config.ovpn` in your current working directory, or specify its location using the `-config` command-line argument, as above.

```bash
$ ./aws-vpn-client
Loading config from ./config.ovpn
Starting VPN connection to 203.0.113.1:443
2025/02/23 10:05:27 Starting HTTP server at 127.0.0.1:35001
Starting initial OpenVPN to get SAML URL and SID for authentication
Opening webpage for SAML authentication: https://idp.example.com/app/sso/saml?SAMLRequest=XXXXXXXXXXXXXX
2025/02/23 10:05:29 Got SAMLResponse field
Starting OpenVPN with authenticated credentials
SID:instance-2/123456789087654321/f8d29a3e-c147-4b96-a583-92e4d5fb6c31 server 203.0.113.1:443
2025-02-23 10:05:31 OpenVPN 2.5.6 aarch64-apple-darwin24.3.0 [SSL (OpenSSL)] [LZO] [LZ4] [MH/RECVDA] [AEAD] built on Feb 22 2025
2025-02-23 10:05:31 library versions: OpenSSL 1.1.1s  1 Nov 2022, LZO 2.10
2025-02-23 10:05:31 WARNING: ignoring --remote-random-hostname because the hostname is an IP address
2025-02-23 10:05:31 TCP/UDP: Preserving recently used remote address: [AF_INET]203.0.113.1:443
2025-02-23 10:05:31 Socket Buffers: R=[786896->786896] S=[9216->9216]
2025-02-23 10:05:31 UDP link local: (not bound)
2025-02-23 10:05:31 UDP link remote: [AF_INET]203.0.113.1:443
2025-02-23 10:05:31 TLS: Initial packet from [AF_INET]203.0.113.1:443, sid=6d3f5341 f33bc984
2025-02-23 10:05:31 VERIFY OK: depth=3, C=US, ST=Arizona, L=Scottsdale, O=Starfield Technologies, Inc., CN=Starfield Services Root Certificate Authority - G2
2025-02-23 10:05:31 VERIFY OK: depth=2, C=US, O=Amazon, CN=Amazon Root CA 1
2025-02-23 10:05:31 VERIFY OK: depth=1, C=US, O=Amazon, CN=Amazon RSA 2048 M03
2025-02-23 10:05:31 VERIFY KU OK
2025-02-23 10:05:31 Validating certificate extended key usage
2025-02-23 10:05:31 ++ Certificate has EKU (str) TLS Web Server Authentication, expects TLS Web Server Authentication
2025-02-23 10:05:31 VERIFY EKU OK
2025-02-23 10:05:31 VERIFY X509NAME OK: CN=vpn.example.com
2025-02-23 10:05:31 VERIFY OK: depth=0, CN=vpn.example.com
2025-02-23 10:05:31 Control Channel: TLSv1.3, cipher TLSv1.3 TLS_AES_256_GCM_SHA384, peer certificate: 2048 bit RSA, signature: RSA-SHA256
2025-02-23 10:05:31 [vpn.example.com] Peer Connection Initiated with [AF_INET]203.0.113.1:443
2025-02-23 10:05:32 SENT CONTROL [vpn.example.com]: 'PUSH_REQUEST' (status=1)
2025-02-23 10:05:32 PUSH: Received control message: 'PUSH_REPLY,dhcp-option DNS 172.16.0.2,route 172.16.0.0 255.255.255.0,route 10.0.0.0 255.0.0.0,route-gateway 192.168.100.101,topology subnet,ping 1,ping-restart 20,echo,echo,echo,ifconfig 192.168.100.201 255.255.255.224,peer-id 1,cipher AES-256-GCM'
2025-02-23 10:05:32 OPTIONS IMPORT: timers and/or timeouts modified
2025-02-23 10:05:32 OPTIONS IMPORT: --ifconfig/up options modified
2025-02-23 10:05:32 OPTIONS IMPORT: route options modified
2025-02-23 10:05:32 OPTIONS IMPORT: route-related options modified
2025-02-23 10:05:32 OPTIONS IMPORT: peer-id set
2025-02-23 10:05:32 OPTIONS IMPORT: adjusting link_mtu to 1624
2025-02-23 10:05:32 OPTIONS IMPORT: data channel crypto options modified
2025-02-23 10:05:32 Outgoing Data Channel: Cipher 'AES-256-GCM' initialized with 256 bit key
2025-02-23 10:05:32 Incoming Data Channel: Cipher 'AES-256-GCM' initialized with 256 bit key
2025-02-23 10:05:32 ROUTE_GATEWAY 192.51.100.1/255.255.255.0 IFACE=en0 HWADDR=00:1a:11:4b:c2:e5
2025-02-23 10:05:32 Opened utun device utun7
2025-02-23 10:05:32 /sbin/ifconfig utun7 delete
ifconfig: ioctl (SIOCDIFADDR): Can't assign requested address
2025-02-23 10:05:32 NOTE: Tried to delete pre-existing tun/tap instance -- No Problem if failure
2025-02-23 10:05:32 /sbin/ifconfig utun7 192.168.100.201 192.168.100.201 netmask 255.255.255.224 mtu 1500 up
2025-02-23 10:05:32 /sbin/route add -net 192.168.100.100 192.168.100.201 255.255.255.224
add net 192.168.100.100: gateway 192.168.100.201
2025-02-23 10:05:32 /sbin/route add -net 172.16.0.0 192.168.100.101 255.255.255.0
add net 172.23.0.0: gateway 192.168.100.101
2025-02-23 10:05:32 /sbin/route add -net 10.0.0.0 192.168.100.101 255.0.0.0
add net 10.0.0.0: gateway 192.168.100.101
2025-02-23 10:05:32 Initialization Sequence Completed
```

Use CTRL-C to stop the process and tear down the tunnel:

```bash
...
2025-02-23 10:05:32 Initialization Sequence Completed
^C2025-02-23 10:09:39 event_wait : Interrupted system call (code=4)

Received signal: interrupt
Forwarding signal to OpenVPN process...
2025-02-23 10:09:39 /sbin/route delete -net 172.16.0.0 192.168.100.101 255.255.255.0
delete net 172.16.0.0: gateway 192.168.100.101
2025-02-23 10:09:39 /sbin/route delete -net 10.0.0.0 192.168.100.101 255.0.0.0
delete net 10.0.0.0: gateway 192.168.100.101
2025-02-23 10:09:39 Closing TUN/TAP interface
2025-02-23 10:09:39 SIGINT[hard,] received, process exiting
```

> [!TIP] Reconnecting after network interruption
> Use `ping-exit` in your OpenVPN config (instead of `ping-restart`) to automatically reconnect to the VPN server if the tunnel goes down; `aws-vpn-client` runs a loop which will re-initiate the SAML authentication process if OpenVPN exits for any reason (use CTRL-C to kill the process).

## Installing with Nix

This repository includes [Nix](https://nixos.org/) derivations and a flake to build the patched OpenVPN and the Go binary, and automatically link them (pointing the client at the location of the patched `openvpn`).

To use with Nix flakes, add the repository as an input in your flake:

```nix
  inputs = {
    aws-vpn-client = {
      url = "github:imgrant/aws-vpn-client";
    };
  };
```

> [!TIP]
> Don't use `inputs.nixpkgs.follows` here because we want to ensure that the version of Nixpkgs specified in the flake is used in order for the OpenVPN patch to apply cleanly.

Include it in your outputs section, and pass it to, for example, `environment.systemPackages` in a `darwinConfiguration` or `nixosConfiguration`:

```nix
  outputs = { self, nixpkgs, aws-vpn-client, ... }: {
    darwinConfigurations.your-machine = darwin.lib.darwinSystem {
      system = "aarch64-darwin";  # or "x86_64-darwin" for Intel Macs
      modules = [
        {
          environment.systemPackages = [
            aws-vpn-client.packages.${pkgs.system}.default
          ];
        }
      ];
    };
  };
```

You could specify the package in `home.packages` to install it as a user-specific package via [Home Manager](https://nix-community.github.io/home-manager/) instead.

You can also enter a development shell with all dependencies (including the patched OpenVPN build) by running:

```bash
nix develop
```

## Acknowledgements

This work is based on (forked from) [Ralim/aws-vpn-client](https://github.com/Ralim/aws-vpn-client),
which is a single Go binary encompassing the Go listener and shell scripts from the original proof-of-concept
in [samm-git/aws-vpn-client](https://github.com/samm-git/aws-vpn-client) — see also the inceptive [blog post](https://smallhacks.wordpress.com/2020/07/08/aws-client-vpn-internals/) which describes the reverse engineering and implementation of the proof-of-concept.

## Licence

This project is licensed under the MIT Licence - see the [LICENCE](LICENSE) file for details.
