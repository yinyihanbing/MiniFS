package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// 基础存储目录
	baseDir = "./storage"
	// 文件存储目录
	fileDir = baseDir + "/files"
	// 字符串存储目录
	stringDir = baseDir + "/strings"
)

// 添加缓存相关的结构和变量
type CacheItem struct {
	Data      interface{}
	Timestamp time.Time
}

var (
	cache    sync.Map
	cacheTTL = 10 * time.Minute
)

// 添加缓存操作的辅助函数
func getCacheKey(prefix, key string) string {
	return prefix + ":" + key
}

func setCache(prefix, key string, data interface{}) {
	cacheKey := getCacheKey(prefix, key)
	cache.Store(cacheKey, CacheItem{
		Data:      data,
		Timestamp: time.Now(),
	})
}

func getCache(prefix, key string) (interface{}, bool) {
	cacheKey := getCacheKey(prefix, key)
	if item, ok := cache.Load(cacheKey); ok {
		cacheItem := item.(CacheItem)
		if time.Since(cacheItem.Timestamp) < cacheTTL {
			return cacheItem.Data, true
		}
		// 缓存过期，删除它
		cache.Delete(cacheKey)
	}
	return nil, false
}

// 初始化存储目录
func initStorageDirs() error {
	dirs := []string{fileDir, stringDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建目录失败 %s: %v", dir, err)
		}
	}
	return nil
}

func main() {
	// 初始化存储目录
	if err := initStorageDirs(); err != nil {
		panic(err)
	}

	r := gin.Default()

	// 添加CORS中间件
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// 上传文件接口
	r.POST("/store/:key", func(c *gin.Context) {
		key := c.Param("key")
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "没有找到上传的文件",
			})
			return
		}

		// 获取文件扩展名
		ext := filepath.Ext(file.Filename)
		filename := filepath.Join(fileDir, key+ext)

		if err := c.SaveUploadedFile(file, filename); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "保存文件失败",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":  fmt.Sprintf("文件已保存，key: %s", key),
			"filename": file.Filename,
		})
	})

	// 获取文件接口
	r.GET("/get/:key", func(c *gin.Context) {
		key := c.Param("key")

		// 查找目录下对应key的文件
		files, err := filepath.Glob(filepath.Join(fileDir, key+".*"))
		if err != nil || len(files) == 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "文件不存在",
			})
			return
		}

		// 返回找到的第一个文件
		c.File(files[0])
	})

	// 存储字符串接口
	r.POST("/string/:key", func(c *gin.Context) {
		key := c.Param("key")
		var data struct {
			Value string `json:"value"`
		}

		if err := c.BindJSON(&data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "无效的JSON数据",
			})
			return
		}

		filename := filepath.Join(stringDir, key+".txt")
		if err := os.WriteFile(filename, []byte(data.Value), 0644); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "保存字符串失败",
			})
			return
		}

		cache.Delete(getCacheKey("string", key)) // 清除旧缓存

		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("字符串已保存，key: %s", key),
		})
	})

	// 获取字符串接口
	r.GET("/string/:key", func(c *gin.Context) {
		key := c.Param("key")

		// 先检查缓存
		if data, found := getCache("string", key); found {
			c.JSON(http.StatusOK, gin.H{
				"value": data.(string),
			})
			return
		}

		filename := filepath.Join(stringDir, key+".txt")
		data, err := os.ReadFile(filename)
		if err != nil {
			if os.IsNotExist(err) {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "字符串不存在",
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "读取字符串失败",
			})
			return
		}

		// 将字符串存入缓存
		setCache("string", key, string(data))

		c.JSON(http.StatusOK, gin.H{
			"value": string(data),
		})
	})

	// 添加检查key是否存在的接口
	r.GET("/exists/:key", func(c *gin.Context) {
		key := c.Param("key")

		// 检查字符串存储
		stringPath := filepath.Join(stringDir, key+".txt")
		stringExists := false
		if _, err := os.Stat(stringPath); err == nil {
			stringExists = true
		}

		// 检查文件存储
		files, _ := filepath.Glob(filepath.Join(fileDir, key+".*"))
		fileExists := len(files) > 0

		c.JSON(http.StatusOK, gin.H{
			"exists": stringExists || fileExists,
			"details": gin.H{
				"string": stringExists,
				"file":   fileExists,
			},
		})
	})

	// 启动服务器
	r.Run(":8282")
}
