package models

type Collections struct {
	Transaction   string
	Associate     string
	Loan          string
	Miscellaneous string
	Customer      string
}

var Collection = Collections{
	Transaction:   "transaction",
	Associate:     "associate",
	Loan:          "loan",
	Miscellaneous: "miscellaneous",
	Customer:      "customer",
}
