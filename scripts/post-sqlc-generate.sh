#!/bin/bash
echo '
func (q *Queries) DB() DBTX {
    return q.db
}' >> internal/db/db.go