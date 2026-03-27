# Yocto QEMU x86-64 Setup Script

This repository contains a helper script that sets up a Yocto Project build environment based on `poky-master` for the `qemux86-64` machine, adds a custom meta-layer with your recipes, builds `core-image-minimal`, and launches it in QEMU.

---

## 1. Install Dependencies

### 1.1. Arch Linux

On Arch, install the required packages via `pacman`:

```bash
sudo pacman -S --needed \
  base-devel chrpath cpio diffstat file gawk gcc git iputils iputils-ping \
  libacl locales python python-pip python-jinja python-pexpect python-subunit \
  socat texinfo unzip wget xz zstd rpcsvc-proto python-websockets
```

### 1.2. Ubuntu / Debian

On Ubuntu or Debian-based systems, install the equivalent packages:

```bash
sudo apt update
sudo apt install -y \
  build-essential chrpath cpio debianutils diffstat file \
  gawk gcc git iputils-ping libacl1 locales python3 python3-git python3-jinja2 \
  python3-pexpect python3-pip python3-subunit socat texinfo unzip wget xz-utils zstd
```

Additional recommendations:

- Ensure `locale` is generated (for example, `en_US.UTF-8`):
  ```bash
  sudo locale-gen en_US.UTF-8
  sudo update-locale LANG=en_US.UTF-8
  ```

---

## 2. Repository Layout Expectations

The script assumes the following layout relative to the directory from which you run it:

- `conf/local.conf` – your custom local configuration.
- `conf/qemux86-64.conf` – machine configuration for `qemux86-64`.
- `conf/layer.conf` – layer configuration for the custom meta-layer.
- `recipes/recipes-kernel/` – custom kernel recipes.
- `recipes/recipes-core/` – additional core recipes.
- `bitbake/` – BitBake tree containing `bin/bitbake-setup`.

---

## 3. Running the Setup Script

1. Make the script executable:

   ```bash
   chmod +x create-and-run-system.sh
   ```

2. Run the script from the project root:

   ```bash
   ./create-and-run-system.sh
   ```

3. The script will:

   1. **Clean previous build directory**  
      - Removes any existing `bitbake-builds/` directory to start from a clean state.

   2. **Initialize the BitBake build directory**  
      - Runs `bitbake-setup` to create a `poky-master` build tree configured for `qemux86-64`.

   3. **Copy base layer and custom recipes**  
      - Copies `meta-poky` into a new `meta-custom` layer.  
      - Copies your kernel recipes from `recipes/recipes-kernel/` into `meta-custom`.  
      - Copies core recipes from `recipes/recipes-core/` into `meta-custom/recipes-core/`.

   4. **Download kernel image and modules**  
      - Changes into the kernel recipe’s `files/` directory.  
      - Downloads a prebuilt kernel image (`bzImage...bin`) and matching modules archive (`modules...tgz`) for `genericx86-64` from `downloads.yoctoproject.org`.

   5. **Install configuration files**  
      - Creates the `conf/machine/` directory in `meta-custom`.  
      - Copies:
        - `conf/qemux86-64.conf` into the machine configuration directory.  
        - `conf/local.conf` into `bitbake-builds/poky-master/build/conf/`.  
        - `conf/layer.conf` into the `meta-custom/conf/` directory.

   6. **Source the Yocto build environment**  
      - Sources `bitbake-builds/poky-master/build/init-build-env` so `bitbake`, `bitbake-config-build`, and `bitbake-layers` are on the PATH and correctly configured.

   7. **Configure build fragments and layers**  
      - Disables the default `machine/qemux86-64` fragment.  
      - Enables `core/yocto/root-login-with-empty-password` fragment.  
      - Adds the `meta-custom` layer via `bitbake-layers add-layer`.

   8. **Build the image**  
      - Runs `bitbake core-image-minimal` to build a minimal root filesystem and kernel for `qemux86-64`.

   9. **Launch QEMU**  
      - Starts QEMU with  
        `runqemu qemux86-64 core-image-minimal wic nographic snapshot slirp`,  
        booting the built image in an emulated environment.

After these steps complete successfully, you will have a running `core-image-minimal` Yocto image for `qemux86-64` in QEMU.
