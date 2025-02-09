#!/bin/sh

echo "hello"

# Create the data directory if it doesn't exist (important!)
mkdir -p data  # Relative path to create 'data' directory within /app

# Check if the database file exists
if [ ! -f data/orders.db ]; then # Relative path to data/orders.db within /app
    echo "Creating database..."
    sqlite3 data/orders.db < migrations/00-create-orders.sql # Relative path to migrations/00-create-orders.sql within /app
else
    echo "Database already exists."
fi

# Start your application
exec "$@"  # Execute the command passed to the container
