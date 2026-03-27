SUMMARY = "Pre-built kernel"
LICENSE = "CLOSED"

KERNEL_VERSION = "6.12.69+git0+5b1ff7df00"

inherit deploy

SRC_URI = "file://bzImage--6.12.69+git0+5b1ff7df00_a7fbaf7533-r0-genericx86-64-20260216222645.bin \
           file://modules--6.12.69+git0+5b1ff7df00_a7fbaf7533-r0-genericx86-64-20260216222645.tgz"

PROVIDES += "virtual/kernel"
RPROVIDES:${PN} += "linux-qemux86-64"

RPROVIDES:${PN} += " \
    kernel-module-x-tables \
    kernel-module-ip-tables \
    kernel-module-iptable-filter \
    kernel-module-iptable-nat \
    kernel-module-nf-defrag-ipv4 \
    kernel-module-nf-conntrack \
    kernel-module-nf-conntrack-ipv4 \
    kernel-module-nf-nat \
    kernel-module-ipt-masquerade \
    kernel-module-ip6-tables \
    kernel-module-ip6table-filter \
    kernel-module-scsi-debug \
"
RPROVIDES:${PN} += " \
    kernel-modules \
    kernel-module-loop \
    kernel-module-algif-hash \
    kernel-module-uvesafb \
    kernel-module-af-packet \
    kernel-module-unix \
    kernel-module-ipv6 \
    kernel-module-autofs4 \
    kernel-module-ext4 \
    kernel-module-sd-mod \
"
S = "${UNPACKDIR}"

do_install() {
    install -d ${D}/boot
    install -m 0644 ${UNPACKDIR}/bzImage--6.12.69+git0+5b1ff7df00_a7fbaf7533-r0-genericx86-64-20260216222645.bin ${D}/boot/vmlinuz-custom
    cp -r ${UNPACKDIR}/lib ${D}/
}

do_deploy() {
    install -d ${DEPLOYDIR}
    install -m 0644 \
        ${UNPACKDIR}/bzImage--6.12.69+git0+5b1ff7df00_a7fbaf7533-r0-genericx86-64-20260216222645.bin \
        ${DEPLOYDIR}/bzImage
}
addtask deploy before do_build after do_install

FILES:${PN} += "/boot /lib/modules"

COMPATIBLE_MACHINE = "qemux86-64"
