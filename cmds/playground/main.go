package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maxthom/mir/pkgs/mir_v1"
)

// PostgreSQL connection
type PostgresDB struct {
	pool *pgxpool.Pool
}

func ConnectToDb(ctx context.Context, dsn string) (*PostgresDB, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	// Verify the connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to verify the connection: %w", err)
	}

	return &PostgresDB{pool: pool}, nil
}

func (db *PostgresDB) Close() {
	db.pool.Close()
}

// Initialize database schema
func (db *PostgresDB) InitSchema(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS devices (
		-- Object fields
		id VARCHAR PRIMARY KEY,
		api_version VARCHAR NOT NULL,
		kind VARCHAR NOT NULL,
		name VARCHAR NOT NULL,
		namespace VARCHAR NOT NULL DEFAULT 'default',
		labels JSONB,
		annotations JSONB,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW(),

		-- Spec fields
		device_id VARCHAR,
		disabled BOOLEAN,

		-- Properties with JSONB for desired/reported
		properties_desired JSONB,
		properties_reported JSONB,

		-- Status fields
		online BOOLEAN,
		last_heartbeat TIMESTAMP,
		schema_compressed BYTEA,
		schema_package_names TEXT[],
		schema_last_fetch TIMESTAMP,
		status_properties_desired JSONB,
		status_properties_reported JSONB,
	);

	CREATE INDEX IF NOT EXISTS idx_devices_namespace ON devices(namespace);
	CREATE INDEX IF NOT EXISTS idx_devices_name ON devices(name);
	CREATE INDEX IF NOT EXISTS idx_devices_labels ON devices USING GIN(labels);
	CREATE INDEX IF NOT EXISTS idx_devices_device_id ON devices(device_id);
	`

	_, err := db.pool.Exec(ctx, schema)
	return err
}

// CRUD Operations
func (db *PostgresDB) CreateDevice(ctx context.Context, device *mir_v1.Device) error {
	// Marshal JSONB fields with JSON v2
	labelsJSON, _ := json.Marshal(device.Meta.Labels)
	annotationsJSON, _ := json.Marshal(device.Meta.Annotations)
	propertiesDesiredJSON, _ := json.Marshal(device.Properties.Desired)
	propertiesReportedJSON, _ := json.Marshal(device.Properties.Reported)
	statusPropertiesDesiredJSON, _ := json.Marshal(device.Status.Properties.Desired)
	statusPropertiesReportedJSON, _ := json.Marshal(device.Status.Properties.Reported)
	statusEventsJSON, _ := json.Marshal(device.Status.Events)

	query := `
		INSERT INTO devices (
			id, api_version, kind, name, namespace, labels, annotations,
			device_id, disabled,
			properties_desired, properties_reported,
			online, last_heartbeat, schema_compressed, schema_package_names, schema_last_fetch,
			status_properties_desired, status_properties_reported, status_events
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
		)
	`

	_, err := db.pool.Exec(ctx, query,
		device.Meta.Name+"/"+device.Meta.Namespace, // Use name/namespace as ID
		device.ApiVersion,
		device.Kind,
		device.Meta.Name,
		device.Meta.Namespace,
		labelsJSON,
		annotationsJSON,
		device.Spec.DeviceId,
		device.Spec.Disabled,
		propertiesDesiredJSON,
		propertiesReportedJSON,
		device.Status.Online,
		device.Status.LastHearthbeat,
		device.Status.Schema.CompressedSchema,
		device.Status.Schema.PackageNames,
		device.Status.Schema.LastSchemaFetch,
		statusPropertiesDesiredJSON,
		statusPropertiesReportedJSON,
		statusEventsJSON,
	)

	return err
}

func (db *PostgresDB) GetDevice(ctx context.Context, name, namespace string) (*mir_v1.Device, error) {
	id := name + "/" + namespace
	query := `
		SELECT
			id, api_version, kind, name, namespace, labels, annotations, created_at, updated_at,
			device_id, disabled,
			properties_desired, properties_reported,
			online, last_heartbeat, schema_compressed, schema_package_names, schema_last_fetch,
			status_properties_desired, status_properties_reported, status_events
		FROM devices
		WHERE id = $1
	`

	row := db.pool.QueryRow(ctx, query, id)

	var device mir_v1.Device
	var labelsJSON, annotationsJSON []byte
	var propertiesDesiredJSON, propertiesReportedJSON []byte
	var statusPropertiesDesiredJSON, statusPropertiesReportedJSON []byte
	var statusEventsJSON []byte
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&id,
		&device.ApiVersion,
		&device.Kind,
		&device.Meta.Name,
		&device.Meta.Namespace,
		&labelsJSON,
		&annotationsJSON,
		&createdAt,
		&updatedAt,
		&device.Spec.DeviceId,
		&device.Spec.Disabled,
		&propertiesDesiredJSON,
		&propertiesReportedJSON,
		&device.Status.Online,
		&device.Status.LastHearthbeat,
		&device.Status.Schema.CompressedSchema,
		&device.Status.Schema.PackageNames,
		&device.Status.Schema.LastSchemaFetch,
		&statusPropertiesDesiredJSON,
		&statusPropertiesReportedJSON,
		&statusEventsJSON,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("device not found: %s/%s", name, namespace)
		}
		return nil, err
	}

	// Unmarshal JSONB fields with JSON v2
	if labelsJSON != nil {
		json.Unmarshal(labelsJSON, &device.Meta.Labels)
	}
	if annotationsJSON != nil {
		json.Unmarshal(annotationsJSON, &device.Meta.Annotations)
	}
	if propertiesDesiredJSON != nil {
		json.Unmarshal(propertiesDesiredJSON, &device.Properties.Desired)
	}
	if propertiesReportedJSON != nil {
		json.Unmarshal(propertiesReportedJSON, &device.Properties.Reported)
	}
	if statusPropertiesDesiredJSON != nil {
		json.Unmarshal(statusPropertiesDesiredJSON, &device.Status.Properties.Desired)
	}
	if statusPropertiesReportedJSON != nil {
		json.Unmarshal(statusPropertiesReportedJSON, &device.Status.Properties.Reported)
	}
	if statusEventsJSON != nil {
		json.Unmarshal(statusEventsJSON, &device.Status.Events)
	}

	return &device, nil
}

func (db *PostgresDB) UpdateDevice(ctx context.Context, device *mir_v1.Device) error {
	// Marshal JSONB fields with JSON v2
	labelsJSON, _ := json.Marshal(device.Meta.Labels)
	annotationsJSON, _ := json.Marshal(device.Meta.Annotations)
	propertiesDesiredJSON, _ := json.Marshal(device.Properties.Desired)
	propertiesReportedJSON, _ := json.Marshal(device.Properties.Reported)
	statusPropertiesDesiredJSON, _ := json.Marshal(device.Status.Properties.Desired)
	statusPropertiesReportedJSON, _ := json.Marshal(device.Status.Properties.Reported)
	statusEventsJSON, _ := json.Marshal(device.Status.Events)

	id := device.Meta.Name + "/" + device.Meta.Namespace
	query := `
		UPDATE devices SET
			api_version = $2,
			kind = $3,
			name = $4,
			namespace = $5,
			labels = $6,
			annotations = $7,
			updated_at = NOW(),
			device_id = $8,
			disabled = $9,
			properties_desired = $10,
			properties_reported = $11,
			online = $12,
			last_heartbeat = $13,
			schema_compressed = $14,
			schema_package_names = $15,
			schema_last_fetch = $16,
			status_properties_desired = $17,
			status_properties_reported = $18,
			status_events = $19
		WHERE id = $1
	`

	_, err := db.pool.Exec(ctx, query,
		id,
		device.ApiVersion,
		device.Kind,
		device.Meta.Name,
		device.Meta.Namespace,
		labelsJSON,
		annotationsJSON,
		device.Spec.DeviceId,
		device.Spec.Disabled,
		propertiesDesiredJSON,
		propertiesReportedJSON,
		device.Status.Online,
		device.Status.LastHearthbeat,
		device.Status.Schema.CompressedSchema,
		device.Status.Schema.PackageNames,
		device.Status.Schema.LastSchemaFetch,
		statusPropertiesDesiredJSON,
		statusPropertiesReportedJSON,
		statusEventsJSON,
	)

	return err
}

func (db *PostgresDB) DeleteDevice(ctx context.Context, name, namespace string) error {
	id := name + "/" + namespace
	query := `DELETE FROM devices WHERE id = $1`
	_, err := db.pool.Exec(ctx, query, id)
	return err
}

func (db *PostgresDB) ListDevices(ctx context.Context, namespace string) ([]mir_v1.Device, error) {
	query := `
		SELECT
			id, api_version, kind, name, namespace, labels, annotations, created_at, updated_at,
			device_id, disabled,
			properties_desired, properties_reported,
			online, last_heartbeat, schema_compressed, schema_package_names, schema_last_fetch,
			status_properties_desired, status_properties_reported, status_events
		FROM devices
		WHERE namespace = $1
		ORDER BY name
	`

	rows, err := db.pool.Query(ctx, query, namespace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []mir_v1.Device
	for rows.Next() {
		var device mir_v1.Device
		var id string
		var labelsJSON, annotationsJSON []byte
		var propertiesDesiredJSON, propertiesReportedJSON []byte
		var statusPropertiesDesiredJSON, statusPropertiesReportedJSON []byte
		var statusEventsJSON []byte
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&id,
			&device.ApiVersion,
			&device.Kind,
			&device.Meta.Name,
			&device.Meta.Namespace,
			&labelsJSON,
			&annotationsJSON,
			&createdAt,
			&updatedAt,
			&device.Spec.DeviceId,
			&device.Spec.Disabled,
			&propertiesDesiredJSON,
			&propertiesReportedJSON,
			&device.Status.Online,
			&device.Status.LastHearthbeat,
			&device.Status.Schema.CompressedSchema,
			&device.Status.Schema.PackageNames,
			&device.Status.Schema.LastSchemaFetch,
			&statusPropertiesDesiredJSON,
			&statusPropertiesReportedJSON,
			&statusEventsJSON,
		)

		if err != nil {
			return nil, err
		}

		// Unmarshal JSONB fields with JSON v2
		if labelsJSON != nil {
			json.Unmarshal(labelsJSON, &device.Meta.Labels)
		}
		if annotationsJSON != nil {
			json.Unmarshal(annotationsJSON, &device.Meta.Annotations)
		}
		if propertiesDesiredJSON != nil {
			json.Unmarshal(propertiesDesiredJSON, &device.Properties.Desired)
		}
		if propertiesReportedJSON != nil {
			json.Unmarshal(propertiesReportedJSON, &device.Properties.Reported)
		}
		if statusPropertiesDesiredJSON != nil {
			json.Unmarshal(statusPropertiesDesiredJSON, &device.Status.Properties.Desired)
		}
		if statusPropertiesReportedJSON != nil {
			json.Unmarshal(statusPropertiesReportedJSON, &device.Status.Properties.Reported)
		}
		if statusEventsJSON != nil {
			json.Unmarshal(statusEventsJSON, &device.Status.Events)
		}

		devices = append(devices, device)
	}

	return devices, nil
}

func main() {
	ctx := context.Background()

	// Connection string for PostgreSQL
	dsn := "postgresql://admin:mir-operator@localhost:5432/mir?sslmode=disable"

	// Connect to database
	db, err := ConnectToDb(ctx, dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize schema
	if err := db.InitSchema(ctx); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	fmt.Println("Connected to PostgreSQL and initialized schema")

	// Create a test device
	device := mir_v1.NewDevice()
	device.Meta.Name = "test-device"
	device.Meta.Namespace = "default"
	device.Meta.Labels = map[string]string{
		"type":     "sensor",
		"location": "building-a",
	}
	device.Spec.DeviceId = "device-001"
	device.Properties.Desired = map[string]interface{}{
		"temperature": 25.5,
		"humidity":    60,
	}
	device.Properties.Reported = map[string]interface{}{
		"temperature": 24.8,
		"humidity":    58,
	}
	online := true
	device.Status.Online = &online
	device.Status.LastHearthbeat = &time.Time{}
	*device.Status.LastHearthbeat = time.Now()

	// CREATE
	fmt.Println("Creating device...")
	if err := db.CreateDevice(ctx, &device); err != nil {
		log.Fatalf("Failed to create device: %v", err)
	}
	fmt.Println("Device created successfully")

	// READ
	fmt.Println("Reading device...")
	retrievedDevice, err := db.GetDevice(ctx, "test-device", "default")
	if err != nil {
		log.Fatalf("Failed to get device: %v", err)
	}
	fmt.Printf("Retrieved device: %+v\n", retrievedDevice.Meta.Name)
	fmt.Printf("Device ID: %s\n", retrievedDevice.Spec.DeviceId)
	fmt.Printf("Labels: %+v\n", retrievedDevice.Meta.Labels)
	fmt.Printf("Desired properties: %+v\n", retrievedDevice.Properties.Desired)
	fmt.Printf("Reported properties: %+v\n", retrievedDevice.Properties.Reported)

	// UPDATE
	fmt.Println("Updating device...")
	retrievedDevice.Properties.Reported["temperature"] = 26.2
	retrievedDevice.Meta.Labels["updated"] = "true"
	if err := db.UpdateDevice(ctx, retrievedDevice); err != nil {
		log.Fatalf("Failed to update device: %v", err)
	}
	fmt.Println("Device updated successfully")

	// READ again to verify update
	updatedDevice, err := db.GetDevice(ctx, "test-device", "default")
	if err != nil {
		log.Fatalf("Failed to get updated device: %v", err)
	}
	fmt.Printf("Updated reported temperature: %v\n", updatedDevice.Properties.Reported["temperature"])
	fmt.Printf("Updated labels: %+v\n", updatedDevice.Meta.Labels)

	// LIST
	fmt.Println("Listing devices...")
	devices, err := db.ListDevices(ctx, "default")
	if err != nil {
		log.Fatalf("Failed to list devices: %v", err)
	}
	fmt.Printf("Found %d devices in default namespace\n", len(devices))

	// DELETE
	fmt.Println("Deleting device...")
	if err := db.DeleteDevice(ctx, "test-device", "default"); err != nil {
		log.Fatalf("Failed to delete device: %v", err)
	}
	fmt.Println("Device deleted successfully")

	// Verify deletion
	_, err = db.GetDevice(ctx, "test-device", "default")
	if err != nil {
		fmt.Println("Device successfully deleted (not found)")
	}

	fmt.Println("CRUD operations completed successfully!")
}
