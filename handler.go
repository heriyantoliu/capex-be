package main

import (
	"capex/export"
	"capex/notification"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/dustin/go-humanize"
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
	db.AutoMigrate(&CapexTrx{}, &Plant{}, &Approval{}, &CapexAppr{}, &UserRole{}, &User{}, &CapexAsset{}, &UserCostCenterRole{}, &CostCenterRole{})
}

func getCreateInfo(c *gin.Context) {
	type budget struct {
		BudgetCode   string `json:"budgetCode"`
		BudgetAmount int64  `json:"budgetAmount"`
		Remaining    int64  `json:"budgetRemaining"`
		OwnerName    string `json:"ownerName"`
		Pernr        string `json:"payrollID"`
		Position     string `json:"position"`
		CostCenter   string `json:"costCenter"`
		BudgetDesc   string `json:"budgetDesc"`
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

	type uom struct {
		Uom  string `json:"uom"`
		Desc string `json:"desc"`
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
		UomInfo        []uom        `json:"uomInfo"`
	}{}

	username := c.MustGet("USERNAME").(string)
	if username == "" {
		c.AbortWithError(http.StatusNotFound, errors.New("Username unknown"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Username unknown",
		})
		return
	}

	err := db.Table("tb_purpose").Find(&infoBody.PurposeInfo).Error
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

	err = db.Table("tb_uom").Find(&infoBody.UomInfo).Error
	if err != nil {
		c.AbortWithStatus(404)
		return
	}

	err = db.Debug().Table("tb_ccenter as c").
		Select("c.ccenter, c.description").
		Joins("JOIN cost_center_role as cr on c.ccenter = cr.cost_center").
		Joins("JOIN user_cost_center_role as ucr on cr.role = ucr.role").
		Where("ucr.username = ?", username).
		Order("c.ccenter").
		Find(&infoBody.CostCenterInfo).Error
	if err != nil {
		c.AbortWithStatus(404)
		return
	}

	err = db.Debug().Table("tb_budget as b").
		Select("b.budget_code, b.budget_amount, b.remaining, b.owner_name, b.pernr, b.position, b.cost_center, b.budget_desc").
		Joins("JOIN cost_center_role as cr on b.cost_center = cr.cost_center").
		Joins("JOIN user_cost_center_role as ucr on cr.role = ucr.role").
		Where("ucr.username = ?", username).
		Order("b.cost_center").
		Find(&infoBody.BudgetInfo).Error
	if err != nil {
		c.AbortWithStatus(404)
		return
	}

	c.JSON(200, infoBody)
}

func getRoles(c *gin.Context) {
	usernameToken, err := validateUsername(c)
	if err != nil {
		return
	}

	username := c.Param("id")

	if username != usernameToken {
		c.AbortWithStatus(404)
		return
	}

	var userRoles []UserRole
	err = db.Where("username = ?", username).Find(&userRoles).Error
	if err != nil || len(userRoles) <= 0 {
		c.AbortWithStatus(404)
		return
	}

	roleBody := struct {
		Username string   `json:"username"`
		Role     []string `json:"role"`
	}{}

	roleBody.Username = username
	for _, role := range userRoles {
		roleBody.Role = append(roleBody.Role, role.Role)
	}

	c.JSON(200, roleBody)
	return
}

func getCapexTrx(c *gin.Context) {
	var err error

	createdBy := c.Query("created")
	waitAppr := c.Query("wait_appr")
	replicate, _ := strconv.ParseBool(c.Query("replicate"))

	var capexTrxAll []CapexTrx
	if createdBy != "" {
		err = db.Where("created_by = ?", createdBy).Find(&capexTrxAll).Error
	} else if waitAppr != "" {
		var userRoles UserRole
		err = db.Where("username = ? AND role = 'ACCAPPROVER'", waitAppr).First(&userRoles).Error
		if userRoles.Role == "ACCAPPROVER" {
			err = db.Where("acc_approved = ''").Or("next_approval = ?", waitAppr).Find(&capexTrxAll).Error
		} else {
			err = db.Where("next_approval = ?", waitAppr).Find(&capexTrxAll).Error
		}
	} else if replicate {
		err = db.Where("status in (?)", []string{"A", "SAP", "RI"}).Find(&capexTrxAll).Error
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
		Approver  string
		Status    string
		Remark    string
		CreatedAt time.Time
		UpdatedAt time.Time
		Name      string
	}

	type requestor struct {
		Username  string
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

	var ccRole CostCenterRole

	err = db.Table("cost_center_role as cr").
		Select("cr.cost_center").
		Joins("JOIN user_cost_center_role as ucr on cr.role = ucr.role").
		Joins("JOIN capex_trx as trx on ucr.cost_center = trx.cost_center").
		Where("trx.cost_center = ?", capexTrx.CostCenter).
		First(&ccRole).
		Error
	if err != nil {
		c.AbortWithError(http.StatusForbidden, errors.New("Not Authorized"))
		c.JSON(http.StatusForbidden, gin.H{
			"message": "Not Authorized",
		})
		return
	}

	capexBody.CapexDetail = capexTrx

	// var capexAppr []CapexAppr
	// err = db.Where("capex_id = ?", ID).Find(&capexBody.Approver).Error
	err = db.Table("capex_appr as c").
		Select("c.seq, c.approver, c.status, c.remark, c.created_at, c.updated_at, u.name").Joins("JOIN user as u on c.approver = u.username").
		Where("c.capex_id = ?", ID).Order("seq").
		Find(&capexBody.Approver).
		Error

	err = db.Table("user").
		Select("username, name, position, payroll_id").
		Where("username = ?", capexBody.CapexDetail.CreatedBy).
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

func validateUsername(c *gin.Context) (username string, err error) {
	username = c.MustGet("USERNAME").(string)
	if username == "" {
		c.AbortWithError(http.StatusNotFound, errors.New("Username unknown"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Username unknown",
		})
		return "", errors.New("Username unknown")
	}
	return username, nil
}

func createCapexTrx(c *gin.Context) {

	var err error

	username := c.MustGet("USERNAME").(string)
	if username == "" {
		c.AbortWithError(http.StatusNotFound, errors.New("Username unknown"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Username unknown",
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

	capexTrx.CreatedBy = username

	var user User

	_ = db.Where("username = ?", username).First(&user).Error

	capexTrx.RequestorPosition = user.Position

	remainingAmount := struct {
		Remaining int64
	}{}
	if capexTrx.BudgetType == "B" {
		err = db.Table("tb_budget").
			Select("remaining").
			Where("budget_code = ?", capexTrx.BudgetApprovalCode).
			First(&remainingAmount).
			Error
		if err != nil {
			c.AbortWithError(http.StatusNotFound, errors.New("budget code tidak valid"))
			c.JSON(http.StatusNotFound, gin.H{
				"message": "budget code tidak valid",
			})
			return
		}

		remainingAmount.Remaining -= int64(capexTrx.TotalAmount)

	}

	tx := db.Begin()
	err = tx.Create(&capexTrx).Error
	if err != nil {
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

func createCapexAsset(c *gin.Context) {

	var capexAsset CapexAsset
	c.BindJSON(&capexAsset)

	capexID := c.Param("id")

	var capexTrx CapexTrx

	err := db.Where("id = ?", capexID).First(&capexTrx).Error
	if err != nil || capexTrx.ID == 0 {
		c.AbortWithError(http.StatusNotFound, errors.New("Capex not found"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Capex not found",
		})
		return
	}

	capexIDUint, _ := strconv.ParseUint(capexID, 10, 0)

	capexAsset.CapexID = uint(capexIDUint)

	tx := db.Begin()
	err = tx.Create(&capexAsset).Error
	if err != nil {
		tx.Rollback()
		c.AbortWithError(http.StatusInternalServerError, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	err = tx.Model(&capexTrx).Update("Status", "SAP").Error
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

	c.JSON(http.StatusOK, capexTrx)
	return

}

func getCapexAsset(c *gin.Context) {
	_, err := validateUsername(c)
	if err != nil {
		return
	}

	capexID := c.Param("id")

	var capexAsset []CapexAsset

	err = db.Where("capex_id = ?", capexID).Find(&capexAsset).Error
	if err != nil {
		c.AbortWithError(http.StatusNotFound, errors.New("Asset not found"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Asset not found",
		})
		return
	}

	c.JSON(http.StatusOK, capexAsset)
	return

}

func updateCapexTrx(c *gin.Context) {
	username, err := validateUsername(c)
	if err != nil {
		return
	}

	capexID := c.Param("id")

	var resBody, capexTrx CapexTrx
	c.BindJSON(&resBody)
	if resBody.AssetClass == "" {
		c.AbortWithError(http.StatusNotFound, errors.New("Asset Class must be filled"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Asset Class must be filled",
		})
		return
	}

	if resBody.AssetGenMode == "" {
		c.AbortWithError(http.StatusNotFound, errors.New("Asset generation mode must be filled"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Asset generation mode must be filled",
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
	capexTrx.AssetGenMode = resBody.AssetGenMode
	capexTrx.ACCApproved = "X"
	capexTrx.Status = "I"

	var approval []Approval
	err = db.Where("cost_center = ? and asset_class = ? and amount_low <= ? and amount_high >= ?",
		capexTrx.CostCenter,
		capexTrx.AssetClass,
		capexTrx.TotalAmount,
		capexTrx.TotalAmount,
	).Order("seq").Find(&approval).Error
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
	err = tx.Model(&capexTrx).Updates(CapexTrx{
		AssetClass:        resBody.AssetClass,
		AssetActivityType: resBody.AssetActivityType,
		AssetGroup:        resBody.AssetGroup,
		AssetGenMode:      resBody.AssetGenMode,
		ACCApproved:       "X",
		Status:            "I",
		NextApproval:      capexTrx.NextApproval,
	}).Error
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

	go notifApprover(capexTrx.ID, capexTrx.NextApproval, username)

	c.JSON(200, capexTrx)
	return
}

func replicateCapex(c *gin.Context) {

	_, err := validateUsername(c)
	if err != nil {
		return
	}

	ID := c.Param("id")

	var capexTrx CapexTrx
	err = db.Where("id = ?", ID).First(&capexTrx).Error
	if err != nil {
		c.AbortWithError(http.StatusNotFound, errors.New("Invalid Capex ID"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Invalid Capex ID",
		})
		return
	}

	if capexTrx.Status != "A" {
		c.AbortWithError(http.StatusBadRequest, errors.New("Capex not fully approved or replicated already"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Capex not fully approved or replicated already",
		})
		return
	}

	err = exportToCSV(capexTrx)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	capexTrx.Status = "RI"
	db.Model(&capexTrx).Update("status", "RI")

	c.JSON(http.StatusOK, capexTrx)
	return
}

func approveCapex(c *gin.Context) {

	username, err := validateUsername(c)
	if err != nil {
		return
	}

	capexID := c.Param("id")

	// approveBody := struct {
	// 	CapexID uint `json:"capexID"`
	// 	Seq     uint `json:"seq"`
	// }{}

	// c.BindJSON(&approveBody)

	var capexTrx CapexTrx
	err = db.Where("id = ?", capexID).First(&capexTrx).Error
	if err != nil || capexTrx.ID == 0 {
		c.AbortWithError(http.StatusNotFound, errors.New("Capex ID not valid"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Capex ID not valid",
		})
		return
	}

	if capexTrx.NextApproval != username {
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
	err = db.Where("capex_id = ?", capexID).Order("seq").Find(&capexAppr).Error
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
			if approver.Approver != username {
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
			if approver.Approver == username {
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
					go notifApprover(capexTrx.ID, appr.Approver, username)
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
	username, err := validateUsername(c)
	if err != nil {
		return
	}

	capexID := c.Param("id")

	rejectBody := struct {
		Remark string `json:"remark"`
	}{}

	c.BindJSON(&rejectBody)

	var capexTrx CapexTrx
	err = db.Where("id = ?", capexID).First(&capexTrx).Error
	if err != nil || capexTrx.ID == 0 {
		c.AbortWithError(http.StatusNotFound, errors.New("Capex ID not valid"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Capex ID not valid",
		})
		return
	}

	if capexTrx.NextApproval != username {
		c.AbortWithError(http.StatusNotFound, errors.New("Invalid approver"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Invalid approver",
		})
		return
	}

	var capexAppr []CapexAppr
	err = db.Where("capex_id = ?", capexID).Order("seq").Find(&capexAppr).Error
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
			if approver.Approver != username {
				c.AbortWithError(http.StatusNotFound, errors.New("Not authorized to reject"))
				c.JSON(http.StatusNotFound, gin.H{
					"message": "Not authorized to reject",
				})
				return
			}
			// idx = i
			appr = approver
			break
		} else {
			if approver.Approver == username {
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

func updateUser(c *gin.Context) {
	id := c.MustGet("ID").(float64)
	if id == 0 {
		c.AbortWithError(http.StatusNotFound, errors.New("User unknown"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "User unknown",
		})
		return
	}

	updatePassBody := struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}{}

	c.BindJSON(&updatePassBody)

	var user User
	if err := db.Where("id = ?", id).First(&user).Error; err != nil {
		c.AbortWithError(http.StatusNotFound, errors.New("UserID not exists"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "UserID not exists",
		})
		return
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(updatePassBody.CurrentPassword))
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, errors.New("Incorrect Password"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Incorrect Password",
		})
		return
	}

	if updatePassBody.NewPassword != "" {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(updatePassBody.NewPassword), bcrypt.DefaultCost)
		user.Password = string(hashedPassword)
	}

	err = db.Save(&user).Error
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User Profile Updated",
	})
	return
}

func getUser(c *gin.Context) {
	id := c.MustGet("ID").(float64)

	userID := c.Param("id")
	if id == 0 {
		c.AbortWithError(http.StatusNotFound, errors.New("User unknown"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "User unknown",
		})
		return
	}

	if strconv.Itoa(int(id)) != userID {
		c.AbortWithError(http.StatusForbidden, errors.New("Unauthorized to view this profile"))
		c.JSON(http.StatusForbidden, gin.H{
			"message": "Unauthorized to view this profile",
		})
		return
	}

	var user User

	if err := db.Where("id = ?", id).First(&user).Error; err != nil {
		c.AbortWithError(http.StatusNotFound, errors.New("UserID not exists"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "UserID not exists",
		})
		return
	}

	user.Password = ""
	user.CreatedAt = time.Time{}
	user.UpdatedAt = time.Time{}
	user.DeletedAt = &time.Time{}

	c.JSON(http.StatusOK, &user)
	return
}

func createUser(c *gin.Context) {
	var user User
	var currentUser User
	c.BindJSON(&user)
	if err := db.Where("username = ?", user.Username).First(&currentUser).Error; err == nil {
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
	return
}

func login(c *gin.Context) {

	auth := strings.SplitN(c.Request.Header.Get("Authorization"), " ", 2)
	if auth[0] != "Basic" {
		c.AbortWithError(http.StatusUnauthorized, errors.New("Unauthorized"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Unauthorized",
		})
	}

	payload, _ := base64.StdEncoding.DecodeString(auth[1])
	pair := strings.SplitN(string(payload), ":", 2)

	var username = pair[0]
	var password = pair[1]

	// loginBody := struct {
	// 	Username string `json:"username"`
	// 	Password string `json:"password"`
	// }{}

	// c.BindJSON(&loginBody)

	var user User
	if err := db.Where("username = ?", username).First(&user).Error; err != nil {
		c.AbortWithError(http.StatusNotFound, errors.New("Username not found"))
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Username not found",
		})
		return
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
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
		Email    string `json:"email"`
		Token    string `json:"token"`
	}{}

	respondBodyLogin.ID = user.ID
	respondBodyLogin.Name = user.Name
	respondBodyLogin.Username = user.Username
	respondBodyLogin.Email = user.Email
	respondBodyLogin.Token, err = token.SignedString([]byte(signatureKey))
	if err != nil {
		c.AbortWithError(http.StatusNotFound, err)
		c.JSON(http.StatusNotFound, gin.H{
			"message": err,
		})
		return
	}

	c.JSON(200, respondBodyLogin)
	return
}

func notifApprover(trxID uint, approver string, sender string) {
	to := []string{}
	subject := "Capex " + strconv.Itoa(int(trxID))
	var user User
	_ = db.Where("username = ?", approver).First(&user).Error
	if user.ID == 0 {
		return
	}

	// to = append(to, user.Email)
	to = append(to, user.Email)

	user = User{}
	_ = db.Where("username = ?", sender).First(&user).Error

	var capexTrx CapexTrx
	_ = db.Where("ID = ?", trxID).First(&capexTrx).Error

	budget := struct {
		BudgetAmount int64
		Remaining    int64
		BudgetDesc   string
	}{}

	db.Table("tb_budget").
		Select("budget_amount, remaining, budget_desc").
		Where("budget_code = ?", capexTrx.BudgetApprovalCode).
		First(&budget)

	notification.SendEmail(to, subject, "approval.html", struct {
		CapexID         string
		Sender          string
		BudgetCode      string
		BudgetAmount    string
		CapexAmount     string
		BudgetAvailable string
		BudgetDesc      string
		Domain          string
	}{
		CapexID:         strconv.Itoa(int(trxID)),
		Sender:          user.Name,
		BudgetCode:      capexTrx.BudgetApprovalCode,
		BudgetAmount:    humanize.FormatInteger("#.###,", int(budget.BudgetAmount)),
		CapexAmount:     humanize.FormatInteger("#.###,", int(capexTrx.TotalAmount)),
		BudgetAvailable: humanize.FormatInteger("#.###,", int(budget.Remaining)),
		BudgetDesc:      budget.BudgetDesc,
		Domain:          os.Getenv("domain"),
	})

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

	notification.SendEmail(to, subject, "accounting-appr.html", struct {
		Name    string
		CapexID string
		Domain  string
	}{
		Name:    "Accounting Team",
		CapexID: strconv.Itoa(int(trxID)),
		Domain:  os.Getenv("domain"),
	})
}

func notifReject(trxID uint, message string) {

	user := struct {
		Email string
		Name  string
	}{}
	_ = db.Table("capex_trx as c").
		Select("u.email, u.name").
		Joins("JOIN user as u on c.created_by = u.id").
		Where("c.id = ?", trxID).
		Find(&user).
		Error

	to := []string{user.Email}
	subject := "Reject Capex " + strconv.Itoa(int(trxID))

	notification.SendEmail(to, subject, "reject-capex.html", struct {
		Name    string
		CapexID string
		Message string
		Domain  string
	}{
		Name:    user.Name,
		CapexID: strconv.Itoa(int(trxID)),
		Message: message,
		Domain:  os.Getenv("domain"),
	})

}

func notifFullApprove(trxID uint) {

	user := struct {
		Email string
		Name  string
	}{}
	_ = db.Table("capex_trx as c").
		Select("u.email, u.name").
		Joins("JOIN user as u on c.created_by = u.username").
		Where("c.id = ?", trxID).
		Find(&user).
		Error

	to := []string{user.Email}
	subject := "Capex " + strconv.Itoa(int(trxID)) + " Full Approved"

	notification.SendEmail(to, subject, "full-approve.html", struct {
		Name    string
		CapexID string
		Domain  string
	}{
		Name:    user.Name,
		CapexID: strconv.Itoa(int(trxID)),
		Domain:  os.Getenv("domain"),
	})

}

func exportToCSV(trx CapexTrx) error {
	contents := [][]string{
		{
			"ID",
			"Description",
			"Serial Number",
			"Quantity",
			"UoM",
			"Cost Center",
			"Activity Type",
			"Asset Group",
			"Asset Generation Mode",
		},
	}

	content := []string{
		strconv.Itoa(int(trx.ID)),
		trx.Description,
		trx.SerialNumber,
		strconv.Itoa(int(trx.Quantity)),
		trx.Uom,
		trx.CostCenter,
		trx.AssetActivityType,
		trx.AssetGroup,
		trx.AssetGenMode,
	}
	contents = append(contents, content)

	filename := fmt.Sprintf("%s-%s.csv", strconv.Itoa(int(trx.ID)), time.Now().Format("02012006150405.000"))

	err := export.SaveCSV(filename, contents)
	if err != nil {
		return err
	}

	return nil
}
