#!/system/bin/sh
MODDIR=${0%/*}
LOGFILE="/data/local/tmp/projectx.log"
SERVER="$MODDIR/bin/projectx_server"

# Tunggu boot selesai
until [ "$(getprop sys.boot_completed)" = "1" ]; do
    sleep 2
done

# Tunggu sebentar lagi biar filesystem stabil
sleep 5

echo "[$(date)] ProjectX starting..." >> "$LOGFILE"

# Pindah ke folder modul supaya ./static bisa dibaca
cd "$MODDIR"
chmod 755 "$SERVER" 2>/dev/null

# Loop watchdog: restart server kalau crash
while true; do
    echo "[$(date)] Launching projectx_server..." >> "$LOGFILE"
    "$SERVER" >> "$LOGFILE" 2>&1
    echo "[$(date)] Server exited, restarting in 5s..." >> "$LOGFILE"
    sleep 5
done &
