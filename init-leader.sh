#!/bin/bash
set -e

echo "🔃 Leader init script running..."

echo "host replication all 0.0.0.0/0 md5" >> "$PGDATA/pg_hba.conf"

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOF
  CREATE ROLE replicator WITH REPLICATION LOGIN PASSWORD 'replica_pass';
EOF

echo "✅ Leader Initialization Complete..."