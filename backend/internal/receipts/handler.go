package receipts

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"

	"expenselens/backend/internal/server"
)

// Handler exposes /receipts.
type Handler struct {
	svc    *Service
	userID func(context.Context) bson.ObjectID
}

// NewHandler constructs the handler.
func NewHandler(svc *Service, userID func(context.Context) bson.ObjectID) *Handler {
	return &Handler{svc: svc, userID: userID}
}

// Mount registers routes (caller applies auth middleware).
func (h *Handler) Mount(r chi.Router) {
	r.Route("/receipts", func(r chi.Router) {
		r.Post("/upload", h.upload)
		r.Get("/", h.list)
		r.Get("/recent-uploads", h.recent)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.get)
			r.Patch("/", h.update)
			r.Delete("/", h.delete)
			r.Post("/confirm", h.confirm)
			r.Post("/reextract", h.reextract)
			r.Post("/images", h.addImages)
			r.Delete("/images/{imageId}", h.removeImage)
		})
	})
}

func (h *Handler) id(r *http.Request) (bson.ObjectID, error) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		return bson.ObjectID{}, server.ErrNotFound
	}
	return id, nil
}

// readMultipart collects files from the "files" field (also accepts "file").
func (h *Handler) readMultipart(r *http.Request) ([]UploadFile, error) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return nil, server.BadRequest("expected multipart form: " + err.Error())
	}
	var headers []*multipart.FileHeader
	headers = append(headers, r.MultipartForm.File["files"]...)
	headers = append(headers, r.MultipartForm.File["file"]...)
	if len(headers) == 0 {
		return nil, server.BadRequest("no files in request (use field name \"files\")")
	}
	if len(headers) > 10 {
		return nil, server.BadRequest("at most 10 files per upload")
	}
	var out []UploadFile
	for _, fh := range headers {
		f, err := fh.Open()
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(io.LimitReader(f, h.svc.maxBytes+1))
		f.Close()
		if err != nil {
			return nil, err
		}
		ct := http.DetectContentType(data)
		if ct == "application/octet-stream" {
			ct = fh.Header.Get("Content-Type")
		}
		out = append(out, UploadFile{Name: fh.Filename, ContentType: ct, Data: data})
	}
	return out, nil
}

func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
	files, err := h.readMultipart(r)
	if err != nil {
		server.Error(w, r, err)
		return
	}
	group := r.FormValue("group") == "true" || r.FormValue("group") == "1"
	res, err := h.svc.Upload(r.Context(), h.userID(r.Context()), files, group)
	if err != nil {
		server.Error(w, r, err)
		return
	}
	server.JSON(w, http.StatusAccepted, map[string]any{"receipts": res})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := ListFilter{
		Query: q.Get("q"), Status: q.Get("status"), From: q.Get("from"), To: q.Get("to"), Sort: q.Get("sort"),
	}
	f.Page, _ = strconv.Atoi(q.Get("page"))
	f.PageSize, _ = strconv.Atoi(q.Get("pageSize"))
	if c := q.Get("category"); c != "" {
		cid, err := bson.ObjectIDFromHex(c)
		if err != nil {
			server.Error(w, r, server.BadRequest("invalid category id"))
			return
		}
		f.CategoryID = &cid
	}
	rows, total, err := h.svc.List(r.Context(), h.userID(r.Context()), f)
	if err != nil {
		server.Error(w, r, err)
		return
	}
	if f.PageSize <= 0 || f.PageSize > 100 {
		f.PageSize = 20
	}
	if f.Page < 1 {
		f.Page = 1
	}
	server.JSON(w, http.StatusOK, map[string]any{
		"receipts": rows, "total": total, "page": f.Page, "pageSize": f.PageSize,
	})
}

func (h *Handler) recent(w http.ResponseWriter, r *http.Request) {
	n, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if n <= 0 || n > 50 {
		n = 10
	}
	rows, err := h.svc.Recent(r.Context(), h.userID(r.Context()), n)
	if err != nil {
		server.Error(w, r, err)
		return
	}
	server.JSON(w, http.StatusOK, map[string]any{"receipts": rows})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := h.id(r)
	if err != nil {
		server.Error(w, r, err)
		return
	}
	rc, err := h.svc.Get(r.Context(), h.userID(r.Context()), id)
	if err != nil {
		server.Error(w, r, err)
		return
	}
	server.JSON(w, http.StatusOK, rc)
}

func (h *Handler) decodePatch(r *http.Request) (Patch, error) {
	var p Patch
	if r.ContentLength == 0 {
		return p, nil
	}
	if err := server.Decode(r, &p); err != nil {
		return p, err
	}
	if err := server.Validate(p); err != nil {
		return p, err
	}
	return p, nil
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, err := h.id(r)
	if err != nil {
		server.Error(w, r, err)
		return
	}
	p, err := h.decodePatch(r)
	if err != nil {
		server.Error(w, r, err)
		return
	}
	rc, err := h.svc.Update(r.Context(), h.userID(r.Context()), id, p)
	if err != nil {
		server.Error(w, r, err)
		return
	}
	server.JSON(w, http.StatusOK, rc)
}

func (h *Handler) confirm(w http.ResponseWriter, r *http.Request) {
	id, err := h.id(r)
	if err != nil {
		server.Error(w, r, err)
		return
	}
	p, err := h.decodePatch(r)
	if err != nil {
		server.Error(w, r, err)
		return
	}
	rc, err := h.svc.Confirm(r.Context(), h.userID(r.Context()), id, p)
	if err != nil {
		server.Error(w, r, err)
		return
	}
	server.JSON(w, http.StatusOK, rc)
}

func (h *Handler) reextract(w http.ResponseWriter, r *http.Request) {
	id, err := h.id(r)
	if err != nil {
		server.Error(w, r, err)
		return
	}
	rc, err := h.svc.Reextract(r.Context(), h.userID(r.Context()), id)
	if err != nil {
		server.Error(w, r, err)
		return
	}
	server.JSON(w, http.StatusAccepted, rc)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := h.id(r)
	if err != nil {
		server.Error(w, r, err)
		return
	}
	if err := h.svc.Delete(r.Context(), h.userID(r.Context()), id); err != nil {
		server.Error(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) addImages(w http.ResponseWriter, r *http.Request) {
	id, err := h.id(r)
	if err != nil {
		server.Error(w, r, err)
		return
	}
	files, err := h.readMultipart(r)
	if err != nil {
		server.Error(w, r, err)
		return
	}
	rc, err := h.svc.AddImages(r.Context(), h.userID(r.Context()), id, files)
	if err != nil {
		server.Error(w, r, err)
		return
	}
	server.JSON(w, http.StatusOK, rc)
}

func (h *Handler) removeImage(w http.ResponseWriter, r *http.Request) {
	id, err := h.id(r)
	if err != nil {
		server.Error(w, r, err)
		return
	}
	rc, err := h.svc.RemoveImage(r.Context(), h.userID(r.Context()), id, chi.URLParam(r, "imageId"))
	if err != nil {
		server.Error(w, r, err)
		return
	}
	server.JSON(w, http.StatusOK, rc)
}
