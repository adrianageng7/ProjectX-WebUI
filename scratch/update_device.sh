#!/system/bin/sh
set -e
echo "Updating projectx_server binaries..."
cp -f /data/local/tmp/projectx_server /data/adb/modules/projectx-webui/bin/projectx_server
cp -f /data/local/tmp/projectx_server /data/adb/modules/projectx-webui/bin/projectx_server_arm64
chmod 755 /data/adb/modules/projectx-webui/bin/projectx_server*

echo "Updating static assets..."
cp -rf /data/local/tmp/static_new/* /data/adb/modules/projectx-webui/static/

echo "Restarting projectx_server..."
killall projectx_server 2>/dev/null || true
pkill -9 -f projectx_server 2>/dev/null || true

sleep 3
echo "Running process status:"
ps -A | grep projectx || true
echo "Update complete!"
