#!/bin/bash

HOME=$(pwd)
echo "Starting Yocto poky/qemux86-64 setup in: $HOME"

SETUP_RECIPES="recipes"
RECIPES_KERNEL="recipes-kernel"
RECIPES_CORE="recipes-core"

META_POKY=$HOME/"bitbake-builds/poky-master/layers/meta-yocto/meta-poky"
META_CUSTOM=$HOME/"bitbake-builds/poky-master/layers/meta-yocto/meta-custom"

LINUX_KERNEL_RECIPE=$HOME/"linux-kernel-custom.bb"

LOCAL_CONF_SRC=$HOME/"conf/local.conf"
QEMUX86_64_CONF_SRC=$HOME/"conf/qemux86-64.conf"
LAYER_CONF_SRC=$HOME/"conf/layer.conf"

LOCAL_CONF_DST=$HOME/"bitbake-builds/poky-master/build/conf"
QEMUX86_64_CONF_DST=$HOME/"bitbake-builds/poky-master/layers/meta-yocto/meta-custom/conf/machine"
LAYER_CONF_DST=$HOME/"bitbake-builds/poky-master/layers/meta-yocto/meta-custom/conf"

LINUX_KERNEL_DST=$META_CUSTOM/$RECIPES_KERNEL/"linux-kernel-custom/files"

VMLINUZ="bzImage--6.12.69+git0+5b1ff7df00_a7fbaf7533-r0-genericx86-64-20260216222645.bin"
KERNEL_MODULES="modules--6.12.69+git0+5b1ff7df00_a7fbaf7533-r0-genericx86-64-20260216222645.tgz"

echo
### Step 1: Clean previous build (if any) ###
echo "=== Step 1: Clean previous build (if any) ==="
if test -d bitbake-builds; then
    echo "  Found existing bitbake-builds directory, removing it..."
    rm -rf bitbake-builds
else
    echo "  No existing bitbake-builds directory found."
fi
echo

### Step 2: Initialize BitBake build directory ###
echo "=== Step 2: Initialize BitBake build directory ==="
./bitbake/bin/bitbake-setup init --non-interactive poky-master poky-with-sstate distro/poky machine/qemux86-64
if [ $? -ne 0 ]; then
    echo "ERROR: bitbake-setup failed" >&2
    exit 1
fi
echo

### Step 3: Copy layers and recipes ###
echo "=== Step 3: Copy layers and recipes ==="
echo "  Copying meta-poky to meta-custom: $META_POKY -> $META_CUSTOM"
cp -r "$META_POKY" "$META_CUSTOM"

echo "  Copying kernel recipes: $HOME/$SETUP_RECIPES/$RECIPES_KERNEL -> $META_CUSTOM"
cp -r "$HOME/$SETUP_RECIPES/$RECIPES_KERNEL" "$META_CUSTOM"

echo "  Copying core recipes into $META_CUSTOM/$RECIPES_CORE:"
for file in "$HOME/$SETUP_RECIPES/$RECIPES_CORE"/*; do
    echo "    - $file"
    cp -r "$file" "$META_CUSTOM/$RECIPES_CORE/"
done
echo

### Step 4: Download kernel image and modules ###
echo "=== Step 4: Download kernel image and modules into recipe files directory ==="
cd "$LINUX_KERNEL_DST"
echo "  Downloading kernel image: $VMLINUZ"
wget "https://downloads.yoctoproject.org/releases/yocto/yocto-5.3.2/machines/genericx86-64/$VMLINUZ"
echo "  Downloading kernel modules archive: $KERNEL_MODULES"
wget "https://downloads.yoctoproject.org/releases/yocto/yocto-5.3.2/machines/genericx86-64/$KERNEL_MODULES"
cd "$HOME"
echo

### Step 5: Install configuration files ###
echo "=== Step 5: Install configuration files ==="
echo "  Creating machine conf directory: $QEMUX86_64_CONF_DST"
mkdir -p "$QEMUX86_64_CONF_DST"

echo "  Copying qemux86-64.conf"
cp "$QEMUX86_64_CONF_SRC" "$QEMUX86_64_CONF_DST"

echo "  Copying local.conf"
cp "$LOCAL_CONF_SRC" "$LOCAL_CONF_DST"

echo "  Copying layer.conf"
cp "$LAYER_CONF_SRC" "$LAYER_CONF_DST"
echo

### Step 6: Source BitBake build environment ###
echo "=== Step 6: Source BitBake build environment ==="
INIT_ENV="$HOME/bitbake-builds/poky-master/build/init-build-env"
if [ ! -f "$INIT_ENV" ]; then
    echo "ERROR: Build init script not found at $INIT_ENV" >&2
    exit 1
fi

echo "  Sourcing $INIT_ENV ..."
source "$INIT_ENV"
if [ $? -ne 0 ]; then
    echo "ERROR: Failed to source init-build-env" >&2
    exit 1
fi
echo

### Step 7: Configure build fragments and layers ###
echo "=== Step 7: Configure build fragments and layers ==="
echo "  Disabling fragment: machine/qemux86-64"
bitbake-config-build disable-fragment machine/qemux86-64
if [ $? -ne 0 ]; then
    echo "ERROR: bitbake-config-build disable-fragment failed" >&2
    exit 1
fi

echo "  Enabling fragment: core/yocto/root-login-with-empty-password"
bitbake-config-build enable-fragment core/yocto/root-login-with-empty-password
if [ $? -ne 0 ]; then
    echo "ERROR: bitbake-config-build enable-fragment failed" >&2
    exit 1
fi

echo "  Adding custom meta layer: $META_CUSTOM"
bitbake-layers add-layer "$META_CUSTOM"
if [ $? -ne 0 ]; then
    echo "ERROR: bitbake-layers add-layer failed" >&2
    exit 1
fi
echo

### Step 8: Build core-image-minimal ###
echo "=== Step 8: Build core-image-minimal ==="
bitbake core-image-minimal
if [ $? -ne 0 ]; then
    echo "ERROR: bitbake core-image-minimal failed" >&2
    exit 1
fi
echo

### Step 9: Launch QEMU with built image ###
echo "=== Step 9: Launch QEMU with built image ==="
runqemu qemux86-64 core-image-minimal wic nographic snapshot slirp
if [ $? -ne 0 ]; then
    echo "ERROR: runqemu failed" >&2
    exit 1
fi
echo

echo "=== All steps completed successfully. ==="
