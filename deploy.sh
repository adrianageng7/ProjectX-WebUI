#!/system/bin/sh
set -e

echo "[1/5] Stopping current projectx_server..."
killall -9 projectx_server 2>/dev/null || true
pkill -9 -f projectx_server 2>/dev/null || true
sleep 1

echo "[2/5] Replacing binary..."
rm -f /data/adb/modules/projectx-webui/bin/projectx_server
cp /data/local/tmp/projectx_server /data/adb/modules/projectx-webui/bin/projectx_server
chmod 0755 /data/adb/modules/projectx-webui/bin/projectx_server
chown root:root /data/adb/modules/projectx-webui/bin/projectx_server

echo "[3/5] Updating static assets..."
mkdir -p /data/adb/modules/projectx-webui/static
cp -rf /data/local/tmp/static/* /data/adb/modules/projectx-webui/static/
chmod -R 0755 /data/adb/modules/projectx-webui/static
chown -R root:root /data/adb/modules/projectx-webui/static

echo "[4/5] Cleaning up /data/local/tmp..."
rm -rf /data/local/tmp/projectx_server
rm -rf /data/local/tmp/static
rm -f /data/local/tmp/deploy.sh

echo "[5/5] Restarting projectx_server..."
if ! pidof projectx_server >/dev/null 2>&1; then
    cd /data/adb/modules/projectx-webui
    nohup /data/adb/modules/projectx-webui/bin/projectx_server >> /data/local/tmp/projectx.log 2>&1 &
fi

echo "DEPLOY_COMPLETE"
