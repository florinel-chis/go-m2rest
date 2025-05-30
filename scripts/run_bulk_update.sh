#!/bin/bash

# Load environment variables from parent directory
if [ -f "../.env" ]; then
    export $(cat ../.env | grep -v '^#' | xargs)
fi

# Default values
CSV_FILE="${1:-stock_updates.csv}"
PRODUCT_COUNT="${2:-100}"
CONCURRENT="${3:-5}"

echo "Bulk Product Update Script"
echo "========================="
echo "CSV File: $CSV_FILE"
echo "Product Count: $PRODUCT_COUNT"
echo "Concurrent Operations: $CONCURRENT"
echo ""

# Check if required environment variables are set
if [ -z "$MAGENTO_HOST" ] || [ -z "$MAGENTO_BEARER_TOKEN" ]; then
    echo "Error: MAGENTO_HOST and MAGENTO_BEARER_TOKEN must be set"
    echo "Please set them in ../.env or export them"
    exit 1
fi

# Build the Go script
echo "Building bulk update script..."
go build -o bulk_product_update bulk_product_update.go

if [ $? -ne 0 ]; then
    echo "Failed to build script"
    exit 1
fi

# Run based on arguments
case "${4:-both}" in
    "create")
        echo "Creating $PRODUCT_COUNT products only..."
        ./bulk_product_update -create-only -count=$PRODUCT_COUNT -concurrent=$CONCURRENT -csv=$CSV_FILE
        ;;
    "update")
        echo "Updating stock from $CSV_FILE only..."
        ./bulk_product_update -update-only -csv=$CSV_FILE -concurrent=$CONCURRENT
        ;;
    *)
        echo "Creating $PRODUCT_COUNT products and updating stock from $CSV_FILE..."
        ./bulk_product_update -count=$PRODUCT_COUNT -concurrent=$CONCURRENT -csv=$CSV_FILE
        ;;
esac

# Cleanup
rm -f bulk_product_update

echo ""
echo "Done!"