package extraction

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Gemini calls the Generative Language REST API with a JSON response schema.
type Gemini struct {
	apiKey string
	model  string
	base   string
	http   *http.Client
}

// NewGemini constructs the client. model is e.g. "gemini-3.1-pro-preview".
func NewGemini(apiKey, model string) *Gemini {
	return &Gemini{
		apiKey: apiKey, model: model,
		base: "https://generativelanguage.googleapis.com/v1beta",
		http: &http.Client{Timeout: 120 * time.Second},
	}
}

func (g *Gemini) Name() string { return "gemini:" + g.model }

// wire is the shape we ask the model for. Amounts are decimal numbers in the
// receipt's currency; we convert to minor units afterwards.
type wire struct {
	Merchant      string   `json:"merchant"`
	Date          string   `json:"date"`
	Time          *string  `json:"time"`
	Currency      string   `json:"currency"`
	Subtotal      *float64 `json:"subtotal"`
	Tax           *float64 `json:"tax"`
	Total         float64  `json:"total"`
	PaymentMethod *string  `json:"paymentMethod"`
	GSTIN         *string  `json:"gstin"`
	TaxBreakdown  []struct {
		Label  string  `json:"label"`
		Amount float64 `json:"amount"`
	} `json:"taxBreakdown"`
	Items []struct {
		Name      string   `json:"name"`
		Quantity  *float64 `json:"quantity"`
		UnitPrice *float64 `json:"unitPrice"`
		Amount    float64  `json:"amount"`
	} `json:"items"`
	SuggestedCategory  string `json:"suggestedCategory"`
	WarrantyCandidates []struct {
		ItemName string `json:"itemName"`
		Reason   string `json:"reason"`
	} `json:"warrantyCandidates"`
	Confidence float64 `json:"confidence"`
}

func responseSchema(categories []string) map[string]any {
	num := map[string]any{"type": "number", "nullable": true}
	str := map[string]any{"type": "string", "nullable": true}
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"merchant":      map[string]any{"type": "string"},
			"date":          map[string]any{"type": "string", "description": "YYYY-MM-DD"},
			"time":          str,
			"currency":      map[string]any{"type": "string", "description": "ISO 4217, e.g. INR"},
			"subtotal":      num,
			"tax":           num,
			"total":         map[string]any{"type": "number"},
			"paymentMethod": map[string]any{"type": "string", "nullable": true, "enum": []string{"cash", "upi", "card", "netbanking", "wallet", "other"}},
			"gstin":         str,
			"taxBreakdown": map[string]any{"type": "array", "items": map[string]any{
				"type": "object", "properties": map[string]any{"label": map[string]any{"type": "string"}, "amount": map[string]any{"type": "number"}},
				"required": []string{"label", "amount"},
			}},
			"items": map[string]any{"type": "array", "items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string"}, "quantity": num, "unitPrice": num, "amount": map[string]any{"type": "number"},
				},
				"required": []string{"name", "amount"},
			}},
			"suggestedCategory": map[string]any{"type": "string", "enum": categories},
			"warrantyCandidates": map[string]any{"type": "array", "items": map[string]any{
				"type": "object", "properties": map[string]any{"itemName": map[string]any{"type": "string"}, "reason": map[string]any{"type": "string"}},
				"required": []string{"itemName", "reason"},
			}},
			"confidence": map[string]any{"type": "number"},
		},
		"required": []string{"merchant", "date", "currency", "total", "items", "suggestedCategory", "confidence"},
	}
}

func prompt(in Input) string {
	return fmt.Sprintf(`You are an expert at reading purchase receipts and invoices, especially Indian ones (₹, GST/CGST/SGST/IGST, GSTIN, UPI, Paytm, PhonePe).
Extract the fields from the attached receipt file(s). All files belong to ONE receipt (multiple pages or photos).

Rules:
- Amounts are plain decimal numbers in the receipt's currency (e.g. 1230.50), no symbols.
- "total" is the final amount paid after taxes and discounts.
- "date" must be YYYY-MM-DD. If the year is missing, assume the most recent plausible year. Today is %s (%s).
- "items": one entry per line item with the line amount. Skip subtotal/tax/discount/total lines.
- "suggestedCategory" must be one of: %s.
- "warrantyCandidates": items that typically carry a manufacturer warranty (electronics, appliances, tools, furniture). Empty array if none.
- "confidence": 0 to 1, how sure you are about merchant/date/total.
- If a value is not on the receipt, use null. Never invent values.
- Default currency hint: %s.`,
		time.Now().Format("2006-01-02"), in.Timezone, strings.Join(in.Categories, ", "), in.Currency)
}

// Extract sends the files inline. Inline requests must stay under ~20MB;
// receipts uploads are capped below that by MAX_UPLOAD_MB.
func (g *Gemini) Extract(ctx context.Context, in Input) (*Result, error) {
	if len(in.Files) == 0 {
		return nil, errors.New("extraction: no files")
	}
	parts := []map[string]any{{"text": prompt(in)}}
	for _, f := range in.Files {
		parts = append(parts, map[string]any{"inline_data": map[string]any{
			"mime_type": f.ContentType, "data": base64.StdEncoding.EncodeToString(f.Data),
		}})
	}
	body := map[string]any{
		"contents": []map[string]any{{"role": "user", "parts": parts}},
		"generationConfig": map[string]any{
			"temperature":      0.1,
			"responseMimeType": "application/json",
			"responseSchema":   responseSchema(in.Categories),
		},
	}
	buf, _ := json.Marshal(body)
	url := fmt.Sprintf("%s/models/%s:generateContent", g.base, g.model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", g.apiKey)
	resp, err := g.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("extraction: gemini request: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("extraction: gemini status %d: %s", resp.StatusCode, truncate(raw, 400))
	}
	var out struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		PromptFeedback *struct {
			BlockReason string `json:"blockReason"`
		} `json:"promptFeedback"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("extraction: gemini response: %w", err)
	}
	if out.PromptFeedback != nil && out.PromptFeedback.BlockReason != "" {
		return nil, fmt.Errorf("extraction: gemini blocked: %s", out.PromptFeedback.BlockReason)
	}
	if len(out.Candidates) == 0 || len(out.Candidates[0].Content.Parts) == 0 {
		return nil, errors.New("extraction: gemini returned no candidates")
	}
	var text strings.Builder
	for _, p := range out.Candidates[0].Content.Parts {
		text.WriteString(p.Text)
	}
	return parseWire([]byte(text.String()), g.Name())
}

// parseWire converts the model JSON into a Result with minor-unit amounts.
func parseWire(data []byte, model string) (*Result, error) {
	var w wire
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, fmt.Errorf("extraction: model JSON: %w", err)
	}
	r := &Result{
		Merchant: strings.TrimSpace(w.Merchant), Date: w.Date, Currency: strings.ToUpper(w.Currency),
		Total: toMinor(w.Total), SuggestedCategory: w.SuggestedCategory, Confidence: w.Confidence,
		Model: model, Raw: json.RawMessage(data), Items: []Item{},
	}
	if w.Time != nil {
		r.Time = *w.Time
	}
	if w.Subtotal != nil {
		v := toMinor(*w.Subtotal)
		r.Subtotal = &v
	}
	if w.Tax != nil {
		v := toMinor(*w.Tax)
		r.Tax = &v
	}
	if w.PaymentMethod != nil {
		r.PaymentMethod = *w.PaymentMethod
	}
	if w.GSTIN != nil {
		r.GSTIN = strings.ToUpper(strings.TrimSpace(*w.GSTIN))
	}
	for _, t := range w.TaxBreakdown {
		r.TaxBreakdown = append(r.TaxBreakdown, TaxLine{Label: t.Label, Amount: toMinor(t.Amount)})
	}
	for _, it := range w.Items {
		item := Item{Name: strings.TrimSpace(it.Name), Quantity: it.Quantity, Amount: toMinor(it.Amount)}
		if it.UnitPrice != nil {
			v := toMinor(*it.UnitPrice)
			item.UnitPrice = &v
		}
		r.Items = append(r.Items, item)
	}
	for _, wc := range w.WarrantyCandidates {
		r.WarrantyCandidates = append(r.WarrantyCandidates, WarrantyCandidate{ItemName: wc.ItemName, Reason: wc.Reason})
	}
	if r.Currency == "" {
		r.Currency = "INR"
	}
	return r, nil
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "…"
}
