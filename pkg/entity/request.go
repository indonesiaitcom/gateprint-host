package entity

type ReceiptItem struct {
	Name     string `json:"name"`     // Nama Produk
	Qty      int    `json:"qty"`      // Jumlah beli
	Price    string `json:"price"`    // Harga satuan (format string agar mudah di print: "10.000")
	Subtotal string `json:"subtotal"` // Total per item
}

type ReceiptData struct {
	// Header Info
	StoreName string `json:"store_name"`
	Address   string `json:"address"`
	Phone     string `json:"phone"`
	Cashier   string `json:"cashier"`
	
	// Transaction Info
	TransactionID string        `json:"transaction_id"`
	Date          string        `json:"date"`
	Items         []ReceiptItem `json:"items"`
	
	// Payment Info
	Total   string `json:"total"`
	Payment string `json:"payment"` // Misal: "Tunai", "QRIS", "Debit"
	Change  string `json:"change"`  // Kembalian
	
	// Custom Footer/Marketing
	LogoPath string `json:"logo_path"` // Path absolut di laptop: "C:/assets/logo.png"
	QRData   string `json:"qr_data"`   // URL untuk cek resi atau QRIS statis
	Footer   string `json:"footer"`    // Pesan: "Barang yang sudah dibeli..."
}
