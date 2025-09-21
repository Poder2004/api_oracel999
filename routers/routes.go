package routers

import (
	handlersadmin "my-go-project/Handler/admin"
	handlers "my-go-project/Handler/member"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SetupRouter ฟังก์ชันสำหรับตั้งค่า routes ของแอป
func SetupRouter(r *gin.Engine, db *gorm.DB) {

	r.POST("/register", func(c *gin.Context) {
		handlers.RegisterHandler(c, db)
	})

	r.POST("/login", func(c *gin.Context) {
		handlers.LoginHandler(c, db)
	})

	r.GET("/lotto/lucky", func(c *gin.Context) {
		handlers.LottoLucky(c, db)
	})

	r.GET("/lotto/Auspicious", func(c *gin.Context) {
		handlers.LottoAuspicious(c, db)
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

	r.GET("/lotto/random", func(c *gin.Context) {
		handlers.RandomLotto(c, db)
	})

	r.GET("/rewards/latest", func(c *gin.Context) {
		handlers.GetLatestRewards(c, db)
	})

	r.GET("/rewards/check", func(c *gin.Context) {
		handlers.CheckUserLotto(c, db)
	})
	r.POST("/rewards/cashIn", func(c *gin.Context) {
		handlers.CashIn(c, db)
	})

	//admin
	r.POST("/lotto/generate", func(c *gin.Context) {
		handlersadmin.ResetAndInsertLotto(c, db)
	})
	r.GET("/lotto", func(c *gin.Context) {
		handlersadmin.GetAllLotto(c, db)
	})

	r.GET("/lottos/count", func(c *gin.Context) {
		handlersadmin.LottoCount(c, db)
	})

	r.POST("/lotto/preview-update", func(c *gin.Context) {
		handlersadmin.PreviewNewLotto(c)
	})

	r.GET("/rewards/generate-preview", func(c *gin.Context) {
		handlersadmin.GenerateRewardsPreview(c, db)
	})

	r.POST("/rewards/release", func(c *gin.Context) {
		handlersadmin.ReleaseRewards(c, db)
	})

	r.GET("/rewards/currsent", func(c *gin.Context) {
		handlersadmin.GetCurrentRewards(c, db)

	})

	r.POST("/admin/clearData", func(c *gin.Context) {
		handlersadmin.ClearDataHandler(c, db)
	})
}
