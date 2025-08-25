package repositories

import (
	"context"
	"fmt"
	"time"

	"complete_demo/models"

	"gorm.io/gorm"
)

// ProductRepository 产品仓库接口
type ProductRepository interface {
	Create(ctx context.Context, product *models.Product) error
	Update(ctx context.Context, product *models.Product) error
	Delete(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*models.Product, error)
	FindBySKU(ctx context.Context, sku string) (*models.Product, error)
	FindAll(ctx context.Context) ([]*models.Product, error)
	FindByConditions(ctx context.Context, filters map[string]interface{}, search string, orderBy string, limit, offset int) ([]*models.Product, int64, error)
	Count(ctx context.Context) (int64, error)
	CountByStatus(ctx context.Context, status string) (int64, error)
	CountByStockRange(ctx context.Context, min, max int) (int64, error)
	GetCategories(ctx context.Context) ([]string, error)
	GetBrands(ctx context.Context) ([]string, error)
	GetCategoryStats(ctx context.Context) (map[string]int64, error)
	GetBrandStats(ctx context.Context) (map[string]int64, error)
}

// DefaultProductRepository 默认产品仓库实现
// frame:repository
type DefaultProductRepository struct {
	db *gorm.DB
}

// NewProductRepository 创建产品仓库
func NewProductRepository(db *gorm.DB) ProductRepository {
	return &DefaultProductRepository{db: db}
}

// Create 创建产品
// frame:transaction
func (r *DefaultProductRepository) Create(ctx context.Context, product *models.Product) error {
	return r.db.WithContext(ctx).Create(product).Error
}

// Update 更新产品
// frame:transaction
func (r *DefaultProductRepository) Update(ctx context.Context, product *models.Product) error {
	return r.db.WithContext(ctx).Save(product).Error
}

// Delete 删除产品
// frame:transaction
func (r *DefaultProductRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Product{}, id).Error
}

// FindByID 根据ID查找产品
// frame:transaction(readOnly=true)
func (r *DefaultProductRepository) FindByID(ctx context.Context, id uint) (*models.Product, error) {
	var product models.Product
	err := r.db.WithContext(ctx).First(&product, id).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// FindBySKU 根据SKU查找产品
// frame:transaction(readOnly=true)
func (r *DefaultProductRepository) FindBySKU(ctx context.Context, sku string) (*models.Product, error) {
	var product models.Product
	err := r.db.WithContext(ctx).Where("sku = ?", sku).First(&product).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// FindAll 查找所有产品
// frame:transaction(readOnly=true)
func (r *DefaultProductRepository) FindAll(ctx context.Context) ([]*models.Product, error) {
	var products []*models.Product
	err := r.db.WithContext(ctx).Find(&products).Error
	return products, err
}

// FindByConditions 根据条件查找产品
// frame:transaction(readOnly=true)
func (r *DefaultProductRepository) FindByConditions(ctx context.Context, filters map[string]interface{}, search string, orderBy string, limit, offset int) ([]*models.Product, int64, error) {
	var products []*models.Product
	var total int64

	// 构建查询
	query := r.db.WithContext(ctx).Model(&models.Product{})

	// 应用过滤器
	for key, value := range filters {
		if value != "" {
			query = query.Where(key+" = ?", value)
		}
	}

	// 应用搜索
	if search != "" {
		query = query.Where(search)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 应用排序和分页
	if orderBy != "" {
		query = query.Order(orderBy)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	// 执行查询
	err := query.Find(&products).Error
	return products, total, err
}

// Count 统计产品总数
// frame:transaction(readOnly=true)
func (r *DefaultProductRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Product{}).Count(&count).Error
	return count, err
}

// CountByStatus 根据状态统计产品数
// frame:transaction(readOnly=true)
func (r *DefaultProductRepository) CountByStatus(ctx context.Context, status string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Product{}).Where("status = ?", status).Count(&count).Error
	return count, err
}

// CountByStockRange 根据库存范围统计产品数
// frame:transaction(readOnly=true)
func (r *DefaultProductRepository) CountByStockRange(ctx context.Context, min, max int) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Product{}).Where("stock BETWEEN ? AND ?", min, max).Count(&count).Error
	return count, err
}

// GetCategories 获取所有产品分类
// frame:transaction(readOnly=true)
func (r *DefaultProductRepository) GetCategories(ctx context.Context) ([]string, error) {
	var categories []string
	err := r.db.WithContext(ctx).Model(&models.Product{}).Distinct("category").Pluck("category", &categories).Error
	return categories, err
}

// GetBrands 获取所有产品品牌
// frame:transaction(readOnly=true)
func (r *DefaultProductRepository) GetBrands(ctx context.Context) ([]string, error) {
	var brands []string
	err := r.db.WithContext(ctx).Model(&models.Product{}).Distinct("brand").Pluck("brand", &brands).Error
	return brands, err
}

// GetCategoryStats 获取分类统计
// frame:transaction(readOnly=true)
func (r *DefaultProductRepository) GetCategoryStats(ctx context.Context) (map[string]int64, error) {
	var stats []struct {
		Category string `json:"category"`
		Count    int64  `json:"count"`
	}

	err := r.db.WithContext(ctx).Model(&models.Product{}).
		Select("category, count(*) as count").
		Group("category").
		Scan(&stats).Error

	if err != nil {
		return nil, err
	}

	result := make(map[string]int64)
	for _, stat := range stats {
		result[stat.Category] = stat.Count
	}

	return result, nil
}

// GetBrandStats 获取品牌统计
// frame:transaction(readOnly=true)
func (r *DefaultProductRepository) GetBrandStats(ctx context.Context) (map[string]int64, error) {
	var stats []struct {
		Brand string `json:"brand"`
		Count int64  `json:"count"`
	}

	err := r.db.WithContext(ctx).Model(&models.Product{}).
		Select("brand, count(*) as count").
		Group("brand").
		Scan(&stats).Error

	if err != nil {
		return nil, err
	}

	result := make(map[string]int64)
	for _, stat := range stats {
		result[stat.Brand] = stat.Count
	}

	return result, nil
}

// FindByCategory 根据分类查找产品
// frame:transaction(readOnly=true)
func (r *DefaultProductRepository) FindByCategory(ctx context.Context, category string, limit int) ([]*models.Product, error) {
	var products []*models.Product
	err := r.db.WithContext(ctx).Where("category = ? AND status = ?", category, "ACTIVE").Limit(limit).Find(&products).Error
	return products, err
}

// FindByBrand 根据品牌查找产品
// frame:transaction(readOnly=true)
func (r *DefaultProductRepository) FindByBrand(ctx context.Context, brand string, limit int) ([]*models.Product, error) {
	var products []*models.Product
	err := r.db.WithContext(ctx).Where("brand = ? AND status = ?", brand, "ACTIVE").Limit(limit).Find(&products).Error
	return products, err
}

// FindByPriceRange 根据价格范围查找产品
// frame:transaction(readOnly=true)
func (r *DefaultProductRepository) FindByPriceRange(ctx context.Context, minPrice, maxPrice float64, limit int) ([]*models.Product, error) {
	var products []*models.Product
	err := r.db.WithContext(ctx).Where("price BETWEEN ? AND ? AND status = ?", minPrice, maxPrice, "ACTIVE").Limit(limit).Find(&products).Error
	return products, err
}

// FindLowStockProducts 查找库存不足的产品
// frame:transaction(readOnly=true)
func (r *DefaultProductRepository) FindLowStockProducts(ctx context.Context, threshold int) ([]*models.Product, error) {
	var products []*models.Product
	err := r.db.WithContext(ctx).Where("stock <= ? AND status = ?", threshold, "ACTIVE").Find(&products).Error
	return products, err
}

// UpdateStock 更新产品库存
// frame:transaction
func (r *DefaultProductRepository) UpdateStock(ctx context.Context, id uint, stock int) error {
	return r.db.WithContext(ctx).Model(&models.Product{}).Where("id = ?", id).Update("stock", stock).Error
}

// BulkUpdateStock 批量更新库存
// frame:transaction
func (r *DefaultProductRepository) BulkUpdateStock(ctx context.Context, updates map[uint]int) error {
	for id, stock := range updates {
		if err := r.UpdateStock(ctx, id, stock); err != nil {
			return fmt.Errorf("更新产品 %d 库存失败: %w", id, err)
		}
	}
	return nil
}

// GetProductStats 获取产品统计信息
// frame:transaction(readOnly=true)
func (r *DefaultProductRepository) GetProductStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 总产品数
	var totalCount int64
	if err := r.db.WithContext(ctx).Model(&models.Product{}).Count(&totalCount).Error; err != nil {
		return nil, err
	}
	stats["total_products"] = totalCount

	// 按状态统计
	var statusStats []struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	if err := r.db.WithContext(ctx).Model(&models.Product{}).Select("status, count(*) as count").Group("status").Scan(&statusStats).Error; err != nil {
		return nil, err
	}
	stats["status_stats"] = statusStats

	// 按分类统计
	categoryStats, err := r.GetCategoryStats(ctx)
	if err != nil {
		return nil, err
	}
	stats["category_stats"] = categoryStats

	// 按品牌统计
	brandStats, err := r.GetBrandStats(ctx)
	if err != nil {
		return nil, err
	}
	stats["brand_stats"] = brandStats

	// 库存统计
	var lowStockCount int64
	if err := r.db.WithContext(ctx).Model(&models.Product{}).Where("stock <= 10").Count(&lowStockCount).Error; err != nil {
		return nil, err
	}
	stats["low_stock_count"] = lowStockCount

	// 今日新增产品数
	today := time.Now().Truncate(24 * time.Hour)
	var todayCount int64
	if err := r.db.WithContext(ctx).Model(&models.Product{}).Where("created_at >= ?", today).Count(&todayCount).Error; err != nil {
		return nil, err
	}
	stats["today_new_products"] = todayCount

	return stats, nil
}
