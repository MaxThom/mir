#!/bin/bash
# Must be executed from the root of the project

# Function to update or add an environment variable in a .env file
# Usage: update_env_var <file_path> <variable_name> <new_value>
update_env_var() {
    local file_path="$1"
    local var_name="$2"
    local new_value="$3"

    if [ -f "$file_path" ]; then
        # Check if the variable exists in the file
        if grep -q "^${var_name}=" "$file_path"; then
            # Variable exists, update it
            sed -i "s/^${var_name}=.*/${var_name}=${new_value}/" "$file_path"
            echo "Updated ${var_name} to ${new_value} in ${file_path}"
        else
            # Variable doesn't exist, append it
            echo "" >> "$file_path"
            echo "${var_name}=${new_value}" >> "$file_path"
            echo "Added ${var_name}=${new_value} to ${file_path}"
        fi
    else
        echo "Error: File ${file_path} not found"
        return 1
    fi
}

if [ $# -eq 0 ]
then
    echo "Error: VERSION argument required"
    echo "Usage: $0 VERSION"
    exit 1
fi

VERSION=$1
TEMP_FOLDER=.release-compose
OUTPUT_FILE="mir-compose.tar.gz"

echo "Creating Mir Compose release bundle for version ${VERSION}..."

# Clean up any existing temp folder
rm -rf $TEMP_FOLDER

# Create temp folder structure
mkdir -p $TEMP_FOLDER/mir-compose

# Copy the required directories
echo "Copying compose files..."
cp -r infra/local_mir_support $TEMP_FOLDER/mir-compose/
cp -r infra/surrealdb $TEMP_FOLDER/mir-compose/
cp -r infra/influxdb $TEMP_FOLDER/mir-compose/
cp -r infra/promstack $TEMP_FOLDER/mir-compose/
cp -r infra/mir $TEMP_FOLDER/mir-compose/
cp -r infra/natsio $TEMP_FOLDER/mir-compose/

# Update MIR_VERSION in the copied .env file
ENV_FILE="$TEMP_FOLDER/mir-compose/local_mir_support/.env"
if [ -f "$ENV_FILE" ]; then
    update_env_var "$ENV_FILE" "MIR_VERSION" "$VERSION"
else
    echo "Warning: .env file not found in local_mir_support, creating one..."
    cat > "$ENV_FILE" <<EOF
# Mir Docker Compose Environment Variables

# Version of Mir to deploy
MIR_VERSION=${VERSION}
EOF
fi

# Create README for the compose bundle
echo "Creating README..."
cat > $TEMP_FOLDER/mir-compose/README.md <<EOF
# Mir Compose Bundle - Version ${VERSION}

This bundle contains all the necessary Docker Compose files to run Mir IoT Hub version ${VERSION}.

## Quick Start

1. Navigate to the compose directory:
   \`\`\`bash
   cd local_mir_support/
   \`\`\`

2. Start the entire stack:
   \`\`\`bash
   docker compose up -d
   \`\`\`

3. Access Mir:
   - Mir API: http://localhost:3015
   - Grafana: http://localhost:3000 (admin/mir-operator)
   - InfluxDB: http://localhost:8086 (admin/mir-operator)

## Services Included

- **Mir**: IoT Hub core service (version ${VERSION})
- **NATS**: Message broker for inter-service communication
- **InfluxDB**: Time-series database for telemetry data
- **SurrealDB**: Graph database for device metadata
- **Prometheus Stack**: Monitoring and observability
  - Prometheus
  - Grafana
  - Loki
  - Promtail
  - Alertmanager

## Configuration

The \`.env\` file in \`local_mir_support/\` contains the Mir version.
You can modify other settings in the individual compose files as needed.

## Stopping the Stack

To stop all services:
\`\`\`bash
cd mir-compose/local_mir_support
docker compose down
\`\`\`

To stop and remove all data:
\`\`\`bash
docker compose down -v
\`\`\`

## Documentation

For more information, visit: https://book.mirhub.io
EOF

# Create the tar.gz archive
echo "Creating archive ${OUTPUT_FILE}..."
tar -czf $TEMP_FOLDER/${OUTPUT_FILE} -C $TEMP_FOLDER mir-compose

# Move the archive to the temp folder root for upload
#mv $TEMP_FOLDER/${OUTPUT_FILE} $TEMP_FOLDER/

# Clean up the copied directories, keeping only the archive
rm -rf $TEMP_FOLDER/mir-compose
rm -f $TEMP_FOLDER/README.md

echo ""
echo "Mir Compose bundle ${OUTPUT_FILE} created successfully! 🚀"
echo "Archive location: $TEMP_FOLDER/${OUTPUT_FILE}"
