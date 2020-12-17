#!/bin/bash
set -e

mkdir -p openvpncompile
curl "https://swupdate.openvpn.org/community/releases/openvpn-2.4.9.tar.gz" -o openvpncompile/openvpn-2.4.9.tar.gz
cd openvpncompile
tar -zxvf openvpn-2.4.9.tar.gz
rm -rf openvpncompile/openvpn-2.4.9.tar.gz
cd openvpn-2.4.9
patch -p1 <../../openvpn-v2.4.9-aws.patch
./configure
make -j
cd ../..
cp openvpncompile/openvpn-2.4.9/src/openvpn/openvpn ./openvpn-patched
rm -rf openvpncompile
