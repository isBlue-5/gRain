// Package entity 演示实体注解的使用示例
package entity

import (
	"time"
)

// Product 产品实体
// frame:entity(table="products")
type Product struct {
	ID          uint      `db:"id,primaryKey" json:"id"`
	Name        string    `db:"name,required" json:"name" validate:"required,min=3,max=100"`
	Description string    `db:"description" json:"description" validate:"max=500"`
	Price       float64   `db:"price,required" json:"price" validate:"required,min=0.01"`
	Stock       int       `db:"stock" json:"stock" validate:"min=0"`
	CategoryID  uint      `db:"category_id" json:"categoryId" validate:"required"`
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time `db:"updated_at" json:"updatedAt"`
}

// Category 产品分类
// frame:entity(table="categories")
type Category struct {
	ID        uint      `db:"id,primaryKey" json:"id"`
	Name      string    `db:"name,required,unique" json:"name" validate:"required,min=2,max=50"`
	ParentID  *uint     `db:"parent_id" json:"parentId"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

// Order 订单
// frame:entity(table="orders")
type Order struct {
	ID         uint      `db:"id,primaryKey" json:"id"`
	UserID     uint      `db:"user_id,required" json:"userId" validate:"required"`
	TotalPrice float64   `db:"total_price,required" json:"totalPrice" validate:"required,min=0.01"`
	Status     string    `db:"status,required" json:"status" validate:"required,oneof=pending paid shipped delivered cancelled"`
	CreatedAt  time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt  time.Time `db:"updated_at" json:"updatedAt"`
}

// OrderItem 订单项
// frame:entity(table="order_items")
type OrderItem struct {
	ID        uint      `db:"id,primaryKey" json:"id"`
	OrderID   uint      `db:"order_id,required" json:"orderId" validate:"required"`
	ProductID uint      `db:"product_id,required" json:"productId" validate:"required"`
	Quantity  int       `db:"quantity,required" json:"quantity" validate:"required,min=1"`
	Price     float64   `db:"price,required" json:"price" validate:"required,min=0.01"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}
