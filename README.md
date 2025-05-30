# go-m2rest

[![GoDoc](https://godoc.org/github.com/florinel-chis/go-m2rest?status.svg)](https://godoc.org/github.com/florinel-chis/go-m2rest)

**A Modern and Robust Golang Library for Magento 2 REST API**

This Golang library provides a comprehensive interface for interacting with the Magento 2 REST API. Built with modern Go practices (1.21+), it features structured logging, concurrent operations, and extensive test coverage.

## Features

* **Modern Go Practices:** Uses Go 1.21+ features including the `any` type, structured errors with wrapping, and context support (coming soon)
* **Comprehensive Logging:** Structured logging with zerolog for better debugging and monitoring
* **Robust Error Handling:** Custom error types for common Magento 2 API scenarios
* **Authentication Support:** Multiple authentication methods including Integration tokens, Customer tokens, and Admin credentials
* **Extensive API Coverage:**
    * **Products:** Simple, configurable, bundle, grouped, and virtual products
    * **Categories:** Full hierarchy management and product assignments
    * **Attributes:** Create and manage product attributes with options
    * **Carts:** Guest and customer cart management with full checkout flow
    * **Orders:** Order retrieval, updates, and comments
    * **Stock Management:** Inventory updates and stock status
* **Performance Features:**
    * Automatic retry logic for transient errors
    * Concurrent operations support
    * Connection pooling via resty v2

## Getting Started

### Prerequisites

1. **Go 1.21+** - Required for modern Go features
2. **Magento 2 Instance** - With REST API enabled
3. **API Credentials** - Integration token or admin/customer credentials

### Installation

```bash
go get github.com/florinel-chis/go-m2rest
```

### Quick Start

```go
package main

import (
    "log"
    "github.com/florinel-chis/go-m2rest"
)

func main() {
    // Configure store connection
    storeConfig := &magento2.StoreConfig{
        Scheme:    "https",
        HostName:  "your-store.com",
        StoreCode: "default",
    }

    // Create API client with integration token
    client, err := magento2.NewAPIClientFromIntegration(
        storeConfig, 
        "your_integration_token",
    )
    if err != nil {
        log.Fatal(err)
    }

    // Create a simple product
    product := magento2.Product{
        Sku:            "test-product-001",
        Name:           "Test Product",
        Price:          29.99,
        TypeID:         "simple",
        AttributeSetID: 4,
        Status:         1,
        Visibility:     4,
        Weight:         1.0,
    }

    mProduct, err := magento2.CreateOrReplaceProduct(&product, true, client)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Created product: %s (ID: %d)", 
        mProduct.Product.Sku, 
        mProduct.Product.ID)
}
```

## Testing

The library includes comprehensive test coverage with both unit and functional tests.

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests from the tests directory
go test ./tests -v

# Run specific test
go test ./tests -run TestAdvancedProducts_VirtualProduct -v
```

### Test Configuration

Create a `.env` file in the project root:

```env
MAGENTO_HOST=http://your-magento-site.com
MAGENTO_BEARER_TOKEN=your_integration_token
MAGENTO_STORE_CODE=default
TEST_DEBUG=true
```

## Docker Support

The library includes Docker support for easy testing and development without installing Go locally.

### Quick Start with Docker

```bash
# One-line command to run all tests
docker run --rm -e MAGENTO_BEARER_TOKEN=your_token_here -e MAGENTO_HOST=http://magento.local ghcr.io/florinel-chis/go-m2rest:latest

# Or build and run locally
docker build -t go-m2rest . && docker run --rm -e MAGENTO_BEARER_TOKEN=your_token go-m2rest
```

### Using Docker Compose

```bash
# Run all tests
MAGENTO_BEARER_TOKEN=your_token docker-compose run --rm test

# Run specific test
MAGENTO_BEARER_TOKEN=your_token TEST_NAME=TestAdvancedProducts_VirtualProduct docker-compose run --rm test-specific

# Development mode with live code reload
docker-compose run --rm dev
```

### Using the Docker Helper Script

For even easier usage, use the included `docker-run.sh` script:

```bash
# Run all tests
MAGENTO_BEARER_TOKEN=your_token ./docker-run.sh test

# Run specific test
MAGENTO_BEARER_TOKEN=your_token TEST_NAME=TestAdvancedProducts_VirtualProduct ./docker-run.sh test-specific

# Quick connectivity test
MAGENTO_BEARER_TOKEN=your_token ./docker-run.sh test-quick

# Create 50 products
MAGENTO_BEARER_TOKEN=your_token ./docker-run.sh bulk-create 50

# Update stock from CSV
MAGENTO_BEARER_TOKEN=your_token ./docker-run.sh bulk-update stock_updates.csv

# Open shell in container
./docker-run.sh shell

# Build Docker images
./docker-run.sh build
```

### Docker Environment Variables

The Docker setup supports all the same environment variables as the native setup:

- `MAGENTO_HOST` - Magento URL (default: `http://magento.local`)
- `MAGENTO_BEARER_TOKEN` - Integration token (required)
- `MAGENTO_STORE_CODE` - Store code (default: `all`)
- `TEST_DEBUG` - Enable debug logging (default: `true`)
- `TEST_NAME` - Specific test to run (for test-specific command)

### Building Custom Docker Image

You can customize the Dockerfile to include your own environment variables:

```dockerfile
# In Dockerfile, update the ENV section
ENV MAGENTO_HOST=http://your-magento.com \
    MAGENTO_BEARER_TOKEN=your_permanent_token \
    MAGENTO_STORE_CODE=your_store
```

Then build and run:

```bash
docker build -t my-m2rest .
docker run --rm my-m2rest
```

## Bulk Operations

The library includes utilities for bulk operations. See the `scripts/` directory for examples.

### Bulk Product Creation and Stock Update

```bash
cd scripts
./run_bulk_update.sh stock_updates.csv 100 10 both
```

This will:
- Create 100 simple products
- Update their stock from the CSV file
- Use 10 concurrent operations

## Advanced Usage

### Working with Different Product Types

```go
// Virtual Product (no shipping)
virtualProduct := magento2.Product{
    Sku:    "virtual-service-001",
    Name:   "Virtual Service",
    TypeID: "virtual",
    Price:  99.99,
    // No weight needed for virtual products
}

// Grouped Product
groupedProduct := magento2.Product{
    Sku:    "grouped-product-001",
    Name:   "Product Bundle",
    TypeID: "grouped",
    // Price comes from associated products
}

// Configurable Product
configurableProduct := magento2.Product{
    Sku:    "configurable-001",
    Name:   "T-Shirt",
    TypeID: "configurable",
    // Requires attribute configuration
}
```

### Cart Operations

```go
// Create guest cart
guestCart, err := magento2.NewGuestCartFromAPIClient(client)

// Add items
item := magento2.CartItem{
    Sku:     "test-product-001",
    Qty:     2,
    QuoteID: guestCart.QuoteID,
}
err = guestCart.AddItems([]magento2.CartItem{item})

// Estimate shipping
shippingAddr := &magento2.ShippingAddress{
    Address: magento2.Address{
        CountryID: "US",
        Postcode:  "10001",
        City:      "New York",
        Street:    []string{"123 Main St"},
        Firstname: "John",
        Lastname:  "Doe",
        Telephone: "555-1234",
        Email:     "john@example.com",
    },
}
carriers, err := guestCart.EstimateShippingCarrier(shippingAddr)
```

## API Coverage

### Products API
- `CreateOrReplaceProduct()` - Create or update products
- `GetProductBySKU()` - Retrieve product details
- `UpdateProductStockItemBySKU()` - Update inventory
- Support for all product types

### Categories API
- `CreateCategory()` - Create categories
- `GetCategoryByID()` - Retrieve category details
- `GetCategoriesList()` - List all categories
- `AssignProductsToCategoryByID()` - Manage product assignments

### Attributes API
- `CreateAttribute()` - Create product attributes
- `GetAttributeByCode()` - Retrieve attribute details
- `AddOption()` - Add dropdown options
- Attribute set and group management

### Cart API
- Guest and customer cart support
- Add/remove items
- Shipping and payment estimation
- Order placement

### Orders API
- `GetOrderByIncrementID()` - Retrieve orders
- `UpdateOrderEntity()` - Update order status
- `AddOrderComment()` - Add order notes

## Project Structure

```
go-m2rest/
├── *.go                 # Main library files
├── tests/              # Test files
│   ├── functional_test.go
│   ├── advanced_product_test.go
│   └── test_config.go
├── scripts/            # Utility scripts
│   ├── bulk_product_update.go
│   └── run_bulk_update.sh
├── .env.example        # Example configuration
└── README.md           # This file
```

## Error Handling

The library provides structured error types:

```go
product, err := magento2.GetProductBySKU("non-existent", client)
if err != nil {
    if err == magento2.ErrNotFound {
        // Handle not found case
    } else {
        // Handle other errors
    }
}
```

## Logging

The library uses zerolog for structured logging:

```go
// Logs are automatically structured with context
// Output includes API endpoints, payloads, and responses
```

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

### Development Setup

```bash
# Clone the repository
git clone https://github.com/florinel-chis/go-m2rest.git
cd go-m2rest

# Install dependencies
go mod download

# Run tests
go test ./...

# Run linter (optional)
golangci-lint run
```

## License

This project is licensed under the [MIT License](LICENSE).

## Support

For issues, feature requests, or questions:
- Open an issue on [GitHub](https://github.com/florinel-chis/go-m2rest/issues)
- Check the [GoDoc](https://godoc.org/github.com/florinel-chis/go-m2rest) for API documentation

## Changelog

### Recent Updates
- Added support for Go 1.21+ features
- Migrated to resty v2 for better performance
- Added structured logging with zerolog
- Improved test coverage and organization
- Added bulk operations utilities
- Enhanced error handling with wrapped errors

---

**Build robust Magento 2 integrations with modern Go!**