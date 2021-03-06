package main

import (
	"time"

	"github.com/jinzhu/gorm"
)

type CapexTrx struct {
	gorm.Model
	RequestorPosition string `json:"requestorPosition"`
	Year              int    `json:"year"`
	BudgetOwner       string `json:"budgetOwner"`
	CostCenter        string `json:"costCenter"`
	Purpose           string `json:"purpose"`
	BudgetType        string `json:"budgetType"`
	Description       string `json:"description"`
	SerialNumber      string `json:"serialNumber"`
	Quantity          uint64 `json:"quantity"`
	Uom               string `json:"uom"`
	DeliveryDate      string `gorm:"type:date" json:"deliveryDate"`
	Justification     string `json:"justification"`
	UnitPrice         uint64 `json:"unitPrice"`
	TotalAmount       uint64 `json:"totalAmount"`
	TotalBudget       int64  `json:"totalBudget"`
	ActualAmount      uint64 `json:"actualAmount"`
	ForeignAmount     uint64 `json:"foreignAmount"`
	ForeignCurrency   string `json:"foreignCurrency"`
	Plant             string `json:"plant"`
	StorageLocation   string `json:"storageLocation"`
	CreatedBy         string `json:"createdBy"`
	NextApproval      string `json:"nextApproval"`
	Status            string `json:"status"`
	ACCApproved       string `json:"ACCApproved"`
	AssetClass        string `json:"assetClass"`
	AssetActivityType string `json:"assetActivityType"`
	AssetGroup        string `json:"assetGroup"`
	AssetGenMode      string `json:"assetGenMode"`
	AssetNote         string `json:"assetNote"`
	Switched          bool   `json:"switched"`
}

type UserRole struct {
	Username string `gorm:"primary_key;auto_increment:false" json:"username"`
	Role     string `gorm:"primary_key;type:ENUM('CREATOR','VIEWER','APPROVER','ACCAPPROVER')" json:"role"`
}

type Approval struct {
	CostCenter string  `gorm:"primary_key;auto_increment:false"`
	AssetClass string  `gorm:"primary_key;auto_increment:false"`
	AmountLow  float64 `gorm:"primary_key"`
	Seq        uint    `gorm:"primary_key;auto_increment:false"`
	AmountHigh float64
	Approver   string
}

type CapexAppr struct {
	CapexID   uint   `gorm:"primary_key;auto_increment:false" json:"capexID"`
	Seq       uint   `gorm:"primary_key;auto_increment:false" json:"seq"`
	Approver  string `json:"approver"`
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

type CapexBudget struct {
	CapexID      uint   `gorm:"primary_key:auto_increment:false" json:"capexID"`
	BudgetCode   string `gorm:"primary_key" json:"budgetCode"`
	CostCenter   string `json:"costCenter"`
	Amount       uint64 `json:"amount"`
	Remaining    int64  `json:"remaining"`
	ActualAmount uint64 `json:"actualAmount"`
	MainBudget   bool   `json:"mainBudget"`
}
type CapexAttachment struct {
	CapexID   uint64 `gorm:"primary_key;auto_increment:false" json:"capexId"`
	Filename  string `gorm:"primary_key" json:"filename"`
	Category  string `json:"category"`
	CreatedAt time.Time
}

type CapexMessage struct {
	gorm.Model
	CapexID      uint   `json:"capexId"`
	FromUsername string `json:"fromUsername"`
	ToUsername   string `json:"toUsername"`
	Message      string `json:"message"`
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

type CostCenterRole struct {
	Role       string `json:"role"`
	CostCenter string `json:"costCenter"`
}

type UserCostCenterRole struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}
