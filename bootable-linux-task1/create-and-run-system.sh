#!/bin/bash

HOME=$(pwd)

SETUP_RECIPES="recipes"
RECIPES_KERNEL="recipes-kernel"
RECIPES_CORE="recipes-core"

META_POKY=$HOME/bitbake-builds/poky-master/layers/meta-yocto/meta-poky
META_CUSTOM=$HOME/bitbake-builds/poky-master/layers/meta-yocto/meta-custom

LINUX_KERNEL_RECIPE=$HOME/linux-kernel-custom.bb

LOCAL_CONF_SRC=$HOME/"conf/local.conf"
QEMUX86_64_CONF_SRC=$HOME/"conf/qemux86-64.conf"
LAYER_CONF_SRC=$HOME/"conf/layer.conf"

LOCAL_CONF_DST=$HOME/"bitbake-builds/poky-master/build/conf"
QEMUX86_64_CONF_DST=$HOME/"bitbake-builds/poky-master/layers/meta-yocto/meta-custom/conf/machine"
LAYER_CONF_DST=$HOME/"bitbake-builds/poky-master/layers/meta-yocto/meta-custom/conf"

LINUX_KERNEL_DST=$META_CUSTOM/$RECIPES_KERNEL/linux-kernel-custom/files

VMLINUZ="bzImage--6.12.69+git0+5b1ff7df00_a7fbaf7533-r0-genericx86-64-20260216222645.bin"
KERNEL_MODULES="modules--6.12.69+git0+5b1ff7df00_a7fbaf7533-r0-genericx86-64-20260216222645.tgz"

#yay -S --needed base-devel chrpath cpio diffstat file gawk gcc git iputils iputils-ping libacl locales python python-pip python-jinja python-pexpect python-subunit socat texinfo unzip wget xz zstd rpcsvc-proto python-websockets

if test -d bitbake-builds; then
    rm -rf bitbake-builds
fi

./bitbake/bin/bitbake-setup init --non-interactive poky-master poky-with-sstate distro/poky machine/qemux86-64

cp -r $META_POKY $META_CUSTOM
cp -r $HOME/$SETUP_RECIPES/$RECIPES_KERNEL $META_CUSTOM

for file in $HOME/$SETUP_RECIPES/$RECIPES_CORE/*; do
    echo $file
    cp -r $file $META_CUSTOM/$RECIPES_CORE/
done

cd $LINUX_KERNEL_DST
wget https://downloads.yoctoproject.org/releases/yocto/yocto-5.3.2/machines/genericx86-64/$VMLINUZ
wget https://downloads.yoctoproject.org/releases/yocto/yocto-5.3.2/machines/genericx86-64/$KERNEL_MODULES
cd $HOME

mkdir -p $QEMUX86_64_CONF_DST
cp $QEMUX86_64_CONF_SRC $QEMUX86_64_CONF_DST
cp $LOCAL_CONF_SRC $LOCAL_CONF_DST
cp $LAYER_CONF_SRC $LAYER_CONF_DST

source $HOME/bitbake-builds/poky-master/build/init-build-env

bitbake-config-build disable-fragment machine/qemux86-64
bitbake-config-build enable-fragment core/yocto/root-login-with-empty-password
bitbake-layers add-layer $META_CUSTOM

bitbake core-image-minimal

runqemu qemux86-64 core-image-minimal wic nographic snapshot slirp


