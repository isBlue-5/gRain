// Package entity 演示实体注解的使用示例
package entity

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/grain-framework/grain/pkg/data"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 注意：以下接口和函数会由实体处理器自动生成
// ProductRepository 产品仓库接口
type ProductRepository interface {
	FindByID(ctx context.Context, id uint) (*Product, error)
	FindAll(ctx context.Context) ([]*Product, error)
	Save(ctx context.Context, entity *Product) error
	Update(ctx context.Context, entity *Product) error
	Delete(ctx context.Context, id uint) error
}

// NewProductRepository 创建产品仓库
func NewProductRepository(db data.DBSession) ProductRepository {
	// 在实际使用中，这个函数会由实体处理器自动生成
	return nil
}

// 注意结束

// ProductService 产品服务
// frame:inject
type ProductService struct {
	// 注入产品仓库
	repo ProductRepository `inject:""`
}

// GetProduct 获取产品
// frame:transaction(readOnly=true)
func (s *ProductService) GetProduct(ctx context.Context, id uint) (*Product, error) {
	return s.repo.FindByID(ctx, id)
}

// GetAllProducts 获取所有产品
// frame:transaction(readOnly=true)
func (s *ProductService) GetAllProducts(ctx context.Context) ([]*Product, error) {
	return s.repo.FindAll(ctx)
}

// CreateProduct 创建产品
// frame:transaction
func (s *ProductService) CreateProduct(ctx context.Context, product *Product) error {
	now := time.Now()
	product.CreatedAt = now
	product.UpdatedAt = now
	return s.repo.Save(ctx, product)
}

// UpdateProduct 更新产品
// frame:transaction
func (s *ProductService) UpdateProduct(ctx context.Context, product *Product) error {
	product.UpdatedAt = time.Now()
	return s.repo.Update(ctx, product)
}

// DeleteProduct 删除产品
// frame:transaction
func (s *ProductService) DeleteProduct(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}

// ProductController 产品控制器
// frame:controller(prefix="/api/products")
type ProductController struct {
	// 注入产品服务
	service *ProductService `inject:""`
}

// GetProduct 获取产品
// frame:route(method="GET", path="/:id")
func (c *ProductController) GetProduct(ctx *gin.Context) {
	id := uint(ctx.GetInt("id"))
	product, err := c.service.GetProduct(ctx, id)
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(200, product)
}

// GetAllProducts 获取所有产品
// frame:route(method="GET", path="/")
func (c *ProductController) GetAllProducts(ctx *gin.Context) {
	products, err := c.service.GetAllProducts(ctx)
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(200, products)
}

// CreateProduct 创建产品
// frame:route(method="POST", path="/")
// frame:bind(source="json", model="Product")
func (c *ProductController) CreateProduct(ctx *gin.Context) {
	var product Product
	if err := ctx.ShouldBindJSON(&product); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if err := c.service.CreateProduct(ctx, &product); err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(201, product)
}

// UpdateProduct 更新产品
// frame:route(method="PUT", path="/:id")
// frame:bind(source="json", model="Product")
func (c *ProductController) UpdateProduct(ctx *gin.Context) {
	id := uint(ctx.GetInt("id"))
	var product Product
	if err := ctx.ShouldBindJSON(&product); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	product.ID = id
	if err := c.service.UpdateProduct(ctx, &product); err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(200, product)
}

// DeleteProduct 删除产品
// frame:route(method="DELETE", path="/:id")
func (c *ProductController) DeleteProduct(ctx *gin.Context) {
	id := uint(ctx.GetInt("id"))
	if err := c.service.DeleteProduct(ctx, id); err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(204, nil)
}

// 应用程序入口
func RunExample() {
	// 设置上下文，支持取消信号
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 捕获终止信号
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalCh
		log.Println("接收到终止信号，准备退出...")
		cancel()
	}()

	// 初始化GORM
	db, err := initGorm()
	if err != nil {
		log.Fatalf("初始化GORM失败: %v", err)
	}

	// 自动迁移表结构
	if err := db.AutoMigrate(&Product{}, &Category{}, &Order{}, &OrderItem{}); err != nil {
		log.Fatalf("自动迁移表结构失败: %v", err)
	}

	// 创建GORM会话
	session := data.NewGormSession(db)

	// 创建产品仓库
	productRepo := NewProductRepository(session)

	// 创建产品服务
	productService := &ProductService{
		repo: productRepo,
	}

	// 创建产品控制器
	productController := &ProductController{
		service: productService,
	}

	// 创建Gin引擎
	router := gin.Default()

	// 注册API路由
	productsGroup := router.Group("/api/products")
	{
		productsGroup.GET("/", productController.GetAllProducts)
		productsGroup.GET("/:id", productController.GetProduct)
		productsGroup.POST("/", productController.CreateProduct)
		productsGroup.PUT("/:id", productController.UpdateProduct)
		productsGroup.DELETE("/:id", productController.DeleteProduct)
	}

	// 创建HTTP服务器
	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// 在协程中启动服务器
	go func() {
		log.Println("启动HTTP服务器在 :8080...")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("启动服务器失败: %v", err)
		}
	}()

	// 等待取消信号
	<-ctx.Done()

	// 优雅关闭服务器
	log.Println("正在关闭HTTP服务器...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("服务器关闭失败: %v", err)
	}

	log.Println("服务器已关闭")
}

// 初始化GORM
func initGorm() (*gorm.DB, error) {
	// 配置GORM日志
	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	// 连接数据库
	dsn := "user:password@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, err
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}
