package controllers

import (
	"net/http"
	"strconv"

	"complete_demo/models"
	"complete_demo/services"

	"github.com/gin-gonic/gin"
)

// ProductController 产品控制器
// frame:controller(path="/api/products")
type ProductController struct {
	productService *services.ProductService `inject:""`
}

// NewProductController 创建产品控制器
func NewProductController(productService *services.ProductService) *ProductController {
	return &ProductController{
		productService: productService,
	}
}

// GetProducts 获取产品列表
// frame:route(method="GET", path="/")
// frame:summary(获取产品列表)
// frame:response(200, []models.Product, "成功获取产品列表")
// frame:response(500, gin.H, "服务器内部错误")
func (c *ProductController) GetProducts(ctx *gin.Context) {
	// 获取分页参数
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))

	// 获取查询参数
	category := ctx.Query("category")
	brand := ctx.Query("brand")
	minPrice := ctx.Query("min_price")
	maxPrice := ctx.Query("max_price")
	status := ctx.Query("status")
	search := ctx.Query("search")

	// 构建搜索请求
	searchReq := &models.ProductSearchRequest{
		Query:    search,
		Category: category,
		Brand:    brand,
		Status:   status,
		Page:     page,
		PageSize: pageSize,
	}

	// 处理价格范围
	if minPrice != "" {
		if price, err := strconv.ParseFloat(minPrice, 64); err == nil {
			searchReq.MinPrice = price
		}
	}
	if maxPrice != "" {
		if price, err := strconv.ParseFloat(maxPrice, 64); err == nil {
			searchReq.MaxPrice = price
		}
	}

	result, err := c.productService.SearchProducts(ctx, searchReq)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取产品列表失败",
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetProduct 获取单个产品
// frame:route(method="GET", path="/:id")
// frame:summary(获取单个产品)
// frame:response(200, models.Product, "成功获取产品信息")
// frame:response(404, gin.H, "产品不存在")
// frame:response(500, gin.H, "服务器内部错误")
func (c *ProductController) GetProduct(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的产品ID",
		})
		return
	}

	product, err := c.productService.GetProductByID(ctx, uint(id))
	if err != nil {
		if err.Error() == "产品不存在" {
			ctx.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "产品不存在",
			})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "获取产品失败",
				"error":   err.Error(),
			})
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    product,
	})
}

// CreateProduct 创建产品
// frame:route(method="POST", path="/")
// frame:summary(创建新产品)
// frame:bind(source="json", model="models.ProductCreateRequest")
// frame:validate(groups="create")
// frame:response(201, models.Product, "产品创建成功")
// frame:response(400, gin.H, "请求参数错误")
// frame:response(409, gin.H, "SKU已存在")
// frame:response(500, gin.H, "服务器内部错误")
func (c *ProductController) CreateProduct(ctx *gin.Context) {
	var req models.ProductCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	// 从上下文获取当前用户ID（这里简化处理，实际应该从JWT token获取）
	userID := uint(1) // 临时硬编码，实际应该从认证中间件获取

	createdProduct, err := c.productService.CreateProduct(ctx, &req, userID)
	if err != nil {
		if err.Error() == "SKU已存在" {
			ctx.JSON(http.StatusConflict, gin.H{
				"success": false,
				"message": "创建失败",
				"error":   err.Error(),
			})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "创建产品失败",
				"error":   err.Error(),
			})
		}
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    createdProduct,
	})
}

// UpdateProduct 更新产品
// frame:route(method="PUT", path="/:id")
// frame:summary(更新产品信息)
// frame:bind(source="json", model="models.ProductUpdateRequest")
// frame:validate(groups="update")
// frame:response(200, models.Product, "产品更新成功")
// frame:response(400, gin.H, "请求参数错误")
// frame:response(404, gin.H, "产品不存在")
// frame:response(500, gin.H, "服务器内部错误")
func (c *ProductController) UpdateProduct(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的产品ID",
		})
		return
	}

	var req models.ProductUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	// 从上下文获取当前用户ID（这里简化处理，实际应该从JWT token获取）
	userID := uint(1) // 临时硬编码，实际应该从认证中间件获取

	updatedProduct, err := c.productService.UpdateProduct(ctx, uint(id), &req, userID)
	if err != nil {
		if err.Error() == "产品不存在" {
			ctx.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "产品不存在",
			})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "更新产品失败",
				"error":   err.Error(),
			})
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    updatedProduct,
	})
}

// DeleteProduct 删除产品
// frame:route(method="DELETE", path="/:id")
// frame:summary(删除产品)
// frame:response(200, gin.H, "产品删除成功")
// frame:response(400, gin.H, "无效的产品ID")
// frame:response(404, gin.H, "产品不存在")
// frame:response(500, gin.H, "服务器内部错误")
func (c *ProductController) DeleteProduct(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的产品ID",
		})
		return
	}

	err = c.productService.DeleteProduct(ctx, uint(id))
	if err != nil {
		if err.Error() == "产品不存在" {
			ctx.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "产品不存在",
			})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "删除产品失败",
				"error":   err.Error(),
			})
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"message": "产品删除成功"},
	})
}

// GetProductCategories 获取产品分类
// frame:route(method="GET", path="/categories")
// frame:summary(获取产品分类列表)
// frame:response(200, []string, "成功获取产品分类")
// frame:response(500, gin.H, "服务器内部错误")
func (c *ProductController) GetProductCategories(ctx *gin.Context) {
	categories, err := c.productService.GetProductCategories(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取产品分类失败",
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    categories,
	})
}

// GetProductBrands 获取产品品牌
// frame:route(method="GET", path="/brands")
// frame:summary(获取产品品牌列表)
// frame:response(200, []string, "成功获取产品品牌")
// frame:response(500, gin.H, "服务器内部错误")
func (c *ProductController) GetProductBrands(ctx *gin.Context) {
	brands, err := c.productService.GetProductBrands(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取产品品牌失败",
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    brands,
	})
}

// GetProductStats 获取产品统计信息
// frame:route(method="GET", path="/stats")
// frame:summary(获取产品统计信息)
// frame:response(200, gin.H, "成功获取产品统计")
// frame:response(500, gin.H, "服务器内部错误")
func (c *ProductController) GetProductStats(ctx *gin.Context) {
	stats, err := c.productService.GetProductStats(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取产品统计失败",
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}
