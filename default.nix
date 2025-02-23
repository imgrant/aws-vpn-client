{ lib, buildGoModule, pkg-config, openvpn-aws }:

buildGoModule rec {
  pname = "aws-vpn-client";
  version = "0.1.0";

  src = ./.;
  vendorSha256 = null;

  # Add openvpn-aws to the runtime path
  nativeBuildInputs = [ pkg-config ];

   # Pass the openvpn binary path to the Go program
  ldflags = [
    "-X main.openVpnBinary=${openvpn-aws}/bin/openvpn"
  ];

  meta = with lib; {
    description = "AWS VPN Client wrapper for handling SAML authentication";
    homepage = "https://github.com/imgrant/aws-vpn-client";
    license = licenses.mit;
    platforms = platforms.darwin;
    maintainers = with maintainers; [
      {
        name = "Ian Grant";
        github = "imgrant";
      }
    ];
  };
}