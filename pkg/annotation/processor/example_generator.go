package processor

import (
	"fmt"
	"os"
	"path/filepath"
)

// ExampleGenerator 示例应用生成器
type ExampleGenerator struct {
	generator *Generator
	examples  map[string]ExampleTemplate
}

// ExampleTemplate 示例模板
type ExampleTemplate struct {
	Name         string
	Description  string
	Files        map[string]string
	Dependencies []string
}

// NewExampleGenerator 创建新的示例生成器
func NewExampleGenerator(generator *Generator) *ExampleGenerator {
	eg := &ExampleGenerator{
		generator: generator,
		examples:  make(map[string]ExampleTemplate),
	}

	// 初始化示例模板
	eg.initializeTemplates()

	return eg
}

// initializeTemplates 初始化示例模板
func (eg *ExampleGenerator) initializeTemplates() {
	eg.examples["basic"] = ExampleTemplate{
		Name:        "Basic Example",
		Description: "A simple Hello World application",
		Files: map[string]string{
			"main.go": `package main

import (
	"github.com/gin-gonic/gin"
	"github.com/isBlue-5/grain/pkg/web"
)

func main() {
	r := gin.Default()
	web.RegisterRoutes(r)
	r.Run(":8080")
}`,
			"controller.go": `package main

import "github.com/gin-gonic/gin"

// @frame:controller
type HelloController struct{}

// @frame:route
func (c *HelloController) Hello(ctx *gin.Context) {
	ctx.JSON(200, gin.H{"message": "Hello, World!"})
}`,
		},
		Dependencies: []string{"github.com/gin-gonic/gin"},
	}

	eg.examples["user_management"] = ExampleTemplate{
		Name:        "User Management System",
		Description: "A complete user management system with CRUD operations",
		Files: map[string]string{
			"main.go": `package main

import (
	"github.com/gin-gonic/gin"
	"github.com/isBlue-5/grain/pkg/web"
)

func main() {
	r := gin.Default()
	web.RegisterRoutes(r)
	r.Run(":8080")
}`,
			"models/user.go": `package models

// @frame:entity
type User struct {
	ID       uint   ` + "`json:\"id\"`" + `
	Name     string ` + "`json:\"name\"`" + `
	Email    string ` + "`json:\"email\"`" + `
	Password string ` + "`json:\"-\"`" + `
}`,
			"controllers/user_controller.go": `package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/isBlue-5/grain/examples/user_management/services"
)

// @frame:controller
type UserController struct {
	userService *services.UserService
}

// @frame:route(method="GET", path="/users")
func (c *UserController) ListUsers(ctx *gin.Context) {
	users := c.userService.GetAllUsers()
	ctx.JSON(200, users)
}

// @frame:route(method="POST", path="/users")
func (c *UserController) CreateUser(ctx *gin.Context) {
	var user models.User
	if err := ctx.ShouldBindJSON(&user); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	createdUser := c.userService.CreateUser(&user)
	ctx.JSON(201, createdUser)
}`,
			"services/user_service.go": `package services

import (
	"github.com/isBlue-5/grain/examples/user_management/models"
	"github.com/isBlue-5/grain/examples/user_management/repositories"
)

// @frame:service
type UserService struct {
	userRepo *repositories.UserRepository
}

// @frame:method
func (s *UserService) GetAllUsers() []models.User {
	return s.userRepo.FindAll()
}

// @frame:method
func (s *UserService) CreateUser(user *models.User) *models.User {
	return s.userRepo.Save(user)
}`,
			"repositories/user_repository.go": `package repositories

import (
	"github.com/isBlue-5/grain/examples/user_management/models"
)

// @frame:repository
type UserRepository struct {
	// 这里可以注入数据库连接
}

// @frame:method
func (r *UserRepository) FindAll() []models.User {
	// 实现数据库查询逻辑
	return []models.User{}
}

// @frame:method
func (r *UserRepository) Save(user *models.User) *models.User {
	// 实现数据库保存逻辑
	return user
}`,
		},
		Dependencies: []string{
			"github.com/gin-gonic/gin",
			"github.com/isBlue-5/grain/pkg/web",
		},
	}
}

// GenerateExample 生成示例应用
func (eg *ExampleGenerator) GenerateExample(name string) error {
	template, exists := eg.examples[name]
	if !exists {
		return fmt.Errorf("example template %s not found", name)
	}

	// 创建示例目录
	exampleDir := filepath.Join(eg.generator.outputDir, "examples", name)
	if err := os.MkdirAll(exampleDir, 0755); err != nil {
		return fmt.Errorf("failed to create example directory: %w", err)
	}

	// 生成示例应用代码
	for fileName, content := range template.Files {
		filePath := filepath.Join(exampleDir, fileName)

		// 确保目录存在
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return fmt.Errorf("failed to create file directory: %w", err)
		}

		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", fileName, err)
		}
	}

	// 生成go.mod文件
	goModPath := filepath.Join(exampleDir, "go.mod")
	goModContent := fmt.Sprintf(`module %s

go 1.21

require (
`, name)

	for _, dep := range template.Dependencies {
		goModContent += fmt.Sprintf("\t%s v0.0.0\n", dep)
	}

	goModContent += ")"

	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		return fmt.Errorf("failed to write go.mod: %w", err)
	}

	// 生成README文件
	readmePath := filepath.Join(exampleDir, "README.md")
	readmeContent := fmt.Sprintf(`# %s

%s

## Getting Started

1. Navigate to this directory
2. Run go mod tidy to download dependencies
3. Run go run main.go to start the application

## Features

This example demonstrates:
- Basic gRain framework usage
- Controller and route annotations
- Service layer implementation
- Repository pattern
`, template.Name, template.Description)

	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		return fmt.Errorf("failed to write README: %w", err)
	}

	return nil
}

// ListExamples 列出所有可用的示例
func (eg *ExampleGenerator) ListExamples() []string {
	var names []string
	for name := range eg.examples {
		names = append(names, name)
	}
	return names
}

// GetExampleInfo 获取示例信息
func (eg *ExampleGenerator) GetExampleInfo(name string) (*ExampleTemplate, error) {
	template, exists := eg.examples[name]
	if !exists {
		return nil, fmt.Errorf("example template %s not found", name)
	}
	return &template, nil
}
