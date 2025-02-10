# go-m2rest

[![GoDoc](https://godoc.org/github.com/florinel-chis/go-m2rest?status.svg)](https://godoc.org/github.com/florinel-chis/go-m2rest)

**A Verbose and Robust Golang Library for Magento 2 REST API Interaction**

This Golang library, **`go-m2rest`**, provides a comprehensive and well-documented interface for interacting with the Magento 2 REST API. Built with best practices in mind, including verbose logging for debugging and error tracking, this library aims to simplify and streamline your Magento 2 REST API integrations in Go.

## Features

* **Verbose Logging:** Every API request and response is logged with detailed information, making debugging and monitoring significantly easier.
* **Robust Error Handling:**  Provides structured error handling, including custom error types for common Magento 2 API issues (like `ErrNotFound`, `ItemNotFoundError`, `ErrNoPointer`, `ErrBadRequest`).
* **Authentication Support:** Supports both Customer and Administrator API authentication methods, including token-based integration.
* **Comprehensive API Coverage:** Implements a wide range of Magento 2 REST API endpoints, including:
    * **Products:** Create, retrieve, update products, manage stock items.
    * **Categories:** Create, retrieve, update categories, assign products to categories.
    * **Attributes & Attribute Sets:** Create, retrieve, update attributes and attribute sets, assign attributes to sets and groups.
    * **Configurable Products:** Create and manage configurable products, set options, add child products.
    * **Orders:** Retrieve orders by increment ID, update order entities, add order comments.
    * **Carts:** Guest and Customer cart management, add items, estimate shipping and payment methods, place orders, delete cart items.
* **Well-Structured Code:** Follows Go best practices with clear package structure, idiomatic Go code, and comprehensive documentation.
* **Retries:** Implements automatic retry logic for transient errors like `503 Service Unavailable` and `500 Internal Server Error`.

## Getting Started

### Prerequisites

1. **Go Installation:** Ensure you have Go 1.16 or later installed. You can download it from [https://go.dev/dl/](https://go.dev/dl/).
2. **Magento 2 Setup:** You need a running Magento 2 instance with the REST API enabled.
3. **Magento 2 API Credentials:** Depending on the API you want to use (Admin or Customer), you'll need to set up the necessary integrations or customer accounts in Magento 2 to obtain authentication credentials.

### Installation

To install **`go-m2rest`**, use `go get`:

```bash
go get github.com/florinel-chis/go-m2rest
```

### Usage

Here's a basic example of how to use **`go-m2rest`** to create a product in Magento 2 using Administrator API authentication:

```go
package main

import (
	"log"

	"github.com/florinel-chis/go-m2rest"
)

func main() {
	logger := log.New(log.Writer(), "admin-api-example: ", log.LstdFlags|log.Lshortfile)
	magento2.SetLogger(logger) // Set a custom logger for verbose output

	// 1. Configure Store Settings
	storeConfig := &magento2.StoreConfig{
		Scheme:    "https",          // or "http" depending on your Magento setup
		HostName:  "magento2.example.com", // Replace with your Magento 2 hostname
		StoreCode: "default",        // Replace with your store code if needed
	}

	// 2. Administrator API Authentication (Integration Token)
	bearerToken := "YOUR_ADMIN_INTEGRATION_TOKEN" // Replace with your Admin Integration Token

	// 3. Create API Client
	apiClient, err := magento2.NewAPIClientFromIntegration(storeConfig, bearerToken)
	if err != nil {
		panic(err)
	}
	logger.Printf("Obtained client: '%+v'", apiClient)

	// 4. Define Product Details
	product := magento2.Product{
		Name:           "Example Product",
		Sku:            "example-sku-123",
		Price:          99.99,
		AttributeSetID: 4, // Default attribute set ID
		TypeID:         "simple",
	}
	productSaveOptions := true // Set to true to save product options

	// 5. Create Product in Magento 2
	mProduct, err := magento2.CreateOrReplaceProduct(&product, productSaveOptions, apiClient)
	if err != nil {
		panic(err)
	}

	logger.Printf("Created product: %+v", mProduct)
}
```

**Key steps in the example:**

1. **Import the library:** `import "github.com/florinel-chis/go-m2rest"`
2. **Set up Logger:**  Optionally set a custom logger for more detailed output using `magento2.SetLogger()`.
3. **Configure `StoreConfig`:** Provide your Magento 2 instance's scheme, hostname, and store code.
4. **Authentication:** Choose the appropriate authentication method and provide credentials. The example uses Administrator API authentication with an Integration token.
5. **Create `APIClient`:** Instantiate the `APIClient` using `NewAPIClientFromIntegration` (or other `NewAPIClientFrom...` functions based on your authentication type).
6. **Define Magento 2 entities:** Create Go structs representing Magento 2 objects like `Product`, `Category`, `Attribute`, etc.
7. **Use library functions:** Call functions from the `magento2` package (e.g., `CreateOrReplaceProduct`, `GetProductBySKU`, `NewGuestCartFromAPIClient`, etc.) to interact with the Magento 2 API.
8. **Handle Errors:** Check for errors returned by the library functions and handle them appropriately.
9. **Verbose Logging:** Observe the detailed logs in your console output to track API requests, responses, and potential issues.

**Authentication Methods**

This library supports the following authentication methods:

* **Administrator API (Integration Token):** Use `NewAPIClientFromIntegration(storeConfig *StoreConfig, bearer string)` with an Admin Integration token.
* **Customer API (Customer Token):** Use `NewAPIClientFromAuthentication(storeConfig *StoreConfig, payload AuthenticationRequestPayload, authenticationType AuthenticationType)` with customer username and password and `magento2.CustomerAuth` as `authenticationType`.
* **Administrator API (Admin Credentials):** Use `NewAPIClientFromAuthentication(storeConfig *StoreConfig, payload AuthenticationRequestPayload, authenticationType AuthenticationType)` with admin username and password and `magento2.Administrator` as `authenticationType`.
* **Guest API (No Authentication):** Use `NewAPIClientWithoutAuthentication(storeConfig *StoreConfig)` for operations that don't require authentication (like guest cart management).

**Examples**

You can find more detailed examples in the `examples/` directory of this repository. These examples cover various use cases like:

* Placing orders as a guest and a customer.
* Creating products, attributes, attribute sets, and categories using the Admin API.
* Managing configurable products.

## Contributing

Contributions are welcome! If you find bugs, have feature requests, or want to contribute code, please open an issue or submit a pull request on GitHub.

Please follow these guidelines when contributing:

1.  **Fork the repository.**
2.  **Create a branch** for your feature or bug fix.
3.  **Write tests** for your code.
4.  **Ensure all tests pass** before submitting a pull request.
5.  **Follow Go coding conventions** and style.
6.  **Document your code** clearly.

## License

This project is licensed under the [MIT License](LICENSE).

## Support

If you encounter any issues or have questions, please open an issue on GitHub at [https://github.com/florinel-chis/go-m2rest/issues](https://github.com/florinel-chis/go-m2rest/issues).

---

**Enjoy building amazing Magento 2 REST API integrations with Go using `go-m2rest`!**
```
