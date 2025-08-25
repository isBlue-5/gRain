// Package transaction 演示事务注解的使用示例
package transaction

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/grain-framework/grain/pkg/data"
)

// 初始化数据库和服务
func setupApp() (*UserService, *sql.DB, error) {
	// 连接数据库
	db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/testdb?parseTime=true")
	if err != nil {
		return nil, nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// 创建数据库会话
	session := data.NewSQLSession(db)

	// 创建用户服务
	userService := &UserService{
		db: session,
	}

	return userService, db, nil
}

// 初始化数据库表结构
func initDatabase(db *sql.DB) error {
	// 创建用户表
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INT PRIMARY KEY AUTO_INCREMENT,
			username VARCHAR(255) NOT NULL UNIQUE,
			email VARCHAR(255) NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("创建用户表失败: %w", err)
	}

	// 创建用户积分表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_credits (
			user_id INT PRIMARY KEY,
			credit_balance DECIMAL(10,2) NOT NULL DEFAULT 0.0,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)
	if err != nil {
		return fmt.Errorf("创建用户积分表失败: %w", err)
	}

	return nil
}

// 演示事务处理
func demoTransaction(ctx context.Context, service *UserService) error {
	// 创建用户演示
	user1, err := service.CreateUser(ctx, &User{
		Username: "user1",
		Email:    "user1@example.com",
	})
	if err != nil {
		return fmt.Errorf("创建用户1失败: %w", err)
	}
	log.Printf("创建用户1成功: ID=%d, 用户名=%s", user1.ID, user1.Username)

	user2, err := service.CreateUser(ctx, &User{
		Username: "user2",
		Email:    "user2@example.com",
	})
	if err != nil {
		return fmt.Errorf("创建用户2失败: %w", err)
	}
	log.Printf("创建用户2成功: ID=%d, 用户名=%s", user2.ID, user2.Username)

	// 初始化用户积分（通过SQL直接操作，实际应用中应该使用服务方法）
	session := service.db.(*data.SQLSession)
	tx, err := session.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}

	_, err = tx.Exec(ctx, "INSERT INTO user_credits (user_id, credit_balance) VALUES (?, ?)", user1.ID, 100.0)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("初始化用户1积分失败: %w", err)
	}

	_, err = tx.Exec(ctx, "INSERT INTO user_credits (user_id, credit_balance) VALUES (?, ?)", user2.ID, 50.0)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("初始化用户2积分失败: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	log.Printf("初始化用户积分成功")

	// 演示转移积分
	err = service.TransferCredits(ctx, user1.ID, user2.ID, 30.0)
	if err != nil {
		return fmt.Errorf("转移积分失败: %w", err)
	}
	log.Printf("成功从用户1转移30积分到用户2")

	// 演示只读事务
	user1Info, err := service.ReadOnlyMethod(ctx, user1.ID)
	if err != nil {
		return fmt.Errorf("读取用户1信息失败: %w", err)
	}
	log.Printf("用户1信息: ID=%d, 用户名=%s", user1Info.ID, user1Info.Username)

	// 事务传播行为演示
	log.Println("\n--- 事务传播行为演示 ---")
	if err := service.InnerRequiresNew(ctx, user1.ID); err != nil {
		log.Printf("InnerRequiresNew error: %v", err)
	}
	if err := service.InnerSupports(ctx, user1.ID); err != nil {
		log.Printf("InnerSupports error: %v", err)
	}
	if err := service.InnerNotSupported(ctx, user1.ID); err != nil {
		log.Printf("InnerNotSupported error: %v", err)
	}
	if err := service.InnerMandatory(ctx, user1.ID); err != nil {
		log.Printf("InnerMandatory error: %v", err)
	}
	if err := service.InnerNested(ctx, user1.ID); err != nil {
		log.Printf("InnerNested error: %v", err)
	}

	// 事务模板方法（带钩子和日志）演示
	log.Println("\n--- 事务模板方法（带钩子和日志）演示 ---")
	_, _ = data.TransactionTemplateV2(
		ctx,
		service.db,
		func(txCtx context.Context) (interface{}, error) {
			log.Println("[业务] 在事务中执行业务逻辑（可插入/更新/查询等）")
			return nil, nil
		},
		func(ctx context.Context, tx data.Transaction) { log.Printf("[钩子] Before Commit: TxID=%d", tx.ID()) },
		func(ctx context.Context, tx data.Transaction) {
			log.Printf("[钩子] After Commit: TxID=%d, 状态=%s", tx.ID(), tx.Status())
		},
		func(ctx context.Context, tx data.Transaction) {
			log.Printf("[钩子] Before Rollback: TxID=%d", tx.ID())
		},
		func(ctx context.Context, tx data.Transaction) {
			log.Printf("[钩子] After Rollback: TxID=%d, 状态=%s", tx.ID(), tx.Status())
		},
		func(format string, args ...interface{}) { log.Printf(format, args...) },
	)

	return nil
}

// 主函数
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

	// 初始化应用
	service, db, err := setupApp()
	if err != nil {
		log.Fatalf("初始化应用失败: %v", err)
	}
	defer db.Close()

	// 初始化数据库
	if err := initDatabase(db); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 运行演示
	if err := demoTransaction(ctx, service); err != nil {
		log.Fatalf("事务演示失败: %v", err)
	}

	log.Println("演示完成!")
}
