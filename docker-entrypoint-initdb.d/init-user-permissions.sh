#!/bin/bash
# Grant all privileges on the database to the application user from any host
# This script runs during MariaDB initialization after the database and user are created

set -e

# Get database name and user from environment variables (set by MariaDB)
DB_NAME="${MYSQL_DATABASE:-japanesestudent}"
DB_USER="${MYSQL_USER:-appuser}"

# Wait for MariaDB to be ready (it should be, but just in case)
until mysqladmin ping -h localhost --silent; do
  sleep 1
done

# Execute SQL to grant permissions from any host
# This allows connections from Docker network IPs
mysql -u root -p"${MYSQL_ROOT_PASSWORD}" <<EOF
-- Grant all privileges on the database to the user from any host
GRANT ALL PRIVILEGES ON \`${DB_NAME}\`.* TO '${DB_USER}'@'%';
FLUSH PRIVILEGES;
EOF

echo "Successfully granted privileges to ${DB_USER}@'%' on database ${DB_NAME}"

