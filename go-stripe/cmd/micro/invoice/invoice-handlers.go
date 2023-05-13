package main

import (
	"fmt"
	"net/http"

	common_models "github.com/AlexL70/IntermediateWebAppWithGo/go-stripe/internal/common"
	"github.com/phpdave11/gofpdf"
	"github.com/phpdave11/gofpdf/contrib/gofpdi"
)

func (app *application) CreateAndSendInvoice(w http.ResponseWriter, r *http.Request) {
	// receive json
	var order common_models.Order
	err := app.readJSON(w, r, &order)
	if err != nil {
		app.errorLog.Println(err)
		app.BadRequest(w, r, err)
		return
	}
	// generate a pdf invoice
	err = app.createInvoicePDF(order)
	if err != nil {
		app.errorLog.Println(err)
		app.BadRequest(w, r, err)
		return
	}
	// create mail
	attachments := []string{fmt.Sprintf("./invoices/%d.pdf", order.ID)}
	// send the mail with an attachment
	err = app.SendMail("info@widgets.com", order.Email, "Your invoice", "invoice", attachments, nil)
	if err != nil {
		app.errorLog.Println(err)
		app.BadRequest(w, r, err)
		return
	}
	// send response
	resp := responsePayload{
		Error:   false,
		Message: fmt.Sprintf("Invoice %d.pdf has been created and sent to %s.", order.ID, order.Email),
	}
	err = app.writeJson(w, http.StatusOK, resp)
	if err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) createInvoicePDF(order common_models.Order) error {
	pdf := gofpdf.New("P", "mm", "Letter", "")
	pdf.SetMargins(10, 13, 10)
	pdf.SetAutoPageBreak(true, 0)

	importer := gofpdi.NewImporter()
	t := importer.ImportPage(pdf, "./pdf-templates/invoice.pdf", 1, "/MediaBox")
	pdf.AddPage()
	importer.UseImportedTemplate(pdf, t, 0, 0, 215.9, 0)
	// write info
	pdf.SetY(50)
	pdf.SetX(10)
	pdf.SetFont("Times", "", 11)
	pdf.CellFormat(97, 8, fmt.Sprintf("Attention: %s %s", order.FirstName, order.LastName), "", 0, "L", false, 0, "")
	pdf.Ln(5)
	pdf.CellFormat(97, 8, order.Email, "", 0, "L", false, 0, "")
	pdf.Ln(5)
	pdf.CellFormat(97, 8, order.CreatedAt.Format("2006-01-02"), "", 0, "L", false, 0, "")

	pdf.SetY(93)
	pdf.SetX(58)
	pdf.CellFormat(155, 8, order.Product, "", 0, "L", false, 0, "")
	pdf.SetX(166)
	pdf.CellFormat(20, 8, fmt.Sprintf("%d", order.Quantity), "", 0, "C", false, 0, "")
	pdf.SetX(185)
	pdf.CellFormat(20, 8, fmt.Sprintf("$%.2f", float32(order.Amount/100.0)), "", 0, "R", false, 0, "")

	invoicePath := fmt.Sprintf("./invoices/%d.pdf", order.ID)
	err := pdf.OutputFileAndClose(invoicePath)
	return err
}
