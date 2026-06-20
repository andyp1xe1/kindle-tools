#!/bin/sh
# Name: Wallpapers
# Author: ChillGuys Studio
# Icon:
# DontUseFBInk

EXT=/mnt/us/wallpapers
SERVER="$EXT/wallpapers"
LOG=/tmp/wallpapers.log
PORT=6969

APP_ID="com.chillguys.wallpapers"
WIDGET_DIR=/var/local/mesquite/wallpapers
DB=/var/local/appreg.db

if ! wget -q -T 1 -O /dev/null "http://127.0.0.1:$PORT/ping" 2>/dev/null; then
  "$SERVER" >"$LOG" 2>&1 &
  i=0
  while ! wget -q -T 1 -O /dev/null "http://127.0.0.1:$PORT/ping" 2>/dev/null; do
    [ $i -ge 50 ] && exit 1
    usleep 100000
    i=$((i + 1))
  done
fi

rm -rf "$WIDGET_DIR"
mkdir -p "$WIDGET_DIR"

cp "$EXT/config.xml" "$WIDGET_DIR/config.xml"
sed -e "s|{{PORT}}|$PORT|g" "$EXT/index.html" > "$WIDGET_DIR/index.html"
sed -e "s|{{APP_ID}}|$APP_ID|g" -e "s|{{WIDGET_DIR}}|$WIDGET_DIR|g" \
    "$EXT/migrate.sql" | sqlite3 "$DB"

lipc-set-prop com.lab126.appmgrd stop app://$APP_ID 2>/dev/null
sleep 1
nohup lipc-set-prop com.lab126.appmgrd start app://$APP_ID >/dev/null 2>&1 &
