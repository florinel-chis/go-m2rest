package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"time"

	magento2 "github.com/florinel-chis/go-m2rest"
)

type StockUpdate struct {
	SKU string
	Qty float64
}

func main() {
	// Command line flags
	var (
		csvFile      = flag.String("csv", "stock_updates.csv", "CSV file with SKU and quantity")
		createOnly   = flag.Bool("create-only", false, "Only create products, don't update stock")
		updateOnly   = flag.Bool("update-only", false, "Only update stock, don't create products")
		concurrent   = flag.Int("concurrent", 5, "Number of concurrent operations")
		productCount = flag.Int("count", 100, "Number of products to create")
	)
	flag.Parse()

	// Setup logging
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Load configuration
	config, err := loadConfig()
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Create Magento client
	storeConfig := &magento2.StoreConfig{
		Scheme:    config.Scheme,
		HostName:  config.Host,
		StoreCode: config.StoreCode,
	}

	client, err := magento2.NewAPIClientFromIntegration(storeConfig, config.BearerToken)
	if err != nil {
		logger.Error("Failed to create API client", "error", err)
		os.Exit(1)
	}

	// Create products if requested
	if !*updateOnly {
		logger.Info("Creating simple products", "count", *productCount)
		createdSKUs := createBulkProducts(client, *productCount, *concurrent, logger)

		// Save created SKUs to CSV if we're only creating
		if *createOnly {
			if err := saveCreatedSKUsToCSV(*csvFile, createdSKUs); err != nil {
				logger.Error("Failed to save SKUs to CSV", "error", err)
			}
			logger.Info("Created SKUs saved to CSV", "file", *csvFile)
			return
		}
	}

	// Update stock from CSV
	if !*createOnly {
		logger.Info("Loading stock updates from CSV", "file", *csvFile)
		updates, err := loadStockUpdatesFromCSV(*csvFile)
		if err != nil {
			logger.Error("Failed to load stock updates", "error", err)
			os.Exit(1)
		}

		logger.Info("Updating product stock", "count", len(updates))
		updateBulkStock(client, updates, *concurrent, logger)
	}

	logger.Info("Bulk operations completed")
}

func loadConfig() (*Config, error) {
	// Try to load from environment variables
	host := os.Getenv("MAGENTO_HOST")
	if host == "" {
		return nil, fmt.Errorf("MAGENTO_HOST environment variable is required")
	}

	bearerToken := os.Getenv("MAGENTO_BEARER_TOKEN")
	if bearerToken == "" {
		return nil, fmt.Errorf("MAGENTO_BEARER_TOKEN environment variable is required")
	}

	storeCode := os.Getenv("MAGENTO_STORE_CODE")
	if storeCode == "" {
		storeCode = "all"
	}

	// Parse URL to get scheme and host
	scheme := "https"
	if len(host) > 7 && host[:7] == "http://" {
		scheme = "http"
		host = host[7:]
	} else if len(host) > 8 && host[:8] == "https://" {
		host = host[8:]
	}

	return &Config{
		Scheme:      scheme,
		Host:        host,
		StoreCode:   storeCode,
		BearerToken: bearerToken,
	}, nil
}

type Config struct {
	Scheme      string
	Host        string
	StoreCode   string
	BearerToken string
}

func createBulkProducts(client *magento2.Client, count int, concurrent int, logger *slog.Logger) []string {
	timestamp := time.Now().Unix()
	products := make([]magento2.Product, count)
	skus := make([]string, count)

	// Generate products
	for i := 0; i < count; i++ {
		sku := fmt.Sprintf("bulk-product-%d-%d", timestamp, i+1)
		skus[i] = sku
		products[i] = magento2.Product{
			Sku:            sku,
			Name:           fmt.Sprintf("Bulk Product %d", i+1),
			AttributeSetID: 4,                    // Default attribute set
			Price:          9.99 + float64(i%10), // Vary price slightly
			TypeID:         "simple",
			Status:         1, // Enabled
			Visibility:     4, // Catalog, Search
			Weight:         1.0,
		}
	}

	// Create products concurrently
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrent)
	errors := make(chan error, count)

	for i, product := range products {
		wg.Add(1)
		go func(idx int, p magento2.Product) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			mProduct, err := magento2.CreateOrReplaceProduct(&p, true, client)
			if err != nil {
				logger.Error("Failed to create product", "error", err, "sku", p.Sku)
				errors <- err
				return
			}

			logger.Info("Product created", "sku", mProduct.Product.Sku, "id", mProduct.Product.ID, "progress", idx+1, "total", count)
		}(i, product)
	}

	wg.Wait()
	close(errors)

	// Count errors
	errorCount := 0
	for range errors {
		errorCount++
	}

	logger.Info("Product creation completed", "created", count-errorCount, "failed", errorCount)

	return skus
}

func updateBulkStock(client *magento2.Client, updates []StockUpdate, concurrent int, logger *slog.Logger) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrent)
	errors := make(chan error, len(updates))

	for i, update := range updates {
		wg.Add(1)
		go func(idx int, u StockUpdate) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// First, get the product to find stock item ID
			product, err := magento2.GetProductBySKU(u.SKU, client)
			if err != nil {
				logger.Error("Failed to get product", "error", err, "sku", u.SKU)
				errors <- err
				return
			}

			// Extract stock item ID
			if product.Product.ExtensionAttributes != nil {
				if stockData, ok := product.Product.ExtensionAttributes["stock_item"]; ok {
					if stockMap, ok := stockData.(map[string]any); ok {
						if itemID, ok := stockMap["item_id"]; ok {
							stockItemID := fmt.Sprintf("%v", itemID)
							err = product.UpdateQuantityForStockItem(stockItemID, int(u.Qty), true)
							if err != nil {
								logger.Error("Failed to update stock", "error", err, "sku", u.SKU)
								errors <- err
								return
							}

							logger.Info("Stock updated", "sku", u.SKU, "qty", u.Qty, "progress", idx+1, "total", len(updates))
							return
						}
					}
				}
			}

			logger.Error("Could not find stock item ID", "sku", u.SKU)
			errors <- fmt.Errorf("no stock item ID for SKU %s", u.SKU)
		}(i, update)
	}

	wg.Wait()
	close(errors)

	// Count errors
	errorCount := 0
	for range errors {
		errorCount++
	}

	logger.Info("Stock update completed", "updated", len(updates)-errorCount, "failed", errorCount)
}

func loadStockUpdatesFromCSV(filename string) ([]StockUpdate, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	var updates []StockUpdate
	for i, record := range records {
		// Skip header if present
		if i == 0 && (record[0] == "sku" || record[0] == "SKU") {
			continue
		}

		if len(record) < 2 {
			log.Printf("Skipping invalid row %d: %v", i+1, record)
			continue
		}

		qty, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			log.Printf("Skipping row %d: invalid quantity %s", i+1, record[1])
			continue
		}

		updates = append(updates, StockUpdate{
			SKU: record[0],
			Qty: qty,
		})
	}

	return updates, nil
}

func saveCreatedSKUsToCSV(filename string, skus []string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"sku", "qty"}); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write SKUs with random quantities
	for _, sku := range skus {
		qty := 10 + (time.Now().UnixNano() % 90) // Random qty between 10-100
		if err := writer.Write([]string{sku, fmt.Sprintf("%d", qty)}); err != nil {
			return fmt.Errorf("failed to write row: %w", err)
		}
	}

	return nil
}
