# go-m2rest drift report

- Date: 2026-10-08 07:39 UTC
- Store: http://127.0.0.1:8084
- Schema: Magento Community, info.version 2.4, 147 paths
- Magento (vendor composer.json): magento/magento2-base 2.4.8
- Vendor tree: /Users/flo/fch/magento248/vendor
- go-m2rest commit: f5a941c32100-dirty, 69 registered routes
- Token scope: /rest/all/schema lists only the routes the token may call; a route missing from it is DRIFT unless an etc/webapi.xml in the vendor tree declares it (then INFO).

**Summary: 0 DRIFT, 138 INFO**

## DRIFT

None.

## INFO

| Route | Subject | Finding |
|---|---|---|
| `DELETE /V1/carts/mine/items/{itemId}` | `DELETE /V1/carts/mine/items/{itemId}` | not in the schema (hidden by token scope); declared in magento/module-quote/etc/webapi.xml |
| `GET /V1/carts/mine` | `GET /V1/carts/mine` | not in the schema (hidden by token scope); declared in magento/module-quote/etc/webapi.xml |
| `GET /V1/carts/mine/payment-methods` | `GET /V1/carts/mine/payment-methods` | not in the schema (hidden by token scope); declared in magento/module-quote/etc/webapi.xml |
| `GET /V1/carts/search` | `BillingAddress` | 1 schema properties of quote-data-address-interface not in the type: region |
| `GET /V1/carts/search` | `CartListResponse` | 1 schema properties of quote-data-cart-search-results-interface not in the type: search_criteria |
| `GET /V1/categories` | `CategoryTreeNode` | 1 schema properties of catalog-data-category-tree-interface not in the type: position |
| `GET /V1/categories/list` | `categorySearchQueryResponse` | 1 schema properties of catalog-data-category-search-results-interface not in the type: total_count |
| `GET /V1/categories/list` | `categorySearchQueryResponse.search_criteria` | 3 schema properties of framework-search-criteria-interface not in the type: current_page, page_size, sort_orders |
| `GET /V1/creditmemos` | `CreditMemoListResponse` | 1 schema properties of sales-data-creditmemo-search-result-interface not in the type: search_criteria |
| `GET /V1/customers/search` | `CustomerListResponse` | 1 schema properties of customer-data-customer-search-results-interface not in the type: search_criteria |
| `GET /V1/inventory/get-product-salable-quantity/{sku}/{stockId}` | `GET /V1/inventory/get-product-salable-quantity/{sku}/{stockId}` | not in the schema (hidden by token scope); declared in magento/module-inventory-sales-api/etc/webapi.xml |
| `GET /V1/inventory/source-items` | `SourceItemListResponse` | 1 schema properties of inventory-api-data-source-item-search-results-interface not in the type: search_criteria |
| `GET /V1/inventory/sources` | `SourceListResponse` | 1 schema properties of inventory-api-data-source-search-results-interface not in the type: search_criteria |
| `GET /V1/inventory/stocks` | `StockListResponse` | 1 schema properties of inventory-api-data-stock-search-results-interface not in the type: search_criteria |
| `GET /V1/invoices` | `InvoiceListResponse` | 1 schema properties of sales-data-invoice-search-result-interface not in the type: search_criteria |
| `GET /V1/orders` | `Item.extension_attributes` | 1 schema properties of sales-data-order-item-extension-interface not in the type: itemized_taxes |
| `GET /V1/orders` | `Item.extension_attributes.gift_message.extension_attributes.wrapping_add_printed_card` | extension attribute not in gift-message-data-message-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Item.extension_attributes.gift_message.extension_attributes.wrapping_allow_gift_receipt` | extension attribute not in gift-message-data-message-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Item.extension_attributes.gift_message.extension_attributes.wrapping_id` | extension attribute not in gift-message-data-message-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Item.extension_attributes.gw_base_price` | extension attribute not in sales-data-order-item-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Item.extension_attributes.gw_base_price_invoiced` | extension attribute not in sales-data-order-item-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Item.extension_attributes.gw_base_price_refunded` | extension attribute not in sales-data-order-item-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Item.extension_attributes.gw_base_tax_amount` | extension attribute not in sales-data-order-item-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Item.extension_attributes.gw_base_tax_amount_invoiced` | extension attribute not in sales-data-order-item-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Item.extension_attributes.gw_base_tax_amount_refunded` | extension attribute not in sales-data-order-item-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Item.extension_attributes.gw_id` | extension attribute not in sales-data-order-item-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Item.extension_attributes.gw_price` | extension attribute not in sales-data-order-item-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Item.extension_attributes.gw_price_invoiced` | extension attribute not in sales-data-order-item-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Item.extension_attributes.gw_price_refunded` | extension attribute not in sales-data-order-item-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Item.extension_attributes.gw_tax_amount` | extension attribute not in sales-data-order-item-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Item.extension_attributes.gw_tax_amount_invoiced` | extension attribute not in sales-data-order-item-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Item.extension_attributes.gw_tax_amount_refunded` | extension attribute not in sales-data-order-item-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Item.extension_attributes.invoice_text_codes` | extension attribute not in sales-data-order-item-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Item.extension_attributes.tax_codes` | extension attribute not in sales-data-order-item-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Item.extension_attributes.vertex_tax_codes` | extension attribute not in sales-data-order-item-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Item.parent_item` | 95 schema properties of sales-data-order-item-interface not in the type: additional_data, amount_refunded, applied_rule_ids, base_amount_refunded, base_cost, base_discount_amount, base_discount_invoiced, base_discount_refunded, base_discount_tax_compensation_amount, base_discount_tax_compensation_invoiced, base_discount_tax_compensation_refunded, base_original_price, base_price, base_price_incl_tax, base_row_invoiced, base_row_total, base_row_total_incl_tax, base_tax_amount, base_tax_before_discount, base_tax_invoiced, base_tax_refunded, base_weee_tax_applied_amount, base_weee_tax_applied_row_amnt, base_weee_tax_disposition, base_weee_tax_row_disposition, created_at, description, discount_amount, discount_invoiced, discount_percent, discount_refunded, discount_tax_compensation_amount, discount_tax_compensation_canceled, discount_tax_compensation_invoiced, discount_tax_compensation_refunded, event_id, ext_order_item_id, extension_attributes, free_shipping, gw_base_price, gw_base_price_invoiced, gw_base_price_refunded, gw_base_tax_amount, gw_base_tax_amount_invoiced, gw_base_tax_amount_refunded, gw_id, gw_price, gw_price_invoiced, gw_price_refunded, gw_tax_amount, gw_tax_amount_invoiced, gw_tax_amount_refunded, is_qty_decimal, is_virtual, item_id, locked_do_invoice, locked_do_ship, name, no_discount, order_id, original_price, parent_item, parent_item_id, price, price_incl_tax, product_id, product_option, product_type, qty_backordered, qty_canceled, qty_invoiced, qty_ordered, qty_refunded, qty_returned, qty_shipped, quote_item_id, row_invoiced, row_total, row_total_incl_tax, row_weight, sku, store_id, tax_amount, tax_before_discount, tax_canceled, tax_invoiced, tax_percent, tax_refunded, updated_at, weee_tax_applied, weee_tax_applied_amount, weee_tax_applied_row_amount, weee_tax_disposition, weee_tax_row_disposition, weight |
| `GET /V1/orders` | `Order.extension_attributes` | 5 schema properties of sales-data-order-extension-interface not in the type: additional_itemized_taxes, notification_sent, pickup_location_code, send_notification, taxes |
| `GET /V1/orders` | `Order.extension_attributes.amazon_order_reference_id` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.base_customer_balance_amount` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.base_customer_balance_invoiced` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.base_customer_balance_refunded` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.base_customer_balance_total_refunded` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.base_gift_cards_amount` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.base_gift_cards_invoiced` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.base_gift_cards_refunded` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.base_reward_currency_amount` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.company_order_attributes` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.customer_balance_amount` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.customer_balance_invoiced` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.customer_balance_refunded` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.customer_balance_total_refunded` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gift_cards` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gift_cards_amount` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gift_cards_invoiced` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gift_cards_refunded` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gift_message.extension_attributes.wrapping_add_printed_card` | extension attribute not in gift-message-data-message-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gift_message.extension_attributes.wrapping_allow_gift_receipt` | extension attribute not in gift-message-data-message-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gift_message.extension_attributes.wrapping_id` | extension attribute not in gift-message-data-message-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_add_card` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_allow_gift_receipt` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_base_price` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_base_price_incl_tax` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_base_price_invoiced` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_base_price_refunded` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_base_tax_amount` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_base_tax_amount_invoiced` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_base_tax_amount_refunded` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_card_base_price` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_card_base_price_incl_tax` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_card_base_price_invoiced` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_card_base_price_refunded` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_card_base_tax_amount` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_card_base_tax_invoiced` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_card_base_tax_refunded` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_card_price` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_card_price_incl_tax` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_card_price_invoiced` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_card_price_refunded` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_card_tax_amount` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_card_tax_invoiced` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_card_tax_refunded` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_id` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_items_base_price` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_items_base_price_incl_tax` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_items_base_price_invoiced` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_items_base_price_refunded` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_items_base_tax_amount` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_items_base_tax_invoiced` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_items_base_tax_refunded` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_items_price` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_items_price_incl_tax` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_items_price_invoiced` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_items_price_refunded` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_items_tax_amount` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_items_tax_invoiced` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_items_tax_refunded` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_price` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_price_incl_tax` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_price_invoiced` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_price_refunded` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_tax_amount` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_tax_amount_invoiced` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.gw_tax_amount_refunded` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.reward_currency_amount` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.reward_points_balance` | extension attribute not in sales-data-order-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.shipping_assignments.shipping.extension_attributes.collection_point` | extension attribute not in sales-data-shipping-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.shipping_assignments.shipping.extension_attributes.ext_order_id` | extension attribute not in sales-data-shipping-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.extension_attributes.shipping_assignments.shipping.extension_attributes.shipping_experience` | extension attribute not in sales-data-shipping-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/orders` | `Order.payment.extension_attributes` | 1 schema properties of sales-data-order-payment-extension-interface not in the type: notification_message |
| `GET /V1/orders` | `Order.payment.extension_attributes.vault_payment_token` | 1 schema properties of vault-data-payment-token-interface not in the type: website_id |
| `GET /V1/orders` | `OrderListResponse` | 1 schema properties of sales-data-order-search-result-interface not in the type: search_criteria |
| `GET /V1/orders` | `OrdersProductOption.extension_attributes.giftcard_item_option` | extension attribute not in catalog-data-product-option-extension-interface (its module is not installed on this store, or the token cannot see it) |
| `GET /V1/products` | `ProductListResponse` | 1 schema properties of catalog-data-product-search-results-interface not in the type: search_criteria |
| `GET /V1/products/attribute-sets/groups/list` | `GET /V1/products/attribute-sets/groups/list` | not in the schema (hidden by token scope); declared in magento/module-catalog/etc/webapi.xml |
| `GET /V1/products/attribute-sets/sets/list` | `GET /V1/products/attribute-sets/sets/list` | not in the schema (hidden by token scope); declared in magento/module-catalog/etc/webapi.xml |
| `GET /V1/products/attribute-sets/{attributeSetId}` | `GET /V1/products/attribute-sets/{attributeSetId}` | not in the schema (hidden by token scope); declared in magento/module-catalog/etc/webapi.xml |
| `GET /V1/products/attribute-sets/{attributeSetId}/attributes` | `GET /V1/products/attribute-sets/{attributeSetId}/attributes` | not in the schema (hidden by token scope); declared in magento/module-catalog/etc/webapi.xml |
| `GET /V1/products/attributes` | `GET /V1/products/attributes` | not in the schema (hidden by token scope); declared in magento/module-catalog/etc/webapi.xml |
| `GET /V1/products/attributes/{attributeCode}` | `GET /V1/products/attributes/{attributeCode}` | not in the schema (hidden by token scope); declared in magento/module-catalog/etc/webapi.xml |
| `GET /V1/shipments` | `ShipmentListResponse` | 1 schema properties of sales-data-shipment-search-result-interface not in the type: search_criteria |
| `GET /V1/stockItems/lowStock/` | `StockItemListResponse` | 1 schema properties of catalog-inventory-data-stock-item-collection-interface not in the type: search_criteria |
| `GET /V1/store/storeViews` | `StoreView` | 2 schema properties of store-data-store-interface not in the type: extension_attributes, is_active |
| `GET /V1/store/websites` | `Website` | 1 schema properties of store-data-website-interface not in the type: extension_attributes |
| `POST /V1/carts/mine` | `POST /V1/carts/mine` | not in the schema (hidden by token scope); declared in magento/module-quote/etc/webapi.xml |
| `POST /V1/carts/mine/estimate-shipping-methods` | `POST /V1/carts/mine/estimate-shipping-methods` | not in the schema (hidden by token scope); declared in magento/module-quote/etc/webapi.xml |
| `POST /V1/carts/mine/items` | `POST /V1/carts/mine/items` | not in the schema (hidden by token scope); declared in magento/module-quote/etc/webapi.xml |
| `POST /V1/carts/mine/shipping-information` | `POST /V1/carts/mine/shipping-information` | not in the schema (hidden by token scope); declared in magento/module-checkout/etc/webapi.xml |
| `POST /V1/guest-carts/{cartId}/estimate-shipping-methods` | `Carrier` | 1 schema properties of quote-data-shipping-method-interface not in the type: extension_attributes |
| `POST /V1/orders` | `POST /V1/orders` | not in the schema (hidden by token scope); declared in magento/module-sales/etc/webapi.xml |
| `POST /V1/orders/{id}/comments` | `POST /V1/orders/{id}/comments` | not in the schema (hidden by token scope); declared in magento/module-sales/etc/webapi.xml |
| `POST /V1/products/attribute-sets` | `POST /V1/products/attribute-sets` | not in the schema (hidden by token scope); declared in magento/module-catalog/etc/webapi.xml |
| `POST /V1/products/attribute-sets/attributes` | `POST /V1/products/attribute-sets/attributes` | not in the schema (hidden by token scope); declared in magento/module-catalog/etc/webapi.xml |
| `POST /V1/products/attribute-sets/groups` | `POST /V1/products/attribute-sets/groups` | not in the schema (hidden by token scope); declared in magento/module-catalog/etc/webapi.xml |
| `POST /V1/products/attributes` | `POST /V1/products/attributes` | not in the schema (hidden by token scope); declared in magento/module-catalog/etc/webapi.xml |
| `POST /V1/products/attributes/{attributeCode}/options` | `POST /V1/products/attributes/{attributeCode}/options` | not in the schema (hidden by token scope); declared in magento/module-catalog/etc/webapi.xml |
| `PUT /V1/carts/mine/order` | `PUT /V1/carts/mine/order` | not in the schema (hidden by token scope); declared in magento/module-quote/etc/webapi.xml |
| `PUT /V1/products/attribute-sets/{attributeSetId}` | `PUT /V1/products/attribute-sets/{attributeSetId}` | not in the schema (hidden by token scope); declared in magento/module-catalog/etc/webapi.xml |
| `PUT /V1/products/attributes/{attributeCode}` | `PUT /V1/products/attributes/{attributeCode}` | not in the schema (hidden by token scope); declared in magento/module-catalog/etc/webapi.xml |
