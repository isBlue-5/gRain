package services

import (
	"context"
	"fmt"
	"strings"

	"complete_demo/models"
	"complete_demo/repositories"
)

// ProductService 产品服务
// frame:service
type ProductService struct {
	productRepo repositories.ProductRepository
}

// NewProductService 创建产品服务
func NewProductService(productRepo repositories.ProductRepository) *ProductService {
	return &ProductService{
		productRepo: productRepo,
	}
}

// GetProductByID 根据ID获取产品
// frame:transaction(readOnly=true)
func (s *ProductService) GetProductByID(ctx context.Context, id uint) (*models.Product, error) {
	return s.productRepo.FindByID(ctx, id)
}

// SearchProducts 搜索产品
// frame:transaction(readOnly=true)
func (s *ProductService) SearchProducts(ctx context.Context, req *models.ProductSearchRequest) (*models.ProductSearchResponse, error) {
	// 构建查询条件
	filters := make(map[string]interface{})
	if req.Category != "" {
		filters["category"] = req.Category
	}
	if req.Brand != "" {
		filters["brand"] = req.Brand
	}
	if req.Status != "" {
		filters["status"] = req.Status
	}

	// 构建搜索查询
	var searchQuery string
	if req.Query != "" {
		searchQuery = fmt.Sprintf("name LIKE '%%%s%%' OR description LIKE '%%%s%%'", req.Query, req.Query)
	}

	// 调用仓库层
	products, total, err := s.productRepo.FindByConditions(ctx, filters, searchQuery, "created_at DESC", req.PageSize, (req.Page-1)*req.PageSize)
	if err != nil {
		return nil, fmt.Errorf("搜索产品失败: %w", err)
	}

	// 转换为响应格式
	var productResponses []*models.ProductResponse
	for _, product := range products {
		productResponses = append(productResponses, product.ToResponse())
	}

	// 计算总页数
	totalPage := (total + int64(req.PageSize) - 1) / int64(req.PageSize)

	return &models.ProductSearchResponse{
		Products:  productResponses,
		Total:     total,
		Page:      req.Page,
		PageSize:  req.PageSize,
		TotalPage: int(totalPage),
	}, nil
}

// CreateProduct 创建产品
// frame:transaction
func (s *ProductService) CreateProduct(ctx context.Context, req *models.ProductCreateRequest, createdBy uint) (*models.Product, error) {
	// 检查SKU是否已存在
	existingProduct, err := s.productRepo.FindBySKU(ctx, req.SKU)
	if err == nil && existingProduct != nil {
		return nil, fmt.Errorf("SKU已存在")
	}

	// 创建产品
	product := &models.Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
		Category:    req.Category,
		Brand:       req.Brand,
		SKU:         req.SKU,
		ImageURL:    req.ImageURL,
		Tags:        req.Tags,
		CreatedBy:   createdBy,
	}

	if err := s.productRepo.Create(ctx, product); err != nil {
		return nil, fmt.Errorf("创建产品失败: %w", err)
	}

	return product, nil
}

// UpdateProduct 更新产品
// frame:transaction
func (s *ProductService) UpdateProduct(ctx context.Context, id uint, req *models.ProductUpdateRequest, updatedBy uint) (*models.Product, error) {
	// 获取现有产品
	product, err := s.productRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("产品不存在: %w", err)
	}

	// 更新字段
	if req.Name != "" {
		product.Name = req.Name
	}
	if req.Description != "" {
		product.Description = req.Description
	}
	if req.Price > 0 {
		product.Price = req.Price
	}
	if req.Stock >= 0 {
		product.Stock = req.Stock
	}
	if req.Category != "" {
		product.Category = req.Category
	}
	if req.Brand != "" {
		product.Brand = req.Brand
	}
	if req.ImageURL != "" {
		product.ImageURL = req.ImageURL
	}
	if req.Tags != "" {
		product.Tags = req.Tags
	}
	if req.Status != "" {
		product.Status = req.Status
	}

	product.UpdatedBy = updatedBy

	// 保存更新
	if err := s.productRepo.Update(ctx, product); err != nil {
		return nil, fmt.Errorf("更新产品失败: %w", err)
	}

	return product, nil
}

// DeleteProduct 删除产品
// frame:transaction
func (s *ProductService) DeleteProduct(ctx context.Context, id uint) error {
	// 检查产品是否存在
	_, err := s.productRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("产品不存在: %w", err)
	}

	// 软删除产品
	return s.productRepo.Delete(ctx, id)
}

// UpdateProductStock 更新产品库存
// frame:transaction
func (s *ProductService) UpdateProductStock(ctx context.Context, id uint, stock int) error {
	// 获取产品
	product, err := s.productRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("产品不存在: %w", err)
	}

	// 更新库存
	product.Stock = stock

	// 保存更新
	if err := s.productRepo.Update(ctx, product); err != nil {
		return fmt.Errorf("更新产品库存失败: %w", err)
	}

	return nil
}

// GetProductCategories 获取产品分类
// frame:transaction(readOnly=true)
func (s *ProductService) GetProductCategories(ctx context.Context) ([]string, error) {
	return s.productRepo.GetCategories(ctx)
}

// GetProductBrands 获取产品品牌
// frame:transaction(readOnly=true)
func (s *ProductService) GetProductBrands(ctx context.Context) ([]string, error) {
	return s.productRepo.GetBrands(ctx)
}

// GetProductStats 获取产品统计信息
// frame:transaction(readOnly=true)
func (s *ProductService) GetProductStats(ctx context.Context) (map[string]interface{}, error) {
	// 获取总产品数
	totalProducts, err := s.productRepo.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取产品总数失败: %w", err)
	}

	// 获取活跃产品数
	activeProducts, err := s.productRepo.CountByStatus(ctx, "ACTIVE")
	if err != nil {
		return nil, fmt.Errorf("获取活跃产品数失败: %w", err)
	}

	// 获取库存不足的产品数
	lowStockProducts, err := s.productRepo.CountByStockRange(ctx, 0, 10)
	if err != nil {
		return nil, fmt.Errorf("获取库存不足产品数失败: %w", err)
	}

	// 获取分类统计
	categories, err := s.productRepo.GetCategoryStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取分类统计失败: %w", err)
	}

	// 获取品牌统计
	brands, err := s.productRepo.GetBrandStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取品牌统计失败: %w", err)
	}

	return map[string]interface{}{
		"total_products":     totalProducts,
		"active_products":    activeProducts,
		"inactive_products":  totalProducts - activeProducts,
		"low_stock_products": lowStockProducts,
		"categories":         categories,
		"brands":             brands,
	}, nil
}

// BulkUpdateProducts 批量更新产品
// frame:transaction
func (s *ProductService) BulkUpdateProducts(ctx context.Context, productIDs []uint, updates *models.ProductUpdateRequest, updatedBy uint) (int, error) {
	updatedCount := 0

	for _, id := range productIDs {
		// 获取产品
		product, err := s.productRepo.FindByID(ctx, id)
		if err != nil {
			continue // 跳过不存在的产品
		}

		// 更新字段
		if updates.Name != "" {
			product.Name = updates.Name
		}
		if updates.Description != "" {
			product.Description = updates.Description
		}
		if updates.Price > 0 {
			product.Price = updates.Price
		}
		if updates.Stock >= 0 {
			product.Stock = updates.Stock
		}
		if updates.Category != "" {
			product.Category = updates.Category
		}
		if updates.Brand != "" {
			product.Brand = updates.Brand
		}
		if updates.ImageURL != "" {
			product.ImageURL = updates.ImageURL
		}
		if updates.Tags != "" {
			product.Tags = updates.Tags
		}
		if updates.Status != "" {
			product.Status = updates.Status
		}

		product.UpdatedBy = updatedBy

		// 保存更新
		if err := s.productRepo.Update(ctx, product); err != nil {
			continue // 跳过更新失败的产品
		}

		updatedCount++
	}

	return updatedCount, nil
}

// ImportProducts 批量导入产品
// frame:transaction
func (s *ProductService) ImportProducts(ctx context.Context, products []*models.Product, createdBy uint) error {
	for _, product := range products {
		product.CreatedBy = createdBy
		if err := s.productRepo.Create(ctx, product); err != nil {
			return fmt.Errorf("导入产品失败: %w", err)
		}
	}
	return nil
}

// ExportProducts 导出产品
// frame:transaction(readOnly=true)
func (s *ProductService) ExportProducts(ctx context.Context, filters map[string]interface{}) ([]*models.Product, error) {
	products, _, err := s.productRepo.FindByConditions(ctx, filters, "", "created_at DESC", 10000, 0)
	return products, err
}

// GetProductRecommendations 获取产品推荐
// frame:transaction(readOnly=true)
func (s *ProductService) GetProductRecommendations(ctx context.Context, productID uint, limit int) ([]*models.Product, error) {
	// 获取当前产品
	product, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("产品不存在: %w", err)
	}

	// 基于分类和品牌推荐相似产品
	filters := map[string]interface{}{
		"category": product.Category,
		"status":   "ACTIVE",
	}

	// 排除当前产品
	excludeQuery := fmt.Sprintf("id != %d", productID)

	recommendations, _, err := s.productRepo.FindByConditions(ctx, filters, excludeQuery, "created_at DESC", limit, 0)
	if err != nil {
		return nil, fmt.Errorf("获取产品推荐失败: %w", err)
	}

	return recommendations, nil
}

// SearchProductsByTags 根据标签搜索产品
// frame:transaction(readOnly=true)
func (s *ProductService) SearchProductsByTags(ctx context.Context, tags []string, page, pageSize int) (*models.ProductSearchResponse, error) {
	// 构建标签搜索查询
	var tagQueries []string
	for _, tag := range tags {
		tagQueries = append(tagQueries, fmt.Sprintf("tags LIKE '%%%s%%'", tag))
	}
	searchQuery := strings.Join(tagQueries, " OR ")

	// 调用仓库层
	products, total, err := s.productRepo.FindByConditions(ctx, nil, searchQuery, "created_at DESC", pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, fmt.Errorf("根据标签搜索产品失败: %w", err)
	}

	// 转换为响应格式
	var productResponses []*models.ProductResponse
	for _, product := range products {
		productResponses = append(productResponses, product.ToResponse())
	}

	// 计算总页数
	totalPage := (total + int64(pageSize) - 1) / int64(pageSize)

	return &models.ProductSearchResponse{
		Products:  productResponses,
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
		TotalPage: int(totalPage),
	}, nil
}
