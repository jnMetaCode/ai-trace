#!/bin/bash
# AI-Trace Basic Workflow Example
# This script demonstrates the complete workflow using curl commands

set -e

BASE_URL="http://localhost:8006"
API_KEY="test-api-key-12345"

echo "=============================================="
echo "AI-Trace Basic Workflow (curl)"
echo "=============================================="

# Step 1: Health Check
echo ""
echo "1. Checking server health..."
curl -s "${BASE_URL}/health" | jq .

# Step 2: Create a Trace
echo ""
echo "2. Creating a trace..."
TRACE_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/v1/traces" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: ${API_KEY}" \
  -d '{
    "name": "curl-example-trace",
    "tenant_id": "default",
    "metadata": {
      "source": "curl-example"
    }
  }')

echo "$TRACE_RESPONSE" | jq .

TRACE_ID=$(echo "$TRACE_RESPONSE" | jq -r '.trace_id')
echo "Trace ID: $TRACE_ID"

# Step 3: Add Input Event
echo ""
echo "3. Adding input event..."
INPUT_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/v1/events/ingest" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: ${API_KEY}" \
  -d "{
    \"trace_id\": \"${TRACE_ID}\",
    \"event_type\": \"input\",
    \"payload\": {
      \"model\": \"gpt-4\",
      \"messages\": [
        {\"role\": \"user\", \"content\": \"What is AI-Trace?\"}
      ]
    }
  }")

echo "$INPUT_RESPONSE" | jq .

# Step 4: Add Output Event
echo ""
echo "4. Adding output event..."
OUTPUT_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/v1/events/ingest" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: ${API_KEY}" \
  -d "{
    \"trace_id\": \"${TRACE_ID}\",
    \"event_type\": \"output\",
    \"payload\": {
      \"response\": \"AI-Trace is an open-source platform for tamper-proof AI attestation.\",
      \"model\": \"gpt-4\",
      \"usage\": {
        \"total_tokens\": 50
      }
    }
  }")

echo "$OUTPUT_RESPONSE" | jq .

# Step 5: Commit to Certificate
echo ""
echo "5. Committing to certificate..."
CERT_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/v1/certs/commit" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: ${API_KEY}" \
  -d "{
    \"trace_id\": \"${TRACE_ID}\",
    \"evidence_level\": \"L1\"
  }")

echo "$CERT_RESPONSE" | jq .

CERT_ID=$(echo "$CERT_RESPONSE" | jq -r '.cert_id')
echo "Certificate ID: $CERT_ID"

# Step 6: Verify Certificate
echo ""
echo "6. Verifying certificate..."
VERIFY_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/v1/certs/verify" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: ${API_KEY}" \
  -d "{
    \"cert_id\": \"${CERT_ID}\"
  }")

echo "$VERIFY_RESPONSE" | jq .

# Step 7: Get Certificate Details
echo ""
echo "7. Getting certificate details..."
curl -s "${BASE_URL}/api/v1/certs/${CERT_ID}" \
  -H "X-API-Key: ${API_KEY}" | jq .

# Step 8: Generate Proof
echo ""
echo "8. Generating minimal disclosure proof..."
PROOF_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/v1/certs/${CERT_ID}/prove" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: ${API_KEY}" \
  -d '{
    "event_indices": [0],
    "disclosed_fields": ["model"]
  }')

echo "$PROOF_RESPONSE" | jq .

echo ""
echo "=============================================="
echo "Workflow completed successfully!"
echo "=============================================="
echo ""
echo "Summary:"
echo "  Trace ID:       $TRACE_ID"
echo "  Certificate ID: $CERT_ID"
echo "  Valid:          $(echo "$VERIFY_RESPONSE" | jq -r '.valid')"
