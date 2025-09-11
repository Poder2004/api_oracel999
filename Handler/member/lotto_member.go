package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"my-go-project/models"
)





func LottoLucky(c *gin.Context, db *gorm.DB) {

	const luckyLottoCount = 3

	const sql = "SELECT * FROM lotto WHERE status = ? ORDER BY RAND() LIMIT ?"

	var items []models.Lotto
	if err := db.Raw(sql, "sell", luckyLottoCount).Scan(&items).Error; err != nil {
		// จัดการ Error กรณีที่การ query ล้มเหลว (เหมือนเดิม)
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"total":  len(items),
		"data":   items,
	})
}

func LottoAuspicious(c *gin.Context, db *gorm.DB) {

	const luckyLottoCount = 3

	const sql = "SELECT * FROM lotto WHERE status = ? ORDER BY RAND() LIMIT ?"

	var items []models.Lotto
	if err := db.Raw(sql, "sell", luckyLottoCount).Scan(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"total":  len(items),
		"data":   items,
	})
}
