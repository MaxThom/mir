#!/bin/bash

# NATS TLS Certificate Generator v2
# Supports both Server-Only and Mutual TLS authentication
# with configurable expiration and SAN options

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
MODE="server-only"
NUM_SERVERS=1
NUM_CLIENTS=0
OUTPUT_DIR=".certs"
CA_DAYS=3650  # 10 years
CERT_DAYS=365 # 1 year
CLEAN=false
USE_SAN=true
CUSTOM_CA_CERT=""
CUSTOM_CA_KEY=""
KUBERNETES=false
AUTO_DETECTED_CA=false
PREFIX=""
declare -a SAN_LIST=()

# Default SAN if none provided
DEFAULT_SAN="DNS:localhost,DNS:*.localhost,DNS:local-nats,IP:127.0.0.1,IP:::1"

# Function to display usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

NATS TLS Certificate Generator - Generate certificates for Server-Only or Mutual TLS

OPTIONS:
    -m, --mode MODE           TLS mode: server-only (default) or mutual
    -s, --servers N           Number of server certificates to generate (default: 1)
    -c, --clients N           Number of client certificates to generate (default: 0)
                             Note: Setting clients > 0 automatically sets mode to mutual

    --ca-cert PATH           Path to existing CA certificate (must be used with --ca-key)
    --ca-key PATH            Path to existing CA private key (must be used with --ca-cert)
    --ca-days N              CA certificate validity in days (default: 3650)
    --cert-days N            Server/client certificate validity in days (default: 365)

    --san "DNS:...,IP:..."   Subject Alternative Names for server certificates
                             Can be specified multiple times for different servers
                             Default: DNS:localhost,DNS:*.localhost,DNS:local-nats,IP:127.0.0.1,IP:::1
    --no-san                 Generate server certificates without SAN extensions (CN only)

    -d, --dir PATH           Output directory (default: .certs)
    --clean                  Remove existing certificates before generating new ones

    -p, --prefix PREFIX      Add prefix to server and client certificate files (CA remains unchanged)

    -k, --kubernetes         Generate Kubernetes TLS secrets for all certificates

    -h, --help               Display this help message

EXAMPLES:
    # Server-only mode with 2 servers
    $0 -m server-only -s 2

    # Mutual TLS with 3 servers and 5 clients
    $0 -m mutual -s 3 -c 5

    # Custom expiration times
    $0 -s 2 -c 3 --ca-days 7300 --cert-days 730

    # Custom SAN for servers
    $0 -s 2 --san "DNS:nats.example.com,DNS:*.nats.example.com,IP:10.0.0.1"

    # Different SANs for each server
    $0 -s 3 --san "DNS:server1.example.com,IP:10.0.0.1" \\
              --san "DNS:server2.example.com,IP:10.0.0.2" \\
              --san "DNS:server3.example.com,IP:10.0.0.3"

    # No SAN (CN only)
    $0 -s 2 --no-san

    # Use existing CA
    $0 -m mutual -s 1 -c 2 --ca-cert /path/to/ca.crt --ca-key /path/to/ca.key

    # Generate with Kubernetes secrets
    $0 -m mutual -s 2 -c 3 --kubernetes

    # Generate with prefix for different environments
    $0 -p prod -s 2 -c 3
    $0 -p staging -s 2 -c 3
    # Results in: ca.crt, prod-server-1.crt, staging-server-1.crt, etc.

EOF
    exit 0
}

# Function to print error messages
error() {
    echo -e "${RED}Error: $1${NC}" >&2
    exit 1
}

# Function to print success messages
success() {
    echo -e "${GREEN}$1${NC}"
}

# Function to print info messages
info() {
    echo -e "${BLUE}$1${NC}"
}

# Function to print warning messages
warning() {
    echo -e "${YELLOW}$1${NC}"
}

# Parse command-line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -m|--mode)
            MODE="$2"
            if [[ "$MODE" != "server-only" && "$MODE" != "mutual" ]]; then
                error "Invalid mode: $MODE. Use 'server-only' or 'mutual'"
            fi
            shift 2
            ;;
        -s|--servers)
            NUM_SERVERS="$2"
            if ! [[ "$NUM_SERVERS" =~ ^[0-9]+$ ]] || [ "$NUM_SERVERS" -lt 1 ]; then
                error "Number of servers must be a positive integer"
            fi
            shift 2
            ;;
        -c|--clients)
            NUM_CLIENTS="$2"
            if ! [[ "$NUM_CLIENTS" =~ ^[0-9]+$ ]]; then
                error "Number of clients must be a non-negative integer"
            fi
            if [ "$NUM_CLIENTS" -gt 0 ]; then
                MODE="mutual"
            fi
            shift 2
            ;;
        --ca-cert)
            CUSTOM_CA_CERT="$2"
            shift 2
            ;;
        --ca-key)
            CUSTOM_CA_KEY="$2"
            shift 2
            ;;
        --ca-days)
            CA_DAYS="$2"
            if ! [[ "$CA_DAYS" =~ ^[0-9]+$ ]] || [ "$CA_DAYS" -lt 1 ]; then
                error "CA days must be a positive integer"
            fi
            shift 2
            ;;
        --cert-days)
            CERT_DAYS="$2"
            if ! [[ "$CERT_DAYS" =~ ^[0-9]+$ ]] || [ "$CERT_DAYS" -lt 1 ]; then
                error "Certificate days must be a positive integer"
            fi
            shift 2
            ;;
        --san)
            SAN_LIST+=("$2")
            shift 2
            ;;
        --no-san)
            USE_SAN=false
            shift
            ;;
        -d|--dir)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        --clean)
            CLEAN=true
            shift
            ;;
        -k|--kubernetes)
            KUBERNETES=true
            shift
            ;;
        -p|--prefix)
            PREFIX="$2"
            shift 2
            ;;
        -h|--help)
            usage
            ;;
        *)
            error "Unknown option: $1"
            ;;
    esac
done

# Validate arguments
if [ "$USE_SAN" = false ] && [ ${#SAN_LIST[@]} -gt 0 ]; then
    error "Cannot use both --no-san and --san options"
fi

if [ -n "$CUSTOM_CA_CERT" ] || [ -n "$CUSTOM_CA_KEY" ]; then
    if [ -z "$CUSTOM_CA_CERT" ] || [ -z "$CUSTOM_CA_KEY" ]; then
        error "Both --ca-cert and --ca-key must be provided when using existing CA"
    fi
    if [ ! -f "$CUSTOM_CA_CERT" ]; then
        error "CA certificate file not found: $CUSTOM_CA_CERT"
    fi
    if [ ! -f "$CUSTOM_CA_KEY" ]; then
        error "CA key file not found: $CUSTOM_CA_KEY"
    fi
fi

# Auto-detect existing CA if not specified via flags (before display for correct reporting)
if [ -z "$CUSTOM_CA_CERT" ] && [ -z "$CUSTOM_CA_KEY" ] && [ "$CLEAN" = false ]; then
    if [ -f "$OUTPUT_DIR/ca/ca.crt" ] && [ -f "$OUTPUT_DIR/ca/ca.key" ]; then
        # Verify the existing CA is valid
        if openssl x509 -in "$OUTPUT_DIR/ca/ca.crt" -noout 2>/dev/null && \
           openssl rsa -in "$OUTPUT_DIR/ca/ca.key" -check -noout 2>/dev/null; then
            CUSTOM_CA_CERT="$OUTPUT_DIR/ca/ca.crt"
            CUSTOM_CA_KEY="$OUTPUT_DIR/ca/ca.key"
            AUTO_DETECTED_CA=true
        fi
    fi
fi

# Display configuration
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}           TLS Certificate Generator${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo
info "Configuration:"
echo "  Mode:                 $MODE"
echo "  Server certificates:  $NUM_SERVERS"
if [ "$MODE" = "mutual" ]; then
    echo "  Client certificates:  $NUM_CLIENTS"
fi
echo "  Output directory:     $OUTPUT_DIR"
echo "  CA validity:          $CA_DAYS days"
echo "  Cert validity:        $CERT_DAYS days"
if [ "$USE_SAN" = true ]; then
    if [ ${#SAN_LIST[@]} -eq 0 ]; then
        echo "  SAN:                  Default (localhost, 127.0.0.1, ::1)"
    else
        echo "  SAN:                  Custom (${#SAN_LIST[@]} configuration(s))"
    fi
else
    echo "  SAN:                  Disabled (CN only)"
fi
if [ -n "$CUSTOM_CA_CERT" ]; then
    if [ "$AUTO_DETECTED_CA" = true ]; then
        echo "  Using existing CA:    Yes (auto-detected)"
    else
        echo "  Using existing CA:    Yes (specified)"
    fi
fi
if [ "$KUBERNETES" = true ]; then
    echo "  Kubernetes secrets:   Yes"
fi
if [ -n "$PREFIX" ]; then
    echo "  Certificate prefix:   $PREFIX"
fi
echo

# Create directory structure
if [ "$CLEAN" = true ] && [ -d "$OUTPUT_DIR" ]; then
    warning "Removing existing certificates in $OUTPUT_DIR..."
    rm -rf "$OUTPUT_DIR"
fi

mkdir -p "$OUTPUT_DIR"/{ca,servers}
if [ "$MODE" = "mutual" ] && [ "$NUM_CLIENTS" -gt 0 ]; then
    mkdir -p "$OUTPUT_DIR/clients"
fi

cd "$OUTPUT_DIR"

# Generate or copy CA certificate
info "[1/4] Setting up CA certificate..."
if [ -n "$CUSTOM_CA_CERT" ]; then
    if [ "$AUTO_DETECTED_CA" = true ]; then
        echo "Using auto-detected CA certificate..."
        # Already in place, no need to copy
    else
        echo "Using specified CA certificate..."
        cp "$CUSTOM_CA_CERT" ca/ca.crt
        cp "$CUSTOM_CA_KEY" ca/ca.key
    fi
    # Verify the CA certificate
    openssl x509 -in ca/ca.crt -noout -subject -enddate | sed 's/^/  /'
else
    echo "Generating new CA certificate..."
    openssl genrsa -out ca/ca.key 4096 2>/dev/null
    openssl req -new -x509 -days "$CA_DAYS" -key ca/ca.key -out ca/ca.crt \
        -subj "/C=US/ST=CA/L=San Francisco/O=Mir IoT/OU=Certificate Authority/CN=Mir Root CA" \
        2>/dev/null
    success "✓ CA certificate generated (valid for $CA_DAYS days)"
fi

# Generate server certificates
info "[2/4] Generating server certificates..."
for i in $(seq 1 "$NUM_SERVERS"); do
    # Determine server file name with prefix
    if [ -n "$PREFIX" ]; then
        SERVER_NAME="${PREFIX}-server-$i"
    else
        SERVER_NAME="server-$i"
    fi

    echo "  Generating $SERVER_NAME..."

    # Generate private key
    openssl genrsa -out "servers/${SERVER_NAME}.key" 4096 2>/dev/null

    # Generate certificate request
    openssl req -new -key "servers/${SERVER_NAME}.key" -out "servers/${SERVER_NAME}.csr" \
        -subj "/C=US/ST=CA/L=San Francisco/O=Mir IoT/OU=NATS Server/CN=${SERVER_NAME}" \
        2>/dev/null

    # Handle SAN
    if [ "$USE_SAN" = true ]; then
        # Determine which SAN to use
        if [ ${#SAN_LIST[@]} -eq 0 ]; then
            # Use default SAN
            CURRENT_SAN="$DEFAULT_SAN"
        elif [ ${#SAN_LIST[@]} -eq 1 ]; then
            # Use same SAN for all servers
            CURRENT_SAN="${SAN_LIST[0]}"
        else
            # Use different SAN for each server
            if [ $i -le ${#SAN_LIST[@]} ]; then
                CURRENT_SAN="${SAN_LIST[$((i-1))]}"
            else
                # Use the last SAN for extra servers
                CURRENT_SAN="${SAN_LIST[-1]}"
            fi
        fi

        # Create extensions file
        echo "subjectAltName = $CURRENT_SAN" > "servers/${SERVER_NAME}-ext.cnf"

        # Sign the certificate with SAN
        openssl x509 -req -days "$CERT_DAYS" -in "servers/${SERVER_NAME}.csr" \
            -CA ca/ca.crt -CAkey ca/ca.key -CAcreateserial \
            -out "servers/${SERVER_NAME}.crt" -extfile "servers/${SERVER_NAME}-ext.cnf" 2>/dev/null

        # Clean up
        rm -f "servers/${SERVER_NAME}-ext.cnf"
    else
        # Sign the certificate without SAN
        openssl x509 -req -days "$CERT_DAYS" -in "servers/${SERVER_NAME}.csr" \
            -CA ca/ca.crt -CAkey ca/ca.key -CAcreateserial \
            -out "servers/${SERVER_NAME}.crt" 2>/dev/null
    fi

    # Clean up CSR
    rm -f "servers/${SERVER_NAME}.csr"

    # Set permissions
    chmod 600 "servers/${SERVER_NAME}.key"
    chmod 644 "servers/${SERVER_NAME}.crt"
done
success "✓ Generated $NUM_SERVERS server certificate(s)"

# Generate client certificates for mutual TLS
if [ "$MODE" = "mutual" ] && [ "$NUM_CLIENTS" -gt 0 ]; then
    info "[3/4] Generating client certificates..."
    for i in $(seq 1 "$NUM_CLIENTS"); do
        # Determine client file name with prefix
        if [ -n "$PREFIX" ]; then
            CLIENT_NAME="${PREFIX}-client-$i"
        else
            CLIENT_NAME="client-$i"
        fi

        echo "  Generating $CLIENT_NAME..."

        # Generate private key
        openssl genrsa -out "clients/${CLIENT_NAME}.key" 4096 2>/dev/null

        # Generate certificate request
        openssl req -new -key "clients/${CLIENT_NAME}.key" -out "clients/${CLIENT_NAME}.csr" \
            -subj "/C=US/ST=CA/L=San Francisco/O=Mir IoT/OU=NATS Client/CN=${CLIENT_NAME}" \
            2>/dev/null

        # Sign the certificate (clients don't need SAN)
        openssl x509 -req -days "$CERT_DAYS" -in "clients/${CLIENT_NAME}.csr" \
            -CA ca/ca.crt -CAkey ca/ca.key -CAcreateserial \
            -out "clients/${CLIENT_NAME}.crt" 2>/dev/null

        # Clean up
        rm -f "clients/${CLIENT_NAME}.csr"

        # Set permissions
        chmod 600 "clients/${CLIENT_NAME}.key"
        chmod 644 "clients/${CLIENT_NAME}.crt"
    done
    success "✓ Generated $NUM_CLIENTS client certificate(s)"
else
    info "[3/4] Skipping client certificates (not needed for $MODE mode)"
fi

# Clean up
rm -f ca/*.srl

# Summary
info "[4/4] Certificate generation complete!"
echo
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}                    SUMMARY${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo
echo -e "${GREEN}📁 Output Directory: $(pwd)${NC}"
echo
echo -e "${YELLOW}🔐 CA Certificate:${NC}"
echo "   • ca/ca.crt - CA public certificate (share with clients)"
echo "   • ca/ca.key - CA private key (keep secret!)"
openssl x509 -in ca/ca.crt -noout -subject -enddate | sed 's/^/     /'
echo

echo -e "${YELLOW}🖥️  Server Certificates ($NUM_SERVERS):${NC}"
for i in $(seq 1 "$NUM_SERVERS"); do
    # Determine server file name with prefix
    if [ -n "$PREFIX" ]; then
        SERVER_NAME="${PREFIX}-server-$i"
    else
        SERVER_NAME="server-$i"
    fi

    echo "   • servers/${SERVER_NAME}.crt - Server $i public certificate"
    echo "   • servers/${SERVER_NAME}.key - Server $i private key"
    openssl x509 -in "servers/${SERVER_NAME}.crt" -noout -subject -enddate | sed 's/^/     /'
    if [ "$USE_SAN" = true ]; then
        # Display SAN if present
        SAN_INFO=$(openssl x509 -in "servers/${SERVER_NAME}.crt" -noout -text | grep -A1 "Subject Alternative Name" | tail -1 | sed 's/^[ \t]*//')
        if [ -n "$SAN_INFO" ]; then
            echo "     SAN: $SAN_INFO"
        fi
    fi
    echo
done

if [ "$MODE" = "mutual" ] && [ "$NUM_CLIENTS" -gt 0 ]; then
    echo -e "${YELLOW}👤 Client Certificates ($NUM_CLIENTS):${NC}"
    for i in $(seq 1 "$NUM_CLIENTS"); do
        # Determine client file name with prefix
        if [ -n "$PREFIX" ]; then
            CLIENT_NAME="${PREFIX}-client-$i"
        else
            CLIENT_NAME="client-$i"
        fi

        echo "   • clients/${CLIENT_NAME}.crt - Client $i public certificate"
        echo "   • clients/${CLIENT_NAME}.key - Client $i private key"
        openssl x509 -in "clients/${CLIENT_NAME}.crt" -noout -subject -enddate | sed 's/^/     /'
        [ $i -lt "$NUM_CLIENTS" ] && echo
    done
    echo
fi

# Generate Kubernetes secrets if requested
if [ "$KUBERNETES" = true ]; then
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}             KUBERNETES SECRETS${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo

    # Create k8s directory for YAML manifests
    mkdir -p k8s

    info "Generating Kubernetes TLS secrets..."
    echo

    # Generate CA secret
    echo -e "${YELLOW}CA Secret:${NC}"
    kubectl create secret generic ca-tls-secret \
        --from-file=ca.crt=ca/ca.crt \
        --dry-run=client -o yaml > k8s/ca-tls.secret.yaml 2>/dev/null
    echo "  → k8s/ca.secret.yaml"
    echo

    # Generate server secrets
    echo -e "${YELLOW}Server TLS Secrets:${NC}"
    for i in $(seq 1 "$NUM_SERVERS"); do
        # Determine server file name with prefix
        if [ -n "$PREFIX" ]; then
            SERVER_NAME="${PREFIX}-server-$i"
        else
            SERVER_NAME="server-$i"
        fi

        # Create YAML manifest
        kubectl create secret tls ${SERVER_NAME}-tls \
            --cert=servers/${SERVER_NAME}.crt \
            --key=servers/${SERVER_NAME}.key \
            --dry-run=client -o yaml > k8s/${SERVER_NAME}-tls.secret.yaml 2>/dev/null
        echo "  → k8s/${SERVER_NAME}-tls.secret.yaml"
    done
    echo

    # Generate client secrets for mutual TLS
    if [ "$MODE" = "mutual" ] && [ "$NUM_CLIENTS" -gt 0 ]; then
        echo -e "${YELLOW}Client TLS Secrets:${NC}"
        for i in $(seq 1 "$NUM_CLIENTS"); do
            # Determine client file name with prefix
            if [ -n "$PREFIX" ]; then
                CLIENT_NAME="${PREFIX}-client-$i"
            else
                CLIENT_NAME="client-$i"
            fi

            # Create YAML manifest
            kubectl create secret tls ${CLIENT_NAME}-tls \
                --cert=clients/${CLIENT_NAME}.crt \
                --key=clients/${CLIENT_NAME}.key \
                --dry-run=client -o yaml > k8s/${CLIENT_NAME}-tls.secret.yaml 2>/dev/null
            echo "  → k8s/${CLIENT_NAME}-tls.secret.yaml"
        done
        echo
    fi

    echo -e "${GREEN}✓ Kubernetes secret manifests generated in k8s/ directory${NC}"
    echo
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo
fi

success "✅ Certificate generation completed successfully!"
echo
echo "Next steps:"
echo "  1. Copy certificates to your servers"
echo "  2. Update servers configuration with TLS settings"
echo "  3. Distribute CA certificate to clients"
if [ "$MODE" = "mutual" ] && [ "$NUM_CLIENTS" -gt 0 ]; then
    echo "  4. Distribute client certificates to authorized clients"
fi
if [ "$KUBERNETES" = true ]; then
    echo "  5. Apply Kubernetes secrets: kubectl apply -f k8s/"
fi
echo
