#!/usr/bin/env bash
#cd pkcs11-helper
#
#./configure --disable-debug \
#        --disable-dependency-tracking \
#        --prefix=/Users/ignas/Code/Mysterium/build/built/pkcs11-helper \
#        --disable-threading \
#        --disable-slotevent \
#        --disable-shared \
#        --enable-openssl=/usr/local/Cellar/openssl/1.0.2n/bin/openssl
#
#make
#make install
#exit

#OPENSSL_CFLAGS="-I$${ssl_staging_dir}/$$a/include" \
#					OPENSSL_SSL_CFLAGS="-I$${ssl_staging_dir}/$$a/include" \
#					OPENSSL_CRYPTO_CFLAGS="-I$${ssl_staging_dir}/$$a/include" \
#					OPENSSL_LIBS="$${ssl_staging_dir}/lib/libssl.a -lz $${ssl_staging_dir}/lib/libcrypto.a -lz" \
#					OPENSSL_SSL_LIBS="$${ssl_staging_dir}/lib/libssl.a" \
#					OPENSSL_CRYPTO_LIBS="$${ssl_staging_dir}/lib/libcrypto.a -lz" \

mkdir openvpn
mkdir pkcs11-helper
tar -xvzf openvpn-2.4.4.tar.gz -C `pwd`/openvpn --strip-components=1
tar -xvzf pkcs11-helper-1.22.tar.bz2 -C `pwd`/pkcs11-helper --strip-components=1

BUILDING_PATH="/Users/ignas/Code/go/src/github.com/mysterium/node/scripts/openvpn/built"

PKG_CONFIG_PATH="$BUILDING_PATH/pkcs11-helper/lib/pkgconfig" \
PKCS11_HELPER_CFLAGS="-I$BUILDING_PATH/pkcs11-helper/include/" \
PKCS11_HELPER_LIBS="-L$BUILDING_PATH/pkcs11-helper/lib \
                                                   -lpkcs11-helper" \
CPPFLAGS="-I/usr/local/opt/openssl/include" \
LDFLAGS="-L/usr/local/opt/openssl/lib" \
OPENSSL_CFLAGS="-I/usr/local/opt/openssl/include/openssl/" \
OPENSSL_LIBS="-I/usr/local/opt/openssl/lib/ -lssl" \
./configure \
    --disable-debug \
    --disable-shared \
    --disable-dependency-tracking \
    --disable-silent-rules \
    --with-crypto-library=openssl \
    --enable-pkcs11=yes \
    --prefix="$BUILDING_PATH/openvpn" \
    --enable-static=yes \
    --enable-shared=no

make #LIBS="-all-static"
make install
