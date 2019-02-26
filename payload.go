package main

type Approval struct {
	Status string `json:"Status"`
	Date   string `json:"Date,omitempty"`
}

type Approvals struct {
	RAApproval Approval `json:"RAApproval,omitempty"`
	FPApproval Approval `json:"FPApproval,omitempty"`
}

type FPInfo struct {
	FPName       string `json:"FPName"`
	FPCode       string `json:"FPCode"`
	FPProgram    string `json:"FPProgram"`
	CurrInvstAmt string `json:"CurrInvstAmt,omitempty"`
	Currency     string `json:"Currency,omitempty"`
}

type EmpData struct {
	ID         string `json:"ID"`
	FirstName  string `json:"FirstName"`
	MiddleName string `json:"MiddleName"`
	LastName   string `json:"LastName"`
	Birthday   string `json:"Birthday"`
	Age        string `json:"Age"`
	Address    string `json:"Address"`
	Email      string `json:"Email"`
	ContactNo  string `json:"ContactNo"`
	TaxID      string `json:"TaxID"`
	Employer   string `json:"Employer"`
}

//Private Data
type Employee struct {
	ObjectType       string    `json:"docType"`
	TxID             string    `json:"TxID"` //most recent Transaction ID
	EmpData          EmpData   `json:"EmpData"`
	FPInfo           FPInfo    `json:"FPInfo"`
	EnrollmentStatus string    `json:"EnrollmentStatus"`
	NewFPInfo        FPInfo    `json:"NewFPInfo,omitempty"`
	Approvals        Approvals `json:"Approvals,omitempty"`
}

//Payload from UI
type EnrollmentReq struct {
	EmployeeData EmpData `json:"EmployeeData"`
	FPInfo       FPInfo  `json:"FPInfo"`
}

//Private Data
//Payload to/from UI
type EnrollmentApproval struct {
	EmployeeData EmpData   `json:"EmployeeData,omitempty"`
	FPInfo       FPInfo    `json:"FPInfo,omitempty"`
	Approvals    Approvals `json:"Approvals"`
}

//Payload from UI
type ChangeFPReq struct {
	EmpID      string `json:"EmpID"`
	CurrFPCode string `json:"CurrFPCode"`
	NewFPInfo  FPInfo `json:"NewFPInfo"`
}

//Private Data
//Payload to/from UI
type ChangeFPApproval struct {
	EmployeeData string    `json:"EmployeeData"`
	FPInfo       FPInfo    `json:"FPInfo"`
	NewFPInfo    FPInfo    `json:"NewFPInfo"`
	Approvals    Approvals `json:"Approvals"`
}

//Payload contents
type Payload struct {
	EnrollmentReq      *EnrollmentReq      `json:"EnrollmentReq,omitempty"`
	EnrollmentApproval *EnrollmentApproval `json:"EnrollmentApproval,omitempty"`
	ChangeFPReq        *ChangeFPReq        `json:"ChangeFPReq,omitempty"`
	ChangeFPApproval   *ChangeFPApproval   `json:"ChangeFPApproval,omitempty"`
	Employee           *Employee           `json:"Employee,omitempty"`
	Block              *string             `json:"Block,omitempty"`
	Employees          *[]Employee         `json:"Employees,omitempty"`
}
