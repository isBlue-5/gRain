// Package association 提供关联关系示例
package association

import (
	"time"
)

// Author 作者实体
// frame:entity(table="authors")
type Author struct {
	ID        uint      `db:"id,primaryKey" json:"id"`
	Name      string    `db:"name" json:"name" validate:"required"`
	Email     string    `db:"email" json:"email" validate:"required,email"`
	Bio       string    `db:"bio" json:"bio"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`

	// 一对多关系：作者拥有多篇文章
	// frame:oneToMany(targetEntity="Post")
	Posts []*Post `json:"posts,omitempty"`
}

// Category 分类实体
// frame:entity(table="categories")
type Category struct {
	ID          uint      `db:"id,primaryKey" json:"id"`
	Name        string    `db:"name" json:"name" validate:"required"`
	Description string    `db:"description" json:"description"`
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time `db:"updated_at" json:"updatedAt"`

	// 多对多关系：分类包含多篇文章
	// frame:manyToMany(targetEntity="Post")
	Posts []*Post `json:"posts,omitempty"`
}

// Post 文章实体
// frame:entity(table="posts")
type Post struct {
	ID        uint      `db:"id,primaryKey" json:"id"`
	Title     string    `db:"title" json:"title" validate:"required"`
	Content   string    `db:"content" json:"content" validate:"required"`
	Status    string    `db:"status" json:"status" validate:"required"`
	AuthorID  uint      `db:"author_id" json:"authorId"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`

	// 多对一关系：文章属于一个作者
	// frame:manyToOne(targetEntity="Author")
	Author *Author `json:"author,omitempty"`

	// 多对多关系：文章属于多个分类
	// frame:manyToMany(targetEntity="Category")
	Categories []*Category `json:"categories,omitempty"`

	// 一对多关系：文章拥有多条评论
	// frame:oneToMany(targetEntity="Comment")
	Comments []*Comment `json:"comments,omitempty"`

	// 一对一关系：文章拥有一个元信息
	// frame:oneToOne(targetEntity="PostMeta")
	Meta *PostMeta `json:"meta,omitempty"`
}

// Comment 评论实体
// frame:entity(table="comments")
type Comment struct {
	ID        uint      `db:"id,primaryKey" json:"id"`
	Content   string    `db:"content" json:"content" validate:"required"`
	PostID    uint      `db:"post_id" json:"postId"`
	UserName  string    `db:"user_name" json:"userName" validate:"required"`
	UserEmail string    `db:"user_email" json:"userEmail" validate:"required,email"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`

	// 多对一关系：评论属于一篇文章
	// frame:manyToOne(targetEntity="Post")
	Post *Post `json:"post,omitempty"`
}

// PostMeta 文章元信息实体
// frame:entity(table="post_meta")
type PostMeta struct {
	ID              uint      `db:"id,primaryKey" json:"id"`
	PostID          uint      `db:"post_id" json:"postId"`
	ViewCount       int       `db:"view_count" json:"viewCount"`
	LikeCount       int       `db:"like_count" json:"likeCount"`
	FeaturedImage   string    `db:"featured_image" json:"featuredImage"`
	MetaTitle       string    `db:"meta_title" json:"metaTitle"`
	MetaDescription string    `db:"meta_description" json:"metaDescription"`
	MetaKeywords    string    `db:"meta_keywords" json:"metaKeywords"`
	CreatedAt       time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt       time.Time `db:"updated_at" json:"updatedAt"`

	// 一对一关系：元信息属于一篇文章
	// frame:oneToOne(targetEntity="Post")
	Post *Post `json:"post,omitempty"`
}
