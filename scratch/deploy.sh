#!/system/bin/sh
pkill -f "service.sh"
pkill -f "projectx_server"
sleep 1
rm -f /data/adb/modules/projectx-webui/bin/projectx_server
rm -f /data/adb/modules/projectx-webui/bin/projectx_arm64
cp /data/local/tmp/projectx_server /data/adb/modules/projectx-webui/bin/projectx_server
cp /data/local/tmp/projectx_server /data/adb/modules/projectx-webui/bin/projectx_arm64
chmod 755 /data/adb/modules/projectx-webui/bin/projectx*
cp -r /data/local/tmp/projectx_static/* /data/adb/modules/projectx-webui/static/
nohup sh /data/adb/modules/projectx-webui/service.sh >/dev/null 2>&1 &
sleep 2
echo "DEPLOY_RELOAD_SUCCESS"
