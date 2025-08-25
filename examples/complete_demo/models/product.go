package models

import (
	"time"

	"gorm.io/gorm"
)

// Product 产品模型
// frame:entity(table="products")
type Product struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"not null;size:100"`
	Description string         `json:"description" gorm:"type:text"`
	Price       float64        `json:"price" gorm:"not null;type:decimal(10,2)"`
	Stock       int            `json:"stock" gorm:"not null;default:0"`
	Category    string         `json:"category" gorm:"size:50;index"`
	Brand       string         `json:"brand" gorm:"size:50"`
	SKU         string         `json:"sku" gorm:"uniqueIndex;size:50"`
	ImageURL    string         `json:"image_url" gorm:"size:255"`
	Tags        string         `json:"tags" gorm:"size:255"`
	Status      string         `json:"status" gorm:"default:'ACTIVE';size:20"`
	CreatedBy   uint           `json:"created_by" gorm:"index"`
	UpdatedBy   uint           `json:"updated_by"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 指定表名
func (Product) TableName() string {
	return "products"
}

// BeforeCreate GORM 钩子：创建前处理
func (p *Product) BeforeCreate(tx *gorm.DB) error {
	if p.Status == "" {
		p.Status = "ACTIVE"
	}
	if p.Stock < 0 {
		p.Stock = 0
	}
	return nil
}

// IsActive 检查是否激活
func (p *Product) IsActive() bool {
	return p.Status == "ACTIVE"
}

// IsInStock 检查是否有库存
func (p *Product) IsInStock() bool {
	return p.Stock > 0
}

// GetFormattedPrice 获取格式化价格
func (p *Product) GetFormattedPrice() string {
	return "$" + string(rune(int(p.Price*100)/100)) + "." + string(rune(int(p.Price*100)%100))
}

// ProductCreateRequest 创建产品请求
type ProductCreateRequest struct {
	Name        string  `json:"name" binding:"required,min=1,max=100"`
	Description string  `json:"description" binding:"max=1000"`
	Price       float64 `json:"price" binding:"required,min=0"`
	Stock       int     `json:"stock" binding:"min=0"`
	Category    string  `json:"category" binding:"required,max=50"`
	Brand       string  `json:"brand" binding:"max=50"`
	SKU         string  `json:"sku" binding:"required,max=50"`
	ImageURL    string  `json:"image_url" binding:"max=255"`
	Tags        string  `json:"tags" binding:"max=255"`
}

// ProductUpdateRequest 更新产品请求
type ProductUpdateRequest struct {
	Name        string  `json:"name" binding:"min=1,max=100"`
	Description string  `json:"description" binding:"max=1000"`
	Price       float64 `json:"price" binding:"min=0"`
	Stock       int     `json:"stock" binding:"min=0"`
	Category    string  `json:"category" binding:"max=50"`
	Brand       string  `json:"brand" binding:"max=50"`
	ImageURL    string  `json:"image_url" binding:"max=255"`
	Tags        string  `json:"tags" binding:"max=255"`
	Status      string  `json:"status" binding:"oneof=ACTIVE INACTIVE DISCONTINUED"`
}

// ProductResponse 产品响应
type ProductResponse struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Stock       int       `json:"stock"`
	Category    string    `json:"category"`
	Brand       string    `json:"brand"`
	SKU         string    `json:"sku"`
	ImageURL    string    `json:"image_url"`
	Tags        string    `json:"tags"`
	Status      string    `json:"status"`
	CreatedBy   uint      `json:"created_by"`
	UpdatedBy   uint      `json:"updated_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ToResponse 转换为响应格式
func (p *Product) ToResponse() *ProductResponse {
	return &ProductResponse{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Price:       p.Price,
		Stock:       p.Stock,
		Category:    p.Category,
		Brand:       p.Brand,
		SKU:         p.SKU,
		ImageURL:    p.ImageURL,
		Tags:        p.Tags,
		Status:      p.Status,
		CreatedBy:   p.CreatedBy,
		UpdatedBy:   p.UpdatedBy,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

// ProductSearchRequest 产品搜索请求
type ProductSearchRequest struct {
	Query    string  `json:"query" binding:"max=100"`
	Category string  `json:"category" binding:"max=50"`
	Brand    string  `json:"brand" binding:"max=50"`
	MinPrice float64 `json:"min_price" binding:"min=0"`
	MaxPrice float64 `json:"max_price" binding:"min=0"`
	Status   string  `json:"status" binding:"oneof=ACTIVE INACTIVE DISCONTINUED"`
	Page     int     `json:"page" binding:"min=1"`
	PageSize int     `json:"page_size" binding:"min=1,max=100"`
}

// ProductSearchResponse 产品搜索响应
type ProductSearchResponse struct {
	Products  []*ProductResponse `json:"products"`
	Total     int64              `json:"total"`
	Page      int                `json:"page"`
	PageSize  int                `json:"page_size"`
	TotalPage int                `json:"total_page"`
}
