# Software Design Review (SRS)

## 1. Introduction

### 1.1 Problem

The difficulty for companies to monitor their systems

### 1.2 Purpose

Describe the purpose of the software being developed.

Enable highly secure and reliable communication between your Internet of Things (IoT) application and the devices it manages. Mir IoT Hub provides a cloud or on-premise hosted solution back end to connect virtually any device. Extend your solution from the cloud or your own server to the edge with per-device authentication, built-in device management, device observability, device data, and scaled provisioning.

Mir IoT Hub, act as your command centre:

- processes telemetry and commands with two ways communication
- automatically generate dashboards to observe data
- uses device twin for configuration management
- over the air update
- lightweight and infinitely scalable

### 1.3 Scope

Define the scope of the software, including what it will and will not do.

### 1.4 Definitions, Acronyms, and Abbreviations

Explain any terms and acronyms used throughout the document.

| Key | Definitions                   |
| --- | ----------------------------- |
| Mir | Russian orbital space station |

### 1.4 References

List any external resources referenced in the document.

### 1.5 Overview

Provide a high-level overview of the document's contents.

## 2. System Overview

### 2.1 System Architecture

Describe the overall architecture of the system, including major components and their interactions.

### 2.2 Software Environment

Outline the software environment, including the development platform, languages, and tools.

## 3. System Requirements

### 3.1 Functional Requirements

Detail the functional requirements the software must meet.

### 3.2 Non-Functional Requirements

Describe the non-functional requirements such as performance, security, and usability considerations.

## 4. System Design

### 4.1 High-Level Design

Provide a high-level design overview, including major system modules and their responsibilities.

#### 4.1.1 Twin Module

The Twin module is res#one json with same representation, but with timestamp instead of valu
ponsible for managing the device twin, which is a digital representation of a physical device. It stores device metadata, configuration, and state information, allowing the system to manage and monitor devices remotely.

Twin is the reconciliation logic between desired properties and reported properties.

##### 4.1.1.1 Twin Templates

A Twin Template is a JSON patch that defines the desired state of a device twin. It specifies the desired properties and their values that the device should have. It will then use labels to target a specific set of devices or a list of device ids. These are manually or via script triggered.

```json
{
  "templateId": "template-1",
  "labels": {
    "deviceType": "temperatureSensor"
  },
  "properties": {
    "desired": {
      "temperature": 25,
      "humidity": 50
    }
  }
}
```

##### 4.1.1.2 Twin AutoTemplates

A Twin AutoTemplate is a template that is associated with a set of labels for targets or list of devices id and an event from the message bus such as:

- Device created
- Device updated
- Device deleted
- Device connected
- Device disconnected
- ... could be events for every module, cpu_high, memory_low, etc

Together, this gives the ability to automatically apply a template to a device when it matches the labels. This is useful for devices that are created dynamically or for devices that need to be updated based on certain events.

```json
{
  "templateId": "template-1",
  "labels": {
    "deviceType": "temperatureSensor"
  },
  "properties": {
    "desired": {
      "temperature": 25,
      "humidity": 50
    }
  }
}
```

##### 4.1.1.3 Time Sync

- To make sure order of received properties is maintained, the code can
do addtionnal check using timestamps fields, or the framework could make sure to always put latest

```json
// one json object with all the properties
{
  "properties": {
    "desired": {
      "temperature": 25,
      "humidity": 50
    },
    "reported": {
      "temperature": 24,
      "humidity": 49
    }
  }
}
// one json with same representation, but with timestamp instead of value
{
  "properties": {
    "desired": {
      "temperature": "2022-01-01T12:00:00Z",
      "humidity": "2022-01-01T12:00:00Z"
    },
    "reported": {
      "temperature": "2022-01-01T12:00:00Z",
      "humidity": "2022-01-01T12:00:00Z"
    }
  }
}
```

A JSON alternative could be the timestamp field embeded next to the fields directly.

```json
{
  "properties": {
    "desired": {
    // one way of doing it.
      "temperature": 25,
      "__temperature": "2022-01-01T12:00:00Z",{
      },
      "humidity": "50",
      // allow more fields that could be leverage, like version
      // would version be useful? TODO think about it
      "__humidity": {
        "timestamp": "2022-01-01T12:00:00Z",
        "version": "2"
      }
    },
    "reported": {
       // one way of doing it. all nicely lined up under
       // feel it would be annoying the value like that
       // and  the user experience would be bad
      "temperature": {
        "value": 24,
        "timestamp": "2022-01-01T12:00:00Z"
      },
      "humidity": {
        "value": 49,
        "timestamp": "2022-01-01T12:00:00Z"
      }
    }
  }
}
```

### 4.2 Detailed Design

Dive into the detailed design of each module, including class diagrams, data flow diagrams, and other relevant details.

### 4.3 Data Design

Describe the data architecture, including database schemas, data models, and data storage considerations.

### 4.4 Security Design

Outline security measures, including authentication, authorization, data encryption, and secure data storage.

## 5. Interface Design

### 5.1 User Interface Design

Describe the user interface design, including mockups, user flow diagrams, and user interaction descriptions.

### 5.2 API/Service Interfaces

Detail any application programming interfaces (APIs) or external services the system will interface with.

## 6. Development and Deployment Strategy

### 6.1 Development Strategy

Outline the development methodology, version control system, branching strategy, and build process.

### 6.2 Deployment Strategy

Describe the deployment process, including deployment environments, continuous integration/continuous deployment (CI/CD) pipelines, and rollback plans.

## 7. Testing Plan

### 7.1 Testing Strategy

Outline the overall testing strategy, including types of testing (unit, integration, system, acceptance) and testing tools.

### 7.2 Test Cases

Provide examples of test cases for critical functionalities.

### 7.3 Quality Assurance

Describe quality assurance measures, including code reviews, static code analysis, and performance testing.

## 8. Project Management

### 8.1 Project Timeline

Outline the project timeline, including major milestones and estimated delivery dates.

### 8.2 Risk Management

Identify potential risks and outline mitigation strategies.

### 8.3 Resource Allocation

Detail resource allocation, including team roles and responsibilities.

## 9. Appendices

### 9.1 Glossary

Provide a glossary of terms used in the document.

### 9.2 Index

Include an index, if the document is lengthy.

## 10. Approval

### 10.1 Approval Signatures

Space for signatures from key stakeholders approving the document.

### 10.2 Revision History

Document the history of revisions to the SDD.
