package main

import (
	"capex/export"
	"capex/notification"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"golang.org/x/crypto/bcrypt"
)

type Claims struct {
	jwt.StandardClaims
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

var db *gorm.DB

func initDb() {
	dburl := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local", usernameDB, passwordDB, addressDB, portDB, dbName)
	db, _ = gorm.Open("mysql", dburl)
	// defer db.Close()

	db.SingularTable(true)
	db.AutoMigrate(&CapexTrx{}, &Plant{}, &Approval{}, &CapexAppr{}, &UserRule{}, &User{})
}

func getCreateInfo(c *gin.Context) {
	type budget struct {
		BudgetCode   string `json:"budgetCode"`
		BudgetAmount uint64 `json:"budgetAmount"`
		Remaining    uint64 `json:"budgetRemaining"`
		OwnerName    string `json:"ownerName"`
		Pernr        string `json:"payrollID"`
		Position     string `json:"position"`
		CostCenter   string `json:"costCenter"`
	}

	type purpose struct {
		IdPurpose string `json:"purposeID"`
		Desc      string `json:"purposeDesc"`
	}

	type plant struct {
		Plant     string `json:"plantCode"`
		PlantName string `json:"plantName"`
	}

	type sLoc struct {
		Sloc     string `json:"slocCode"`
		SlocName string `json:"slocName"`
	}

	type costCenter struct {
		Ccenter     string `json:"costCenterCode"`
		Description string `json:"costCenterName"`
	}

	type assetClass struct {
		Assetclass     string `json:"assetClassCode"`
		DescAssetclass string `json:"assetClassDesc"`
	}

	type actType struct {
		IDActtype   string `json:"actTypeCode"`
		DescActtype string `json:"actTypeDesc"`
	}

	type assetGroup struct {
		IDAstgroup   string `json:"assetGrpCode"`
		DescAstgroup string `json:"assetGrpDesc"`
	}

	infoBody := struct {
		BudgetInfo     []budget     `json:"budgetInfo"`
		PurposeInfo    []purpose    `json:"purposeInfo"`
		CostCenterInfo []costCenter `json:"costCenterInfo"`
		PlantInfo      []plant      `json:"plantInfo"`
		SlocInfo       []sLoc       `json:"slocInfo"`
		AssetClassInfo []assetClass `json:"assetClassInfo"`
		ActTypeInfo    []actType    `json:"actTypeInfo"`
		AssetGrpInfo   []assetGroup `json:"assetGrpInfo"`
	}{}

	err := db.Table("tb_budget").Find(&infoBody.BudgetInfo).Error
	if err != nil {
		c.AbortWithStatus(404)
		return
	}

	err = db.Table("tb_purpose").Find(&infoBody.PurposeInfo).Error
	if err != nil {
		c.AbortWithStatus(404)
		return
	}

	err = db.Table("tb_ccenter").Find(&infoBody.CostCenterInfo).Error
	if err != nil {
		c.AbortWithStatus(404)
		return
	}

	err = db.Table("tb_plant").Find(&infoBody.PlantInfo).Error
	if err != nil {
		c.AbortWithStatus(404)
		return
	}

	err = db.Table("tb_sloc").Find(&infoBody.SlocInfo).Error
	if err != nil {
		c.AbortWithStatus(404)
		return
	}

	err = db.Table("tb_assetclass").Find(&infoBody.AssetClassInfo).Error
	if err != nil {
		c.AbortWithStatus(404)
		return
	}

	err = db.Table("tb_acttype").Find(&infoBody.ActTypeInfo).Error
	if err != nil {
		c.AbortWithStatus(404)
		return
	}

	err = db.Table("tb_astgroup").Find(&infoBody.AssetGrpInfo).Error
	if err != nil {
		c.AbortWithStatus(404)
		return
	}

	c.JSON(200, infoBody)
}

func getRules(c *gin.Context) {
	id, err := validateID(c)
	if err != nil {
		return
	}

	cookie, err := c.Cookie("testCookie")
	log.Println("Cookie:", cookie)

	var userRules []UserRule
	err = db.Where("user_id = ?", id).Find(&userRules).Error
	if err != nil || len(userRules) <= 0 {
		c.AbortWithStatus(404)
		return
	}

	ruleBody := struct {
		UserID uint     `json:"userID"`
		Rule   []string `json:"rule"`
	}{}

	ruleBody.UserID = uint(id)
	for _, rule := range userRules {
		ruleBody.Rule = append(ruleBody.Rule, rule.Rule)
	}
	c.SetCookie("testCookie", "ABCDEF", 0, "/", "localhost", true, true)
	c.JSON(200, ruleBody)
}

func getCapexTrx(c *gin.Context) {
	var err error

	createdBy := c.Query("created")
	waitAppr := c.Query("wait_appr")
	// accAppr := c.Query("acc_appr")

	var capexTrxAll []CapexTrx
	if createdBy != "" {
		err = db.Where("created_by = ?", createdBy).Find(&capexTrxAll).Error
	} else if waitAppr != "" {
		var userRules UserRule
		err = db.Where("user_id = ? AND rule = 'ACCAPPROVER'", waitAppr).First(&userRules).Error
		if userRules.Rule == "ACCAPPROVER" {
			err = db.Where("acc_approved = ''").Find(&capexTrxAll).Error
		} else {
			err = db.Where("next_approval = ?", waitAppr).Find(&capexTrxAll).Error
		}
	} else {
		err = db.Find(&capexTrxAll).Error
	}

	if err != nil || len(capexTrxAll) <= 0 {
		c.AbortWithStatus(404)
		fmt.Println(err)
	} else {
		c.JSON(200, capexTrxAll)
	}
}

func getCapexTrxDetail(c *gin.Context) {
	var err error

	ID := c.Param("id")

	type capexApprover struct {
		Seq       uint
		Approver  uint
		Name      string
		Status    string
		Remark    string
		CreatedAt time.Time
		UpdatedAt time.Time
	}

	type requestor struct {
		Id        uint
		Name      string
		Position  string
		PayrollID string
	}

	capexBody := struct {
		CapexDetail CapexTrx        `json:"capexDetail"`
		Approver    []capexApprover `json:"approver"`
		Requestor   requestor       `json:"requestorInfo"`
	}{}

	var capexTrx CapexTrx

	err = db.Where("id = ?", ID).First(&capexTrx).Error
	if err != nil {
		c.AbortWithStatus(404)
		fmt.Println(err)
		return
	}

	capexBody.CapexDetail = capexTrx

	// var capexAppr []CapexAppr
	// err = db.Where("capex_id = ?", ID).Find(&capexBody.Approver).Error
	err = db.Table("capex_appr as c").
		Select("c.seq, c.approver, u.name, c.status, c.remark, c.created_at, c.updated_at").
		Joins("JOIN user as u on c.approver = u.id").
		Where("c.capex_id = ?", ID).
		Find(&capexBody.Approver).
		Error

	err = db.Table("user").
		Select("id, name, position, payroll_id").
		Where("id = ?", capexBody.CapexDetail.CreatedBy).
		First(&capexBody.Requestor).
		Error

	for idx, approver := range capexBody.Approver {
		if approver.CreatedAt == approver.UpdatedAt {
			capexBody.Approver[idx].UpdatedAt = time.Time{}
		}
	}
	// c.JSON(200, capexBody)
	c.JSON(200, capexBody)
	return
}

func validateID(c *gin.Context) (id float64, err error) {
	id = c.MustGet("ID").(float64)
	if id == 0 {
		c.AbortWithError(http.StatusNotFound, errors.New("User unknown"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "User unknown",
		})
		return 0, errors.New("User unknown")
	}
	return id, nil
}

func createCapexTrx(c *gin.Context) {

	var err error

	id := c.MustGet("ID").(float64)
	if id == 0 {
		c.AbortWithError(http.StatusNotFound, errors.New("User unknown"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "User unknown",
		})
		return
	}

	var capexTrx CapexTrx
	err = c.BindJSON(&capexTrx)
	if err != nil {
		c.AbortWithError(http.StatusNotFound, err)
		c.JSON(http.StatusNotFound, gin.H{
			"message": err.Error(),
		})
		return
	}

	capexTrx.CreatedBy = uint(id)

	remainingAmount := struct {
		Remaining uint64
	}{}
	if capexTrx.BudgetType == "B" {
		err = db.Table("tb_budget").
			Select("remaining").
			Where("budget_code = ?", capexTrx.BudgetApprovalCode).
			First(&remainingAmount).
			Error
		if err != nil || remainingAmount.Remaining <= 0 {
			c.AbortWithError(http.StatusNotFound, errors.New("budget amount is not enough"))
			c.JSON(http.StatusNotFound, gin.H{
				"message": "budget amount is not enough",
			})
			return
		}

		remainingAmount.Remaining -= capexTrx.TotalAmount
		if remainingAmount.Remaining < 0 {
			c.AbortWithError(http.StatusNotFound, errors.New("budget amount is not enough"))
			c.JSON(http.StatusNotFound, gin.H{
				"message": "budget amount is not enough",
			})
			return
		}
	}

	tx := db.Begin()
	err = tx.Create(&capexTrx).Error
	if err != nil {
		log.Println("")
		tx.Rollback()
		c.AbortWithError(http.StatusInternalServerError, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	err = tx.Table("tb_budget").
		Where("budget_code = ?", capexTrx.BudgetApprovalCode).
		Updates(map[string]interface{}{"remaining": remainingAmount.Remaining}).Error
	if err != nil {
		tx.Rollback()
		c.AbortWithError(http.StatusInternalServerError, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	err = tx.Commit().Error
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	go notifAccounting(capexTrx.ID)

	c.JSON(200, capexTrx)
	return
}

func updateCapexTrx(c *gin.Context) {
	_, err := validateID(c)
	if err != nil {
		return
	}

	capexID := c.Param("id")

	var resBody, capexTrx CapexTrx
	c.BindJSON(&resBody)
	if resBody.AssetClass == "" {
		log.Println("ERROR BINDING")
		c.AbortWithError(http.StatusNotFound, errors.New("Asset Class must be filled"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Asset Class must be filled",
		})
		return
	}

	err = db.Where("id = ?", capexID).First(&capexTrx).Error
	if err != nil || capexTrx.ID == 0 {
		c.AbortWithError(http.StatusNotFound, errors.New("Capex not found"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Capex not found",
		})
		return
	}

	capexTrx.AssetClass = resBody.AssetClass
	capexTrx.AssetActivityType = resBody.AssetActivityType
	capexTrx.AssetGroup = resBody.AssetGroup
	capexTrx.ACCApproved = "X"
	capexTrx.Status = "I"

	var approval []Approval
	err = db.Where("departement = ? and asset_type = ? and unbudgeted = ? and it = ? and amount_low <= ? and amount_high >= ?", "IT", "IT01", 0, 1, capexTrx.TotalAmount, capexTrx.TotalAmount).
		Order("seq").
		Find(&approval).
		Error
	if err != nil || len(approval) <= 0 {
		c.AbortWithError(http.StatusNotFound, errors.New("Approval not found"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Approval not found",
		})
		return
	}

	for _, appr := range approval {
		if appr.Seq == 1 {
			capexTrx.NextApproval = appr.Approver
			break
		}
	}

	tx := db.Begin()
	err = tx.Model(&capexTrx).Updates(CapexTrx{AssetClass: resBody.AssetClass, AssetActivityType: resBody.AssetActivityType, AssetGroup: resBody.AssetGroup, ACCApproved: "X", Status: "I", NextApproval: capexTrx.NextApproval}).Error
	// err = tx.Save(&capexTrx).Error
	if err != nil {
		tx.Rollback()
		c.AbortWithError(http.StatusInternalServerError, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	// errorFound := false

	for _, appr := range approval {
		err = tx.Create(&CapexAppr{
			CapexID:   capexTrx.ID,
			Seq:       appr.Seq,
			Approver:  appr.Approver,
			Status:    "",
			Remark:    "",
			CreatedAt: time.Now(),
		}).Error
		if err != nil {
			// errorFound = true
			tx.Rollback()
			c.AbortWithError(http.StatusInternalServerError, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}
	}

	err = tx.Commit().Error
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	go notifApprover(capexTrx.ID, capexTrx.NextApproval)

	c.JSON(200, capexTrx)
	return
}

func approveCapex(c *gin.Context) {

	id := c.MustGet("ID").(float64)
	if id == 0 {
		c.AbortWithError(http.StatusNotFound, errors.New("User unknown"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "User unknown",
		})
		return
	}

	approveBody := struct {
		CapexID uint `json:"capexID"`
		Seq     uint `json:"seq"`
	}{}

	c.BindJSON(&approveBody)

	var capexTrx CapexTrx
	err := db.Where("id = ?", approveBody.CapexID).First(&capexTrx).Error
	if err != nil || capexTrx.ID == 0 {
		c.AbortWithError(http.StatusNotFound, errors.New("Capex ID not valid"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Capex ID not valid",
		})
		return
	}

	if capexTrx.NextApproval != uint(id) {
		c.AbortWithError(http.StatusNotFound, errors.New("Invalid approver"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Invalid approver",
		})
		return
	}

	switch capexTrx.Status {
	case "A":
		c.AbortWithError(http.StatusNotFound, errors.New("Capex fully approve"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Capex fully approve",
		})
		return
	case "R":
		c.AbortWithError(http.StatusNotFound, errors.New("Capex rejected"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Capex rejected",
		})
		return
	}

	var capexAppr []CapexAppr
	err = db.Where("capex_id = ?", approveBody.CapexID).Order("seq").Find(&capexAppr).Error
	if err != nil || len(capexAppr) <= 0 {
		c.AbortWithError(http.StatusNotFound, errors.New("Approval Workflow not found"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Approval Workflow not found",
		})
		return
	}

	var appr CapexAppr
	var idx int
	for i, approver := range capexAppr {
		if approver.Status == "" {
			if approver.Approver != uint(id) {
				c.AbortWithError(http.StatusNotFound, errors.New("Not authorized to approve"))
				c.JSON(http.StatusNotFound, gin.H{
					"message": "Not authorized to approve",
				})
				return
			}
			idx = i
			appr = approver
			break
		} else {
			if approver.Seq == approveBody.Seq && approver.Status != "" {
				c.AbortWithError(http.StatusNotFound, errors.New("Approval has been processed"))
				c.JSON(http.StatusNotFound, gin.H{
					"message": "Approval has been processed",
				})
				return
			}
		}

	}

	errorFound := false
	tx := db.Begin()
	err = tx.Model(&appr).Update("status", "A").Error //Approve
	if err != nil {
		errorFound = true
		// tx.Rollback()
		// c.AbortWithError(http.StatusInternalServerError, err)
		// c.JSON(http.StatusInternalServerError, gin.H{
		// 	"message": err.Error(),
		// })
		// return
	}

	capexAppr[idx].Status = "A"
	var appr2 CapexAppr

	for i, appr := range capexAppr {
		if appr.Status == "" {
			for _, appr2 = range capexAppr {
				if appr2.Approver == appr.Approver && appr2.Status == "A" {
					capexAppr[i].Status = "A"
					appr2.Status = "A"
					err = tx.Model(&appr).Update("status", "A").Error
					if err != nil {
						errorFound = true
					}
					break
				}
			}
			if appr2.Status == "" {
				break
			}
		}
	}

	stillNeedApproval := false

	if !errorFound {
		for _, appr := range capexAppr {
			if appr.Status == "" {
				stillNeedApproval = true
				err = tx.Model(&capexTrx).Update("next_approval", appr.Approver).Error
				if err != nil {
					errorFound = true
					// tx.Rollback()
					// c.AbortWithError(http.StatusInternalServerError, err)
					// c.JSON(http.StatusInternalServerError, gin.H{
					// 	"message": err.Error(),
					// })
					// return
				} else {
					go notifApprover(capexTrx.ID, appr.Approver)
				}
				break
			}
		}
	}

	if !errorFound {
		if !stillNeedApproval {
			err = tx.Model(&capexTrx).Updates(map[string]interface{}{"status": "A", "next_approval": 0}).Error
			if err != nil {
				errorFound = true
				// tx.Rollback()
				// c.AbortWithError(http.StatusInternalServerError, err)
				// c.JSON(http.StatusInternalServerError, gin.H{
				// 	"message": err.Error(),
				// })
				// return
			} else {
				go notifFullApprove(capexTrx.ID)
				go exportToCSV(capexTrx)
			}
		}
	}

	if errorFound {
		tx.Rollback()
		c.AbortWithError(http.StatusInternalServerError, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	err = tx.Commit().Error
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"message": "Approve success",
	})
	return
}

func rejectCapex(c *gin.Context) {
	id := c.MustGet("ID").(float64)
	if id == 0 {
		c.AbortWithError(http.StatusNotFound, errors.New("User unknown"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "User unknown",
		})
		return
	}

	rejectBody := struct {
		CapexID uint   `json:"capexID"`
		Seq     uint   `json:"seq"`
		Remark  string `json:"remark"`
	}{}

	c.BindJSON(&rejectBody)

	var capexTrx CapexTrx
	err := db.Where("id = ?", rejectBody.CapexID).First(&capexTrx).Error
	if err != nil || capexTrx.ID == 0 {
		c.AbortWithError(http.StatusNotFound, errors.New("Capex ID not valid"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Capex ID not valid",
		})
		return
	}

	if capexTrx.NextApproval != uint(id) {
		c.AbortWithError(http.StatusNotFound, errors.New("Invalid approver"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Invalid approver",
		})
		return
	}

	var capexAppr []CapexAppr
	err = db.Where("capex_id = ?", rejectBody.CapexID).Order("seq").Find(&capexAppr).Error
	if err != nil || len(capexAppr) <= 0 {
		c.AbortWithError(http.StatusNotFound, errors.New("Approval Workflow not found"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Approval Workflow not found",
		})
		return
	}

	var appr CapexAppr
	// var idx int
	for _, approver := range capexAppr {
		if approver.Status == "" {
			if approver.Approver != uint(id) {
				c.AbortWithError(http.StatusNotFound, errors.New("Not authorized to approve"))
				c.JSON(http.StatusNotFound, gin.H{
					"message": "Not authorized to approve",
				})
				return
			}
			// idx = i
			appr = approver
			break
		} else {
			if approver.Seq == rejectBody.Seq && approver.Status != "" {
				c.AbortWithError(http.StatusNotFound, errors.New("Approval has been processed"))
				c.JSON(http.StatusNotFound, gin.H{
					"message": "Approval has been processed",
				})
				return
			}
		}

	}

	appr.Status = "R"

	errorFound := false
	tx := db.Begin()
	err = tx.Model(&appr).Updates(CapexAppr{Status: "R", Remark: rejectBody.Remark}).Error
	if err != nil {
		errorFound = true
	}

	if errorFound {
		tx.Rollback()
		c.AbortWithError(http.StatusInternalServerError, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	if !errorFound {
		err = tx.Model(&capexTrx).Updates(map[string]interface{}{"status": "R", "next_approval": 0}).Error
		if err != nil {
			tx.Rollback()
			c.AbortWithError(http.StatusInternalServerError, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}
	}

	err = tx.Table("tb_budget").
		Where("budget_code = ?", capexTrx.BudgetApprovalCode).
		Updates(map[string]interface{}{"remaining": gorm.Expr("remaining + ?", capexTrx.TotalAmount)}).
		Error
	if err != nil {
		tx.Rollback()
		c.AbortWithError(http.StatusInternalServerError, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	err = tx.Commit().Error
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	go notifReject(capexTrx.ID, rejectBody.Remark)

	c.JSON(200, gin.H{
		"message": "Reject success",
	})
}

func register(c *gin.Context) {
	var user User
	var currentUser User
	c.BindJSON(&user)
	if err := db.Debug().Where("username = ?", user.Username).First(&currentUser).Error; err == nil {
		c.AbortWithError(http.StatusNotFound, errors.New("Username already exists"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Username already exists",
		})
		return
	}
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	user.Password = string(hashedPassword)
	db.Create(&user)
	c.JSON(201, user)
}

func login(c *gin.Context) {

	loginBody := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{}

	c.BindJSON(&loginBody)

	var user User
	if err := db.Where("username = ?", loginBody.Username).First(&user).Error; err != nil {
		c.AbortWithError(http.StatusNotFound, errors.New("Username not found"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Username not found",
		})
		return
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginBody.Password))
	if err != nil {
		c.AbortWithError(http.StatusNotFound, errors.New("Username or password mismatch"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Username or password mismatch",
		})
		return
	}

	claims := Claims{
		StandardClaims: jwt.StandardClaims{
			Issuer:    "SIDOMUNCUL",
			ExpiresAt: time.Now().Add(time.Duration(24) * time.Hour).Unix(),
		},
		ID:       user.ID,
		Username: user.Username,
		Name:     user.Name,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	respondBodyLogin := struct {
		ID       uint   `json:"id"`
		Name     string `json:"name"`
		Username string `json:"username"`
		Token    string `json:"token"`
	}{}

	respondBodyLogin.ID = user.ID
	respondBodyLogin.Name = user.Name
	respondBodyLogin.Username = user.Username
	respondBodyLogin.Token, err = token.SignedString([]byte(signatureKey))
	if err != nil {
		c.AbortWithError(http.StatusNotFound, err)
		c.JSON(http.StatusNotFound, gin.H{
			"message": err,
		})
		return
	}

	c.JSON(200, respondBodyLogin)

}

func notifApprover(trxID uint, approverID uint) {
	to := []string{}
	subject := "Capex " + strconv.Itoa(int(trxID))
	var user User
	_ = db.Where("ID = ?", approverID).First(&user).Error
	if user.ID == 0 {
		return
	}

	// to = append(to, user.Email)
	to = append(to, user.Email)

	bodyVar := map[string]string{
		"Name":    user.Name,
		"CapexID": strconv.Itoa(int(trxID)),
	}

	notification.SendEmail(to, subject, "approval.html", bodyVar)
}

func notifAccounting(trxID uint) {
	to := []string{}
	subject := "Capex " + strconv.Itoa(int(trxID))
	var users []User
	_ = db.Where("accounting = ?", "X").Find(&users).Error
	if len(users) == 0 {
		return
	}

	for _, user := range users {
		to = append(to, user.Email)
	}

	bodyVar := map[string]string{
		"Name":    "Accounting Team",
		"CapexID": strconv.Itoa(int(trxID)),
	}

	notification.SendEmail(to, subject, "accounting-appr.html", bodyVar)
}

func notifReject(trxID uint, message string) {

	user := struct {
		Email string
		Name  string
	}{}
	_ = db.Debug().Table("capex_trx as c").
		Select("u.email, u.name").
		Joins("JOIN user as u on c.created_by = u.id").
		Where("c.id = ?", trxID).
		Find(&user).
		Error

	to := []string{user.Email}
	subject := "Reject Capex " + strconv.Itoa(int(trxID))

	bodyVar := map[string]string{
		"Name":    user.Name,
		"CapexID": strconv.Itoa(int(trxID)),
		"Message": message,
	}

	notification.SendEmail(to, subject, "reject-capex.html", bodyVar)

}

func notifFullApprove(trxID uint) {

	user := struct {
		Email string
		Name  string
	}{}
	_ = db.Debug().Table("capex_trx as c").
		Select("u.email, u.name").
		Joins("JOIN user as u on c.created_by = u.id").
		Where("c.id = ?", trxID).
		Find(&user).
		Error

	to := []string{user.Email}
	subject := "Capex " + strconv.Itoa(int(trxID)) + " Full Approved"

	bodyVar := map[string]string{
		"Name":    user.Name,
		"CapexID": strconv.Itoa(int(trxID)),
	}

	notification.SendEmail(to, subject, "full-approve.html", bodyVar)

}

func exportToCSV(trx CapexTrx) {
	contents := [][]string{}

	content := []string{
		strconv.Itoa(int(trx.ID)),
		strconv.Itoa(int(trx.Quantity)),
		trx.Uom,
		strconv.Itoa(int(trx.TotalAmount)),
		"IDR",
		trx.Description,
	}
	contents = append(contents, content)

	export.SaveCSV("CAPEX", strconv.Itoa(int(trx.ID)), contents)
}
