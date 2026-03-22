package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// 解析配置
	config, err := pgxpool.ParseConfig("postgres://gg:123456@localhost:5432/ggyynet")
	if err != nil {
		log.Fatal(err)
	}

	// 配置连接池参数
	config.MaxConns = 20                      // 最大连接数
	config.MinConns = 4                       // 最小保持连接数
	config.MaxConnLifetime = 30 * time.Minute // 连接最大存活时间
	config.MaxConnIdleTime = 30 * time.Second // 空闲连接超时

	// 创建连接池
	ctx := context.Background()
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	// 插入
	err = insertUser(ctx, pool, "张三", "zhangsan@example.com", 25)
	if err != nil {
		log.Fatal("insertUser:", err)
	}

	id, err := insertUserReturnID(ctx, pool, "李四", "lisi@example.com", 30)
	if err != nil {
		log.Fatal("insertUserReturnID:", err)
	}
	fmt.Printf("insertUserReturnID 插入成功，ID: %d\n", id)

}

// CREATE TABLE users (
// id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
// name VARCHAR(100) NOT NULL,
// email VARCHAR(100) UNIQUE NOT NULL,
// age INT,
// created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
// );
// Exec 执行不返回行的命令
func insertUser(ctx context.Context, pool *pgxpool.Pool, name, email string, age int) error {
	sql := `INSERT INTO users (name, email, age) VALUES ($1, $2, $3)`

	_, err := pool.Exec(ctx, sql, name, email, age)
	return err
}
func insertUserReturnID(ctx context.Context, pool *pgxpool.Pool, name, email string, age int) (int32, error) {
	sql := `INSERT INTO users (name, email, age) VALUES ($1, $2, $3) RETURNING id`

	var id int32
	err := pool.QueryRow(ctx, sql, name, email, age).Scan(&id)
	return id, err
}

func batchInsertUsers(ctx context.Context, pool *pgxpool.Pool, users []User) error {
	// 开启事务
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	// 确保回滚或提交
	defer tx.Rollback(ctx)

	sql := `INSERT INTO users (name, email, age) VALUES ($1, $2, $3)`

	for _, user := range users {
		_, err := tx.Exec(ctx, sql, user.Name, user.Email, user.Age)
		if err != nil {
			return err
		}
	}

	// 提交事务
	return tx.Commit(ctx)
}

// 根据 ID 删除
func deleteUserByID(ctx context.Context, pool *pgxpool.Pool, id int32) error {
	sql := `DELETE FROM users WHERE id = $1`

	result, err := pool.Exec(ctx, sql, id)
	if err != nil {
		return err
	}

	// 检查实际删除的行数
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("未找到 ID 为 %d 的用户", id)
	}

	fmt.Printf("成功删除 %d 行\n", rowsAffected)
	return nil
}

// 条件删除
func deleteUsersByAge(ctx context.Context, pool *pgxpool.Pool, maxAge int) error {
	sql := `DELETE FROM users WHERE age > $1`
	_, err := pool.Exec(ctx, sql, maxAge)
	return err
}

// 更新用户信息
func updateUser(ctx context.Context, pool *pgxpool.Pool, id int32, name string, age int) error {
	sql := `UPDATE users SET name = $1, age = $2 WHERE id = $3`

	result, err := pool.Exec(ctx, sql, name, age, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("未找到要更新的用户")
	}
	return nil
}

// 部分字段更新（动态构建 SQL）
func updateUserPartial(ctx context.Context, pool *pgxpool.Pool, id int32, updates map[string]interface{}) error {
	// 使用 pgx 的 NamedArgs 或手动构建
	// 这里展示参数化查询方式
	sql := `UPDATE users SET name = COALESCE($1, name), 
                            email = COALESCE($2, email), 
                            age = COALESCE($3, age) 
            WHERE id = $4`

	_, err := pool.Exec(ctx, sql, updates["name"], updates["email"], updates["age"], id)
	return err
}

type User struct {
	ID        int64
	Name      string
	Email     string
	Age       int32
	CreatedAt time.Time
}

// 根据 ID 查询
func getUserByID(ctx context.Context, pool *pgxpool.Pool, id int32) (*User, error) {
	sql := `SELECT id, name, email, age, created_at FROM users WHERE id = $1`

	var user User
	err := pool.QueryRow(ctx, sql, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Age,
		&user.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, err
	}
	return &user, nil
}

// 查询所有用户
func getAllUsers(ctx context.Context, pool *pgxpool.Pool) ([]User, error) {
	sql := `SELECT id, name, email, age, created_at FROM users ORDER BY id`

	rows, err := pool.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close() // 必须关闭

	var users []User
	for rows.Next() {
		var u User
		err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Age, &u.CreatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	// 检查迭代错误
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}
func searchUsers(ctx context.Context, pool *pgxpool.Pool, keyword string, page, pageSize int) ([]User, int, error) {
	// 计算总数
	countSQL := `SELECT COUNT(*) FROM users WHERE name ILIKE $1`
	var total int
	err := pool.QueryRow(ctx, countSQL, "%"+keyword+"%").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 分页查询
	querySQL := `SELECT id, name, email, age, created_at 
                 FROM users 
                 WHERE name ILIKE $1 
                 ORDER BY id 
                 LIMIT $2 OFFSET $3`

	offset := (page - 1) * pageSize
	rows, err := pool.Query(ctx, querySQL, "%"+keyword+"%", pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		rows.Scan(&u.ID, &u.Name, &u.Email, &u.Age, &u.CreatedAt)
		users = append(users, u)
	}

	return users, total, rows.Err()
}
func transfer(ctx context.Context, pool *pgxpool.Pool, fromID, toID, amount int32) error {
	// 使用 BeginTx 可以设置事务选项（隔离级别等）
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, // 隔离级别
	})
	if err != nil {
		return err
	}
	// 确保回滚（如果未提交）
	defer tx.Rollback(ctx)

	// 扣款
	_, err = tx.Exec(ctx, `UPDATE accounts SET balance = balance - $1 WHERE id = $2`, amount, fromID)
	if err != nil {
		return err
	}

	// 入账
	_, err = tx.Exec(ctx, `UPDATE accounts SET balance = balance + $1 WHERE id = $2`, amount, toID)
	if err != nil {
		return err
	}

	// 提交
	return tx.Commit(ctx)
}

// 一行代码查询并映射到结构体
func getUsersSimple(ctx context.Context, pool *pgxpool.Pool) ([]User, error) {
	sql := `SELECT id, name, email, age, created_at FROM users`

	// pgx.CollectRows 自动处理迭代和关闭
	query, _ := pool.Query(ctx, sql)
	return pgx.CollectRows(
		query,
		pgx.RowToStructByPos[User], // 按位置映射到结构体
	)
}

// 或者使用 RowToStructByName 按字段名映射
func getUsersByName(ctx context.Context, pool *pgxpool.Pool) ([]User, error) {
	sql := `SELECT * FROM users`
	rows, _ := pool.Query(ctx, sql)
	return pgx.CollectRows(rows, pgx.RowToStructByName[User])
}
