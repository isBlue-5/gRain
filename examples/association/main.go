// Package association 提供关联关系示例
package association

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 演示关联关系功能
func RunAssociationDemo() {
	// 设置GORM日志
	gormLogger := logger.New(
		log.New(log.Writer(), "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	// 连接数据库
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		log.Fatalf("无法连接数据库: %v", err)
	}

	// 自动迁移
	err = db.AutoMigrate(&Author{}, &Category{}, &Post{}, &Comment{}, &PostMeta{})
	if err != nil {
		log.Fatalf("自动迁移失败: %v", err)
	}

	// 创建上下文（用于后续扩展）
	_ = context.Background()

	// 创建作者
	authors := []*Author{
		{Name: "张三", Email: "zhangsan@example.com", Bio: "资深技术作家"},
		{Name: "李四", Email: "lisi@example.com", Bio: "科技博主"},
	}

	// 创建分类
	categories := []*Category{
		{Name: "技术", Description: "技术相关文章"},
		{Name: "教程", Description: "教程类文章"},
		{Name: "新闻", Description: "新闻类文章"},
	}

	// 保存作者和分类
	for _, author := range authors {
		if err := db.Create(author).Error; err != nil {
			log.Fatalf("创建作者失败: %v", err)
		}
	}

	for _, category := range categories {
		if err := db.Create(category).Error; err != nil {
			log.Fatalf("创建分类失败: %v", err)
		}
	}

	// 创建文章
	posts := []*Post{
		{
			Title:    "Go语言入门",
			Content:  "Go语言是一种由Google开发的开源编程语言...",
			Status:   "published",
			AuthorID: authors[0].ID,
		},
		{
			Title:    "GORM使用指南",
			Content:  "GORM是Go语言中最流行的ORM库之一...",
			Status:   "published",
			AuthorID: authors[0].ID,
		},
		{
			Title:    "Go 1.18新特性",
			Content:  "Go 1.18引入了泛型支持...",
			Status:   "draft",
			AuthorID: authors[1].ID,
		},
	}

	// 保存文章
	for _, post := range posts {
		if err := db.Create(post).Error; err != nil {
			log.Fatalf("创建文章失败: %v", err)
		}
	}

	// 创建评论
	comments := []*Comment{
		{
			Content:   "非常好的文章！",
			PostID:    posts[0].ID,
			UserName:  "读者A",
			UserEmail: "reader_a@example.com",
		},
		{
			Content:   "学习了很多，谢谢分享！",
			PostID:    posts[0].ID,
			UserName:  "读者B",
			UserEmail: "reader_b@example.com",
		},
		{
			Content:   "这篇教程很实用！",
			PostID:    posts[1].ID,
			UserName:  "读者C",
			UserEmail: "reader_c@example.com",
		},
	}

	// 保存评论
	for _, comment := range comments {
		if err := db.Create(comment).Error; err != nil {
			log.Fatalf("创建评论失败: %v", err)
		}
	}

	// 创建文章元信息
	postMetas := []*PostMeta{
		{
			PostID:          posts[0].ID,
			ViewCount:       100,
			LikeCount:       10,
			FeaturedImage:   "go-intro.jpg",
			MetaTitle:       "Go语言入门教程",
			MetaDescription: "适合初学者的Go语言入门教程",
			MetaKeywords:    "Go,编程,入门",
		},
		{
			PostID:          posts[1].ID,
			ViewCount:       50,
			LikeCount:       5,
			FeaturedImage:   "gorm-guide.jpg",
			MetaTitle:       "GORM完全指南",
			MetaDescription: "详细介绍GORM的使用方法",
			MetaKeywords:    "Go,GORM,ORM,数据库",
		},
	}

	// 保存文章元信息
	for _, meta := range postMetas {
		if err := db.Create(meta).Error; err != nil {
			log.Fatalf("创建文章元信息失败: %v", err)
		}
	}

	// 建立文章与分类的多对多关系
	if err := db.Exec("CREATE TABLE IF NOT EXISTS post_category (post_id INTEGER, category_id INTEGER, PRIMARY KEY (post_id, category_id))").Error; err != nil {
		log.Fatalf("创建中间表失败: %v", err)
	}

	// 文章1属于"技术"和"教程"分类
	if err := db.Exec("INSERT INTO post_category (post_id, category_id) VALUES (?, ?)", posts[0].ID, categories[0].ID).Error; err != nil {
		log.Fatalf("建立关联失败: %v", err)
	}
	if err := db.Exec("INSERT INTO post_category (post_id, category_id) VALUES (?, ?)", posts[0].ID, categories[1].ID).Error; err != nil {
		log.Fatalf("建立关联失败: %v", err)
	}

	// 文章2属于"教程"分类
	if err := db.Exec("INSERT INTO post_category (post_id, category_id) VALUES (?, ?)", posts[1].ID, categories[1].ID).Error; err != nil {
		log.Fatalf("建立关联失败: %v", err)
	}

	// 文章3属于"技术"和"新闻"分类
	if err := db.Exec("INSERT INTO post_category (post_id, category_id) VALUES (?, ?)", posts[2].ID, categories[0].ID).Error; err != nil {
		log.Fatalf("建立关联失败: %v", err)
	}
	if err := db.Exec("INSERT INTO post_category (post_id, category_id) VALUES (?, ?)", posts[2].ID, categories[2].ID).Error; err != nil {
		log.Fatalf("建立关联失败: %v", err)
	}

	fmt.Println("=== 关联关系演示 ===")

	// 1. 一对多关系：查找作者的所有文章
	fmt.Println("\n1. 一对多关系：查找作者的所有文章")
	var author Author
	err = db.First(&author, authors[0].ID).Error
	if err != nil {
		log.Fatalf("查找作者失败: %v", err)
	}

	var authorPosts []*Post
	err = db.Where("author_id = ?", author.ID).Find(&authorPosts).Error
	if err != nil {
		log.Fatalf("查找作者文章失败: %v", err)
	}

	fmt.Printf("作者 '%s' 的文章:\n", author.Name)
	for _, post := range authorPosts {
		fmt.Printf("  - %s\n", post.Title)
	}

	// 2. 多对一关系：查找文章的作者
	fmt.Println("\n2. 多对一关系：查找文章的作者")
	var post Post
	err = db.First(&post, posts[0].ID).Error
	if err != nil {
		log.Fatalf("查找文章失败: %v", err)
	}

	var postAuthor Author
	err = db.First(&postAuthor, post.AuthorID).Error
	if err != nil {
		log.Fatalf("查找文章作者失败: %v", err)
	}

	fmt.Printf("文章 '%s' 的作者: %s\n", post.Title, postAuthor.Name)

	// 3. 一对一关系：查找文章的元信息
	fmt.Println("\n3. 一对一关系：查找文章的元信息")
	var postMeta PostMeta
	err = db.Where("post_id = ?", posts[0].ID).First(&postMeta).Error
	if err != nil {
		log.Fatalf("查找文章元信息失败: %v", err)
	}

	fmt.Printf("文章 '%s' 的元信息:\n", post.Title)
	fmt.Printf("  - 浏览次数: %d\n", postMeta.ViewCount)
	fmt.Printf("  - 点赞次数: %d\n", postMeta.LikeCount)
	fmt.Printf("  - 特色图片: %s\n", postMeta.FeaturedImage)

	// 4. 多对多关系：查找文章的所有分类
	fmt.Println("\n4. 多对多关系：查找文章的所有分类")
	var postCategories []*Category
	err = db.Joins("JOIN post_category ON post_category.category_id = categories.id").
		Where("post_category.post_id = ?", posts[0].ID).
		Find(&postCategories).Error
	if err != nil {
		log.Fatalf("查找文章分类失败: %v", err)
	}

	fmt.Printf("文章 '%s' 的分类:\n", post.Title)
	for _, category := range postCategories {
		fmt.Printf("  - %s\n", category.Name)
	}

	// 5. 多对多关系：查找分类下的所有文章
	fmt.Println("\n5. 多对多关系：查找分类下的所有文章")
	var category Category
	err = db.First(&category, categories[1].ID).Error // "教程"分类
	if err != nil {
		log.Fatalf("查找分类失败: %v", err)
	}

	var categoryPosts []*Post
	err = db.Joins("JOIN post_category ON post_category.post_id = posts.id").
		Where("post_category.category_id = ?", category.ID).
		Find(&categoryPosts).Error
	if err != nil {
		log.Fatalf("查找分类文章失败: %v", err)
	}

	fmt.Printf("分类 '%s' 下的文章:\n", category.Name)
	for _, p := range categoryPosts {
		fmt.Printf("  - %s\n", p.Title)
	}

	// 6. 一对多关系：查找文章的所有评论
	fmt.Println("\n6. 一对多关系：查找文章的所有评论")
	var postComments []*Comment
	err = db.Where("post_id = ?", posts[0].ID).Find(&postComments).Error
	if err != nil {
		log.Fatalf("查找文章评论失败: %v", err)
	}

	fmt.Printf("文章 '%s' 的评论:\n", post.Title)
	for _, comment := range postComments {
		fmt.Printf("  - %s: %s\n", comment.UserName, comment.Content)
	}

	// 7. 预加载演示：一次性加载文章及其关联数据
	fmt.Println("\n7. 预加载演示：一次性加载文章及其关联数据")
	var postWithRelations Post
	err = db.Preload("Author").
		Preload("Comments").
		Joins("LEFT JOIN post_meta ON post_meta.post_id = posts.id").
		Select("posts.*, post_meta.view_count, post_meta.like_count").
		First(&postWithRelations, posts[0].ID).Error
	if err != nil {
		log.Fatalf("查找文章及关联数据失败: %v", err)
	}

	fmt.Printf("文章 '%s' 及其关联数据:\n", postWithRelations.Title)
	fmt.Printf("  - 作者: %s\n", postWithRelations.Author.Name)
	fmt.Printf("  - 评论数: %d\n", len(postWithRelations.Comments))

	fmt.Println("\n=== 关联关系演示结束 ===")
}
