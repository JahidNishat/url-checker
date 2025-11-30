#!/bin/bash
set -e

echo "🔃 Replica init script running..."

export PGPASSWORD='replica_pass'

rm -rf "$PGDATA"/*
echo "🗑️ Cleared follower data directory"

echo "⏸️ Waiting to start the postgres leader..."
until pg_isready -h postgres-leader -p 5432 -U replicator; do
  echo "Still waiting..."
  sleep 1
done

pg_basebackup \
-h postgres-leader \
-p 5432 \
-U replicator \
-D "$PGDATA" \
-Fp -Xs -P -R

echo "📦 Base Backup Complete"

# Fix permissions so postgres can start immediately
chown -R postgres:postgres "$PGDATA"
chmod 700 "$PGDATA"

echo "hot_standby=on" >> "$PGDATA/postgresql.auto.conf"

echo "🚀 Replica initialization complete."