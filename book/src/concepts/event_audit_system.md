# Event and Audit System

The Mir IoT Hub includes a comprehensive event and audit system that provides complete visibility into all system operations, device interactions, and state changes. This system serves as the foundation for monitoring, debugging, and maintaining operational awareness across your infrastructure.

The event system provides a complete audit trail for:

#### Device Operations
- Track device lifecycle from registration to decommission
- Monitor device connectivity and health status

#### Command Execution
- Log all commands sent to devices
- Track command execution success/failure

#### Configuration Management
- Record all configuration updates
- Track desired vs. reported state changes


## Event Generation and Subscriptions

The Module SDK provides all the functionnality to both generate new type of events and subscribes to existing ones. Each generated event is captured by the system and stored.


## Event Types

### System Events

**Device Lifecycle Events:**
- `DeviceOnline` - Device connects to the system
- `DeviceOffline` - Device disconnects from the system
- `DeviceCreate` - New device registered
- `DeviceUpdate` - Device metadata updated
- `DeviceDelete` - Device removed from system

**Command Events:**
- `CommandSent` - Command dispatched to device
- `CommandReceived` - Device acknowledged command
- `CommandCompleted` - Command execution finished
- `CommandFailed` - Command execution failed

**Configuration Events:**
- `DesiredPropertiesUpdated` - New configuration sent to device
- `ReportedPropertiesUpdated` - Device reported state change

### Event Data Structure

```yaml
# Event metadata
apiVersion: mir.v1
kind: Event
metadata:
  name: device-power-online-12345
  namespace: default
  uid: 01234567-89ab-cdef-0123-456789abcdef
  createdAt: "2025-01-15T10:30:00Z"
# Event specification
spec:
  type: Normal           # Normal or Warning
  reason: DeviceOnline   # Machine-readable reason
  message: "Device power came online"
  payload:               # JSON payload with event details
    deviceId: "power"
    namespace: "default"
    timestamp: "2025-01-15T10:30:00Z"
  relatedObject:         # Reference to related system object
    apiVersion: mir.v1
    kind: Device
    name: power
    namespace: default
# Event status tracking
status:
  count: 1
  firstAt: "2025-01-15T10:30:00Z"
  lastAt: "2025-01-15T10:30:00Z"
```
