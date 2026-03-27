add_hello_world() {
    echo "Hello World" > ${IMAGE_ROOTFS}/etc/motd
}
ROOTFS_POSTPROCESS_COMMAND += "add_hello_world; "

