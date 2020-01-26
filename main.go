package main

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	client.Del("reserve")

	db, err := sql.Open("mysql", "root:1234@tcp(127.0.0.1:3306)/bp")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()
	db.Exec("UPDATE sale SET reserve=0")
	r := gin.Default()
	r.Use(cors.Default())
	dueDate := time.Date(2020, 1, 26, 07, 32, 0, 0, time.UTC)
	r.GET("/check", func(c *gin.Context) {
		now := time.Now()
		value, _ := client.Get("reserve").Result()
		no, _ := strconv.Atoi(value)
		status := "close"
		if now.After(dueDate) {
			status = "open"
			if no >= 10 {
				status = "sold"
			}
		}
		c.JSON(200, gin.H{
			"status": status,
		})
		return
	})

	r.GET("/ticket", func(c *gin.Context) {
		now := time.Now()
		if !now.After(dueDate) {
			c.JSON(200, gin.H{
				"success": false,
			})
			return
		}
		no, _ := client.Incr("reserve").Result()
		if no <= 10 {
			c.JSON(200, gin.H{
				"success": true,
				"ticket":  fmt.Sprintf("TX%02d", no),
			})
			return
		}
		c.JSON(200, gin.H{
			"success": false,
		})
		return
	})

	r.GET("/v2/check", func(c *gin.Context) {
		now := time.Now()
		status := "close"
		if now.After(dueDate) {
			status = "open"
			tx, _ := db.Begin()

			var reserve int
			row := tx.QueryRow("SELECT reserve FROM sale limit 1 FOR UPDATE")
			row.Scan(&reserve)
			if reserve >= 10 {
				status = "sold"
			}
			tx.Commit()
		}
		c.JSON(200, gin.H{
			"status": status,
		})
		return
	})

	r.GET("/v2/ticket", func(c *gin.Context) {
		now := time.Now()
		if !now.After(dueDate) {
			c.JSON(200, gin.H{
				"success": false,
			})
			return
		}

		var reserve int
		tx, _ := db.Begin()
		row := tx.QueryRow("SELECT reserve FROM sale FOR UPDATE")
		row.Scan(&reserve)
		if reserve < 10 {
			reserve++
			_, err := tx.Exec("UPDATE sale SET reserve=?", reserve)
			if err != nil {
				return
			}
			tx.Commit()
			c.JSON(200, gin.H{
				"success": true,
				"ticket":  fmt.Sprintf("TX%02d", reserve),
			})
			return
		}
		c.JSON(200, gin.H{
			"success": false,
		})
		tx.Commit()
		return
	})
	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
