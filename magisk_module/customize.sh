#!/system/bin/sh

ui_print "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
ui_print "   ProjectX WebUI v2.3.1"
ui_print "   Native Go - Port 80"
ui_print "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
ui_print ""
ui_print "- Created by MarkX (@ntpsbl)"
ui_print "- Komunitas: https://t.me/projectxwebui"
ui_print ""
# Deteksi Environment Root (Magisk / KernelSU / APatch)
if [ -n "$KSU" ]; then
    ui_print "- Root Solution: KernelSU ($KSU_VER)"
elif [ -n "$APATCH" ]; then
    ui_print "- Root Solution: APatch ($APATCH_VER)"
elif [ -n "$MAGISK_VER" ]; then
    ui_print "- Root Solution: Magisk ($MAGISK_VER)"
else
    ui_print "- Root Solution: Standard Root"
fi

# Deteksi arsitektur CPU
ABI=$(getprop ro.product.cpu.abi)
ui_print "- Mendeteksi Arsitektur: $ABI"

case "$ABI" in
    *arm64*|*aarch64*)
        ui_print "- HP 64-bit terdeteksi. Menginstall versi ARM64..."
        rm -f "$MODPATH/bin/projectx_arm"
        mv "$MODPATH/bin/projectx_arm64" "$MODPATH/bin/projectx_server" 2>/dev/null
        ;;
    *armeabi*|*armv7*)
        ui_print "- HP 32-bit terdeteksi. Menginstall versi ARM32..."
        rm -f "$MODPATH/bin/projectx_arm64"
        mv "$MODPATH/bin/projectx_arm" "$MODPATH/bin/projectx_server" 2>/dev/null
        ;;
    *)
        ui_print "! Arsitektur $ABI tidak didukung."
        abort "! Instalasi digagalkan."
        ;;
esac

# Set permission binary Go & scrcpy agar bisa dieksekusi
set_perm_recursive "$MODPATH/bin" root root 0755 0755

# Set permission folder static (HTML/CSS/JS)
set_perm_recursive "$MODPATH/static" root root 0755 0644

ui_print "- Permission diatur."
ui_print ""
ui_print "- Selesai! Restart HP, lalu akses:"
ui_print "  http://(IP_HP_KAMU) atau http://localhost"
ui_print ""
ui_print "- Username: admin | Password: admin"
ui_print "- Log: /data/local/tmp/projectx.log"
ui_print "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
