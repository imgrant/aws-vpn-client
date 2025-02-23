{
  description = "AWS VPN Client wrapper with patched OpenVPN for SAML authentication";

  inputs = {
    nixpkgs.url = "https://flakehub.com/f/NixOS/nixpkgs/0.2205.*";  # Nixpkgs version with OpenVPN v2.5.6
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        
        # Build the patched OpenVPN
        openvpn-aws = pkgs.callPackage ./openvpn-aws.nix { };
        
        # Build the wrapper and SAML listener
        aws-vpn-client = pkgs.callPackage ./default.nix { 
          inherit openvpn-aws;
        };
      in
      {
        packages = {
          inherit aws-vpn-client openvpn-aws;
          default = aws-vpn-client;
        };

        # Development shell with both packages available
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            delve
            openvpn-aws
          ];
        };
      }
    );
}