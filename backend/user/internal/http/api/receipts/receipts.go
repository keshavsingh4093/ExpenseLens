// Package receipts exposes /api/receipts.
package receipts

import (
	"io"
	"mime/multipart"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"

	gohttp "smartreceipt/backend/common/pkg/http"
	"smartreceipt/backend/common/pkg/validate"
	receiptrepo "smartreceipt/backend/common/services/receipt"

	"smartreceipt/backend/user/internal/pkg/middleware"
	"smartreceipt/backend/user/internal/services/receipt"
)

var svc = receipt.New()

// GetRouteSpecs returns the receipt routes. Multipart routes get a longer timeout.
func GetRouteSpecs() []gohttp.RouteSpec {
	return []gohttp.RouteSpec{
		{Method: "POST", Path: "/api/receipts/upload", Handler: handleUpload, TimeOut: 120},
		{Method: "GET", Path: "/api/receipts", Handler: handleList},
		{Method: "GET", Path: "/api/receipts/recent-uploads", Handler: handleRecent},
		{Method: "GET", Path: "/api/receipts/{id}", Handler: handleGet},
		{Method: "PATCH", Path: "/api/receipts/{id}", Handler: handleUpdate},
		{Method: "DELETE", Path: "/api/receipts/{id}", Handler: handleDelete},
		{Method: "POST", Path: "/api/receipts/{id}/confirm", Handler: handleConfirm},
		{Method: "POST", Path: "/api/receipts/{id}/reextract", Handler: handleReextract},
		{Method: "POST", Path: "/api/receipts/{id}/images", Handler: handleAddImages, TimeOut: 120},
		{Method: "DELETE", Path: "/api/receipts/{id}/images/{imageId}", Handler: handleRemoveImage},
	}
}

// readMultipart collects files from the "files" field (also accepts "file").
func readMultipart(r *http.Request) ([]receipt.UploadFile, error) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return nil, gohttp.ErrBadRequest("expected multipart form: " + err.Error())
	}
	var headers []*multipart.FileHeader
	headers = append(headers, r.MultipartForm.File["files"]...)
	headers = append(headers, r.MultipartForm.File["file"]...)
	if len(headers) == 0 {
		return nil, gohttp.ErrBadRequest("no files in request (use field name \"files\")")
	}
	if len(headers) > 10 {
		return nil, gohttp.ErrBadRequest("at most 10 files per upload")
	}
	var out []receipt.UploadFile
	for _, fh := range headers {
		f, err := fh.Open()
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(io.LimitReader(f, svc.MaxBytes()+1))
		f.Close()
		if err != nil {
			return nil, err
		}
		ct := http.DetectContentType(data)
		if ct == "application/octet-stream" {
			ct = fh.Header.Get("Content-Type")
		}
		out = append(out, receipt.UploadFile{Name: fh.Filename, ContentType: ct, Data: data})
	}
	return out, nil
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	files, err := readMultipart(r)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	group := r.FormValue("group") == "true" || r.FormValue("group") == "1"
	res, err := svc.Upload(r.Context(), middleware.UserID(r.Context()), files, group)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Accepted(w, map[string]any{"receipts": res})
}

func handleList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := receiptrepo.ListFilter{
		Query: q.Get("q"), Status: q.Get("status"), From: q.Get("from"), To: q.Get("to"), Sort: q.Get("sort"),
	}
	f.Page, _ = strconv.Atoi(q.Get("page"))
	f.PageSize, _ = strconv.Atoi(q.Get("pageSize"))
	if c := q.Get("category"); c != "" {
		cid, err := bson.ObjectIDFromHex(c)
		if err != nil {
			gohttp.BadRequest(w, "invalid category id")
			return
		}
		f.CategoryID = &cid
	}
	rows, total, err := svc.List(r.Context(), middleware.UserID(r.Context()), f)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	f.Normalize()
	gohttp.Success(w, map[string]any{"receipts": rows, "total": total, "page": f.Page, "pageSize": f.PageSize})
}

func handleRecent(w http.ResponseWriter, r *http.Request) {
	n, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if n <= 0 || n > 50 {
		n = 10
	}
	rows, err := svc.Recent(r.Context(), middleware.UserID(r.Context()), n)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Success(w, map[string]any{"receipts": rows})
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := validate.ObjectID(chi.URLParam(r, "id"))
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	rc, err := svc.Get(r.Context(), middleware.UserID(r.Context()), id)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Success(w, rc)
}

func decodePatch(r *http.Request) (receipt.Patch, error) {
	var p receipt.Patch
	if r.ContentLength == 0 {
		return p, nil
	}
	if err := gohttp.Decode(r, &p); err != nil {
		return p, err
	}
	if err := validate.Struct(p); err != nil {
		return p, err
	}
	return p, nil
}

func handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := validate.ObjectID(chi.URLParam(r, "id"))
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	p, err := decodePatch(r)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	rc, err := svc.Update(r.Context(), middleware.UserID(r.Context()), id, p)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Success(w, rc)
}

func handleConfirm(w http.ResponseWriter, r *http.Request) {
	id, err := validate.ObjectID(chi.URLParam(r, "id"))
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	p, err := decodePatch(r)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	rc, err := svc.Confirm(r.Context(), middleware.UserID(r.Context()), id, p)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Success(w, rc)
}

func handleReextract(w http.ResponseWriter, r *http.Request) {
	id, err := validate.ObjectID(chi.URLParam(r, "id"))
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	rc, err := svc.Reextract(r.Context(), middleware.UserID(r.Context()), id)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Accepted(w, rc)
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := validate.ObjectID(chi.URLParam(r, "id"))
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	if err := svc.Delete(r.Context(), middleware.UserID(r.Context()), id); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.NoContent(w)
}

func handleAddImages(w http.ResponseWriter, r *http.Request) {
	id, err := validate.ObjectID(chi.URLParam(r, "id"))
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	files, err := readMultipart(r)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	rc, err := svc.AddImages(r.Context(), middleware.UserID(r.Context()), id, files)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Success(w, rc)
}

func handleRemoveImage(w http.ResponseWriter, r *http.Request) {
	id, err := validate.ObjectID(chi.URLParam(r, "id"))
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	rc, err := svc.RemoveImage(r.Context(), middleware.UserID(r.Context()), id, chi.URLParam(r, "imageId"))
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Success(w, rc)
}
