package main

// 导入必要的包
import (
	// fmt包：用于字符串格式化，如构建DSN连接字符串
	"fmt"
	// log包：用于日志记录，如启动信息、错误信息等
	"log"
	// time包：用于时间相关操作，如计算请求处理时长
	"time"

	// gin包：Gin是Go语言的Web框架，用于处理HTTP请求和路由
	"github.com/gin-gonic/gin"
	// redis/v8包：Redis客户端库，用于与Redis服务器交互
	"github.com/go-redis/redis/v8"
	// gorm包：ORM框架，用于数据库操作（GORM v2）
	"gorm.io/gorm"
	// mysql驱动：GORM的MySQL驱动，用于连接MySQL数据库（GORM v2）
	"gorm.io/driver/mysql"
	// gorm logger：GORM的日志包，用于配置日志级别
	"gorm.io/gorm/logger"
	// yaml.v2包：用于解析YAML配置文件
	"gopkg.in/yaml.v2"
	// os包：用于文件操作，如打开配置文件
	"os"
	// context包：用于上下文管理，如Redis操作的上下文
	"context"
)

// Config结构体：用于映射config.yaml配置文件
// 每个字段都有yaml标签，指定配置文件中的对应键名
// 结构体嵌套对应配置文件的层级结构
type Config struct {
	// Server子结构体：服务器配置
	Server struct {
		// Port字段：服务器端口，映射到配置文件中的server.port
		Port string `yaml:"port"`
		// Mode字段：Gin运行模式，映射到配置文件中的server.mode
		Mode string `yaml:"mode"`
	} `yaml:"server"` // 映射到配置文件中的server节点

	// Mysql子结构体：MySQL数据库配置
	Mysql struct {
		// Host字段：MySQL主机地址，映射到配置文件中的mysql.host
		Host string `yaml:"host"`
		// Port字段：MySQL端口，映射到配置文件中的mysql.port
		Port string `yaml:"port"`
		// User字段：MySQL用户名，映射到配置文件中的mysql.user
		User string `yaml:"user"`
		// Password字段：MySQL密码，映射到配置文件中的mysql.password
		Password string `yaml:"password"`
		// Dbname字段：数据库名，映射到配置文件中的mysql.dbname
		Dbname string `yaml:"dbname"`
		// Charset字段：字符集，映射到配置文件中的mysql.charset
		Charset string `yaml:"charset"`
		// ParseTime字段：是否解析时间，映射到配置文件中的mysql.parseTime
		ParseTime bool `yaml:"parseTime"`
		// Loc字段：时区，映射到配置文件中的mysql.loc
		Loc string `yaml:"loc"`
	} `yaml:"mysql"` // 映射到配置文件中的mysql节点

	// Redis子结构体：Redis配置
	Redis struct {
		// Host字段：Redis主机地址，映射到配置文件中的redis.host
		Host string `yaml:"host"`
		// Port字段：Redis端口，映射到配置文件中的redis.port
		Port string `yaml:"port"`
		// Password字段：Redis密码，映射到配置文件中的redis.password
		Password string `yaml:"password"`
		// Db字段：Redis数据库索引，映射到配置文件中的redis.db
		Db int `yaml:"db"`
	} `yaml:"redis"` // 映射到配置文件中的redis节点
}

// 全局变量定义
var (
	// config变量：存储配置信息，全局变量便于在各个函数中访问
	config Config
	// db变量：GORM数据库连接实例，全局变量便于在各个函数中使用
	db *gorm.DB
	// rdb变量：Redis客户端实例，全局变量便于在各个函数中使用
	rdb *redis.Client
)

// loadConfig函数：加载配置文件
// 输入：无
// 输出：error - 加载过程中的错误
func loadConfig() error {
	// 打开config.yaml文件
	// os.Open：打开文件，返回文件句柄和可能的错误
	file, err := os.Open("config.yaml")
	if err != nil {
		// 如果打开文件失败，返回错误
		return err
	}
	// defer file.Close()：延迟关闭文件，确保文件最终会被关闭
	defer file.Close()

	// 创建YAML解码器
	// yaml.NewDecoder：创建一个从文件读取并解析YAML的解码器
	d := yaml.NewDecoder(file)
	// 解码YAML到config结构体
	// d.Decode：将YAML数据解码到指定的结构体
	if err := d.Decode(&config); err != nil {
		// 如果解码失败，返回错误
		return err
	}

	// 加载成功，返回nil
	return nil
}

// initDB函数：初始化数据库连接
// 输入：无
// 输出：error - 初始化过程中的错误
func initDB() error {
	// 构建MySQL连接DSN（Data Source Name）
	// fmt.Sprintf：格式化字符串，将配置参数插入到DSN模板中
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=%t&loc=%s",
		config.Mysql.User,      // 用户名
		config.Mysql.Password,  // 密码
		config.Mysql.Host,      // 主机地址
		config.Mysql.Port,      // 端口
		config.Mysql.Dbname,    // 数据库名
		config.Mysql.Charset,   // 字符集
		config.Mysql.ParseTime, // 是否解析时间
		config.Mysql.Loc,       // 时区
	)

	// 声明错误变量
	var err error
	// 连接MySQL数据库（GORM v2）
	// gorm.Open：打开数据库连接，第一个参数是数据库驱动，第二个参数是配置
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		// 启用日志模式
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		// 如果连接失败，返回错误
		return err
	}

	// 初始化成功，返回nil
	return nil
}

// initRedis函数：初始化Redis连接
// 输入：无
// 输出：error - 初始化过程中的错误
func initRedis() error {
	// 创建Redis客户端
	// redis.NewClient：创建一个新的Redis客户端，参数是Redis选项
	rdb = redis.NewClient(&redis.Options{
		// Addr：Redis服务器地址，格式为"host:port"
		Addr: fmt.Sprintf("%s:%s", config.Redis.Host, config.Redis.Port),
		// Password：Redis密码
		Password: config.Redis.Password,
		// DB：Redis数据库索引
		DB: config.Redis.Db,
	})

	// 创建上下文
	// context.Background：创建一个空上下文，用于Redis操作
	ctx := context.Background()
	// 测试Redis连接
	// rdb.Ping：发送PING命令测试连接，Result()返回结果和错误
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		// 如果连接失败，返回错误
		return err
	}

	// 初始化成功，返回nil
	return nil
}

// main函数：程序入口
func main() {
	// 加载配置文件
	// 首先加载配置，因为后续的数据库和Redis连接都依赖配置信息
	if err := loadConfig(); err != nil {
		// 如果加载配置失败，记录错误并退出程序
		log.Fatalf("加载配置失败: %v", err)
	}

	// 设置Gin模式
	// gin.SetMode：设置Gin的运行模式，如debug、release等
	gin.SetMode(config.Server.Mode)

	// 初始化数据库
	// 数据库初始化依赖配置信息，所以在加载配置后执行
	if err := initDB(); err != nil {
		// 如果初始化数据库失败，记录错误并退出程序
		log.Fatalf("初始化数据库失败: %v", err)
	}
	// 延迟关闭数据库连接
	// 在GORM v2中，需要先获取底层的SQL DB实例，然后调用其Close方法
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("获取数据库实例失败: %v", err)
	}
	defer sqlDB.Close()

	// 初始化Redis
	// Redis初始化也依赖配置信息，所以在加载配置后执行
	if err := initRedis(); err != nil {
		// 如果初始化Redis失败，记录错误并退出程序
		log.Fatalf("初始化Redis失败: %v", err)
	}
	// 延迟关闭Redis连接
	// defer rdb.Close()：在main函数结束时关闭Redis连接
	defer rdb.Close()

	// 创建Gin引擎
	// gin.Default()：创建一个默认的Gin引擎，包含Logger和Recovery中间件
	r := gin.Default()

	// 添加日志中间件
	// r.Use：添加中间件到Gin引擎
	r.Use(func(c *gin.Context) {
		// 开始时间：记录请求开始处理的时间
		start := time.Now()

		// 处理请求：调用下一个中间件或路由处理函数
		c.Next()

		// 结束时间：记录请求处理完成的时间
		timestamp := time.Now()
		// 执行时间：计算请求处理的耗时
		latency := timestamp.Sub(start)
		// 请求方法：获取HTTP请求方法（GET/POST等）
		method := c.Request.Method
		// 请求路径：获取HTTP请求路径
		path := c.Request.URL.Path
		// 状态码：获取HTTP响应状态码
		statusCode := c.Writer.Status()
		// 客户端IP：获取客户端的IP地址
		clientIP := c.ClientIP()

		// 日志格式：打印请求信息
		log.Printf("%s | %3d | %13v | %15s | %s",
			method,     // 请求方法
			statusCode, // 状态码
			latency,    // 处理时长
			clientIP,   // 客户端IP
			path,       // 请求路径
		)
	})

	// 测试路由
	// r.GET：注册GET请求路由，路径为"/ping"
	r.GET("/ping", func(c *gin.Context) {
		// c.JSON：返回JSON响应，第一个参数是状态码，第二个参数是响应数据
		c.JSON(200, gin.H{
			"message": "pong", // 响应数据
		})
	})

	// 启动服务器
	// 构建服务器地址，格式为":port"
	addr := fmt.Sprintf(":%s", config.Server.Port)
	// 记录服务器启动信息
	log.Printf("服务器启动在 %s", addr)
	// 启动服务器
	// r.Run：启动HTTP服务器，监听指定地址
	if err := r.Run(addr); err != nil {
		// 如果启动服务器失败，记录错误并退出程序
		log.Fatalf("启动服务器失败: %v", err)
	}
}
