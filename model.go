package main

import (
	"time"

	"github.com/jinzhu/gorm"
)

type CapexTrx struct {
	gorm.Model
	RequestorPosition  string `json:"requestorPosition"`
	BudgetOwner        string `json:"budgetOwner"`
	CostCenter         string `json:"costCenter"`
	Purpose            string `json:"purpose"`
	BudgetType         string `sql:"type:ENUM('B','U')" json:"budgetType"`
	BudgetApprovalCode string `json:"budgetApprovalCode"`
	Description        string `json:"description"`
	SerialNumber       string `json:"serialNumber"`
	Quantity           uint64 `json:"quantity"`
	Uom                string `json:"uom"`
	DeliveryDate       string `gorm:"type:date" json:"deliveryDate"`
	Justification      string `json:"justification"`
	TotalAmount        uint64 `json:"totalAmount"`
	TotalBudget        uint64 `json:"totalBudget"`
	Plant              string `json:"plant"`
	StorageLocation    string `json:"storageLocation"`
	CreatedBy          uint   `json:"createdBy"`
	NextApproval       uint   `json:"nextApproval"`
	Status             string `json:"status"`
	ACCApproved        string `sql:"type:ENUM('X', '')" json:"ACCApproved"`
	AssetClass         string `json:"assetClass"`
	AssetActivityType  string `json:"assetActivityType"`
	AssetGroup         string `json:"assetGroup"`
	AssetGenMode       string `json:"assetGenMode"`
}

type UserRule struct {
	UserID uint   `gorm:"primary_key;auto_increment:false" json:"userID"`
	Rule   string `gorm:"primary_key;type:ENUM('CREATOR','VIEWER','APPROVER','ACCAPPROVER')" json:"rule"`
}

type Approval struct {
	CostCenter string  `gorm:"primary_key;auto_increment:false"`
	AssetType  string  `gorm:"primary_key;auto_increment:false"`
	Unbudgeted bool    `gorm:"primary_key;auto_increment:false"`
	AmountLow  float64 `gorm:"primary_key"`
	Seq        uint    `gorm:"primary_key;auto_increment:false"`
	AmountHigh float64
	Approver   uint
}

type CapexAppr struct {
	CapexID   uint   `gorm:"primary_key;auto_increment:false" json:"capexID"`
	Seq       uint   `gorm:"primary_key;auto_increment:false" json:"seq"`
	Approver  uint   `json:"approver"`
	Status    string `json:"status"`
	Remark    string `json:"remark"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CapexAsset struct {
	CapexID     uint   `gorm:"primary_key;auto_increment:false" json:"capexId"`
	CompanyCode string `gorm:"primary_key;auto_increment:false" json:"companyCode"`
	AssetNo     string `gorm:"primary_key;auto_increment:false" json:"assetNo"`
	AssetSubNo  string `gorm:"primary_key;auto_increment:false" json:"assetSubNo"`
	CreatedAt   time.Time
}

type Plant struct {
	gorm.Model
	PlantCode string `json:"plantCode"`
	PlantDesc string `json:"plantDesc"`
}

type Budget struct {
	BudgetCode   string
	Date         time.Time
	BudgetAmount uint64
	Percen       uint
}

type BudgetOwner struct {
	IdPernr  uint
	Name     string
	Position string
}

type Purpose struct {
	IdPurpose uint
	Desc      string
}

type User struct {
	gorm.Model
	Username  string `json:"username"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Password  string `json:"password"`
	Position  string `json:"position"`
	PayrollID string `json:"payrollID"`
}
