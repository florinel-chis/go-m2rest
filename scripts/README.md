# Bulk Product Update Scripts

This directory contains scripts for bulk product operations in Magento 2.

## Files

- `bulk_product_update.go` - Main Go script for creating products and updating stock
- `run_bulk_update.sh` - Bash wrapper script for easy execution
- `stock_updates_example.csv` - Example CSV file showing the expected format

## Usage

### Prerequisites

1. Set up your environment variables in `../.env`:
```bash
MAGENTO_HOST=http://jti.local
MAGENTO_BEARER_TOKEN=your_bearer_token_here
MAGENTO_STORE_CODE=all
```

### Running the Script

The script can operate in three modes:

#### 1. Create Products and Update Stock (Default)
```bash
./run_bulk_update.sh stock_updates.csv 100 5 both
```

This will:
- Create 100 simple products
- Update their stock quantities from the CSV file

#### 2. Create Products Only
```bash
./run_bulk_update.sh stock_updates.csv 100 5 create
```

This will:
- Create 100 simple products
- Save their SKUs to the CSV file with random quantities

#### 3. Update Stock Only
```bash
./run_bulk_update.sh stock_updates.csv 100 5 update
```

This will:
- Read SKUs and quantities from the CSV file
- Update stock for existing products

### Parameters

1. **CSV File** (default: `stock_updates.csv`) - CSV file with SKU and quantity columns
2. **Product Count** (default: 100) - Number of products to create
3. **Concurrent Operations** (default: 5) - Number of parallel API requests
4. **Mode** (default: both) - Operation mode: `create`, `update`, or `both`

### CSV Format

The CSV file should have two columns:
```csv
sku,qty
bulk-product-1748595659-1,50
bulk-product-1748595659-2,75
bulk-product-1748595659-3,100
```

### Direct Go Script Usage

You can also run the Go script directly:

```bash
# Build first
go build bulk_product_update.go

# Create products only
./bulk_product_update -create-only -count=50 -csv=my_products.csv

# Update stock only
./bulk_product_update -update-only -csv=my_products.csv

# Both operations
./bulk_product_update -count=100 -csv=my_products.csv -concurrent=10
```

### Command Line Flags

- `-csv` - CSV file path (default: `stock_updates.csv`)
- `-create-only` - Only create products, don't update stock
- `-update-only` - Only update stock, don't create products
- `-concurrent` - Number of concurrent operations (default: 5)
- `-count` - Number of products to create (default: 100)

## Product Details

Created products will have:
- **SKU**: `bulk-product-[timestamp]-[number]`
- **Name**: `Bulk Product [number]`
- **Price**: $9.99 - $19.99 (varies)
- **Type**: Simple product
- **Status**: Enabled
- **Visibility**: Catalog, Search
- **Weight**: 1.0

## Performance

With the default concurrency of 5:
- Creating 100 products: ~20-30 seconds
- Updating 100 stock quantities: ~20-30 seconds

Increase concurrency for faster operations, but be mindful of API rate limits.

## Error Handling

- Failed operations are logged but don't stop the process
- Final summary shows successful vs failed operations
- Check console output for specific error details

## Example Output

```
Bulk Product Update Script
=========================
CSV File: stock_updates.csv
Product Count: 100
Concurrent Operations: 5

Building bulk update script...
Creating 100 products and updating stock from stock_updates.csv...
12:30PM INF Product created id=1234 progress=1 sku=bulk-product-1748595659-1 total=100
12:30PM INF Product created id=1235 progress=2 sku=bulk-product-1748595659-2 total=100
...
12:31PM INF Product creation completed created=100 failed=0
12:31PM INF Stock updated progress=1 qty=50 sku=bulk-product-1748595659-1 total=100
...
12:32PM INF Stock update completed failed=0 updated=100
12:32PM INF Bulk operations completed

Done!
```