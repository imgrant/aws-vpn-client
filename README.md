# aws-vpn-client

This is an updated PoC to connect to the AWS Client VPN with OSS OpenVPN using SAML
authentication. Tested on Linux primarily, but should work on OS X.

This is based on the work by [samm's repo](https://github.com/samm-git/aws-vpn-client); and you can read their [ blog post](https://smallhacks.wordpress.com/2020/07/08/aws-client-vpn-internals/) for the implementation details.

This version has taken the shell scripts and folds all of that into a single golang binary.

## Content of the repository

- [openvpn-v2.4.9-aws.patch](openvpn-v2.4.9-aws.patch) - patch required to build
  AWS compatible OpenVPN v2.4.9, based on the
  [AWS source code](https://amazon-source-code-downloads.s3.amazonaws.com/aws/clientvpn/wpf-v1.2.0/openvpn-2.4.5-aws-1.tar.gz) (thanks to @heprotecbuthealsoattac) for the link.
- [main.go](main.go) - a go wrapper to perform the authentication and handle the double-tap of connecting to the vpn
- [compile-patched-openvpn.sh](compile-patched-openvpn.sh) - bash script to download,patch and compile the openvpn client to use for the golang tool

## How to use

1. Build patched openvpn version using `compile-patched-openvpn.sh`
1. Either save your downloaded aws config as `~/.awsvpn.conf` or place it somewhere nice
1. Compile the go wrapper `go build`
1. Run the golang tool, use command arg `-config` to point to your conf file if its not saved as `~/.awsvpn.conf`
1. This should do the rest from here
