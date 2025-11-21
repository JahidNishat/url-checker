#!/bin/bash
set -e
echo "host replication all 0.0.0.0/0 md5" >> "$PGDATA/pg_hba.conf"
echo "✅ Replication rule added to pg_hba.conf"
