#!/bin/bash
set -e

mkdir -p openvpncompile
curl "https://swupdate.openvpn.org/community/releases/openvpn-2.5.1.tar.xz" -o openvpncompile/openvpn-2.5.1.tar.xz
cd openvpncompile
tar -xf openvpn-2.5.1.tar.xz
rm -rf openvpncompile/openvpn-2.5.1.tar.xz
cd openvpn-2.5.1
patch -p1 <../../openvpn-v2.5.1-aws.patch
./configure
make -j
cd ../..
cp openvpncompile/openvpn-2.5.1/src/openvpn/openvpn ./openvpn-patched
rm -rf openvpncompile
