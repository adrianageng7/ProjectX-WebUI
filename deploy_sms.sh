#!/system/bin/sh
echo "Stopping old projectx_server..."
killall -9 projectx_server 2>/dev/null
sleep 1

echo "Updating binary..."
rm -f /data/adb/modules/projectx-webui/bin/projectx_server
cp /data/local/tmp/projectx_server /data/adb/modules/projectx-webui/bin/projectx_server
chmod 755 /data/adb/modules/projectx-webui/bin/projectx_server

echo "Updating static web assets..."
mkdir -p /data/adb/modules/projectx-webui/static
cp -rf /data/local/tmp/webstatic/* /data/adb/modules/projectx-webui/static/
chmod -R 755 /data/adb/modules/projectx-webui/static

echo "Starting projectx_server..."
killall -9 projectx_server 2>/dev/null
sleep 1
cd /data/adb/modules/projectx-webui
nohup /data/adb/modules/projectx-webui/bin/projectx_server > /data/adb/modules/projectx-webui/server.log 2>&1 &
sleep 2

echo "Checking PID..."
pidof projectx_server

echo "Verifying static files..."
ls -l /data/adb/modules/projectx-webui/static/sms.html
ls -l /data/adb/modules/projectx-webui/static/index.html
ls -l /data/adb/modules/projectx-webui/static/assets/index*

echo "Testing SMS API..."
curl -s -b "projectx_session=loggedin" http://127.0.0.1/api/sms/list
echo ""
echo "Deploy finished."
