package routers

import (
	handlers "my-go-project/Handler"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SetupRouter ฟังก์ชันสำหรับตั้งค่า routes ของแอป
func SetupRouter(r *gin.Engine, db *gorm.DB) {
	// ตั้งค่าเส้นทางสำหรับการสมัครสมาชิก
	r.POST("/register", func(c *gin.Context) {
		handlers.RegisterHandler(c, db)
	})

	// ตั้งค่าเส้นทางสำหรับการล็อกอิน
	r.POST("/login", func(c *gin.Context) {
		handlers.LoginHandler(c, db)
	})

	// routers.go
	r.GET("/lotto", func(c *gin.Context) {
		handlers.GetAllLottoASC(c, db)
	})

	r.GET("/lotto/sell", func(c *gin.Context) {
		handlers.GetLottoSell(c, db)
	})

	r.GET("/lotto/lucky", func(c *gin.Context) {
		handlers.LottoLucky(c, db)
	})

	r.GET("/lotto/Auspicious", func(c *gin.Context) {
		handlers.LottoAuspicious(c, db)
	})

	r.POST("/lotto/generate", func(c *gin.Context) {
		handlers.InsertLotto(c, db)
	})

	r.POST("/purchases", func(c *gin.Context) { handlers.CreatePurchase(c, db) }) // ซื้อจริง

	r.GET("/users/purchases", func(c *gin.Context) {
		handlers.ListPurchasedLottosByUser(c, db)
	})

	r.GET("/profile", func(c *gin.Context) {
		handlers.Profile(c, db)
	})

	r.GET("/wallet", func(c *gin.Context) {
		handlers.Wallet(c, db)
	})

	r.GET("/lotto/search", func(c *gin.Context) {
		handlers.SearchLottoByNumber(c, db)
	})

	// --- 2. แก้ไข: เปลี่ยนชื่อ Handler ให้ถูกต้อง ---
	r.GET("/lotto/random", func(c *gin.Context) {
		handlers.RandomLotto(c, db)
	})

	r.GET("/rewards/latest", func(c *gin.Context) {
		handlers.GetLatestRewards(c, db)
	})
	r.GET("/rewards/check", func(c *gin.Context) {
		handlers.CheckUserLotto(c, db)
	})

	//ของกุที่เพิ่มาใหม่
	r.POST("/lotto/generate", func(c *gin.Context) {
		handlers.InsertLotto(c, db)
	})

	// r.GET("/lottos/count", func(c *gin.Context) {
	// 	handlers.LottoCount(c, db)
	// })

	// r.GET("/lotto/search", func(c *gin.Context) {
	// 	handlers.SearchLotto(c, db)
	// })

	// r.GET("/lotto/random", func(c *gin.Context) {
	// 	handlers.RandomLotto(c, db)
	// })

	// r.POST("/lotto/preview-update", func(c *gin.Context) {
	// 	handlers.PreviewUpdateLotto(c, db)
	// })

	// r.POST("/lotto/bulk-update", func(c *gin.Context) {
	// 	handlers.BulkUpdateLottoNumbers(c, db)
	// })

	// r.GET("/rewards/generate-preview", func(c *gin.Context) { handlers.GenerateRewardsPreview(c, db) })
	// r.POST("/rewards/release", func(c *gin.Context) { handlers.ReleaseRewards(c, db) })

	// r.GET("/rewards/currsent", func(c *gin.Context) {
	// 	handlers.GetCurrentRewards(c, db)

	// })

}
