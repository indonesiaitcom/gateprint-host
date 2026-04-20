package entity

type PrintCommand struct {
	Action string      `json:"action"`
	Value  interface{} `json:"value"`
}

type PrintJob struct {
	PrinterName string         `json:"printer_name"`
	Commands    []PrintCommand `json:"commands"`
}
