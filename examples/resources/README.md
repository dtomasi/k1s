# k1s Demo Resources

This directory contains example YAML files for testing the k1s-demo CLI application.

## Resource Types

### Categories
- `electronics-category.yaml` - Main electronics category
- `computers-category.yaml` - Subcategory for computers (parent: electronics)

### Items
- `laptop-item.yaml` - Dell XPS 13 laptop in computers category
- `phone-item.yaml` - iPhone 15 Pro in electronics category
- `monitor-item.yaml` - LG UltraWide monitor in computers category

## Usage Examples

```bash
# Apply all resources
k1s-demo apply -f resources/

# Apply specific resource
k1s-demo apply -f resources/electronics-category.yaml

# Create resources
k1s-demo create -f resources/laptop-item.yaml

# Get resources
k1s-demo get categories
k1s-demo get items
k1s-demo get item laptop-123 -o yaml

# Get with different output formats
k1s-demo get categories -o json
k1s-demo get items -o table
k1s-demo get items -o name

# Delete resources
k1s-demo delete item laptop-123
k1s-demo delete category computers
```

## Resource Hierarchy

```
electronics (Category)
├── phone-456 (Item - iPhone 15 Pro)
└── computers (Category - subcategory)
    ├── laptop-123 (Item - Dell XPS 13)
    └── monitor-789 (Item - LG UltraWide)
```

## Labels and Selectors

Resources include labels for testing selector functionality:

```bash
# Filter by category
k1s-demo get items -l category=computers

# Filter by brand
k1s-demo get items -l brand=apple

# Filter by type
k1s-demo get items -l type=laptop
```