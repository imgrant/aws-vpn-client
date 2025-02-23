{ lib, openvpn, fetchpatch }:

# N.b. OpenVPN version passed in should take the patch below cleanly, ie v2.5.1
# See flake.nix for how to use this with a specific version of Nixpkgs
openvpn.overrideAttrs (oldAttrs: {
  pname = "openvpn-aws";
  
  patches = (oldAttrs.patches or []) ++ [
    (fetchpatch {
      name = "aws-saml.patch";
      url = "https://raw.githubusercontent.com/samm-git/aws-vpn-client/master/openvpn-v2.5.1-aws.patch";
      hash = "sha256-9ijhANqqWXVPa00RBCRACtMIsjiBqYVa91V62L4mNas=";
    })
  ];
})