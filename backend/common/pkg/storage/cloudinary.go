package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

// Cloudinary stores files in a Cloudinary account. Keys become public IDs.
// Objects are uploaded as "upload" type with unguessable IDs; switch to
// "authenticated" delivery + signed URLs if stricter privacy is required.
type Cloudinary struct {
	cld *cloudinary.Cloudinary
}

// NewCloudinary parses a cloudinary://key:secret@cloud URL.
func NewCloudinary(rawURL string) (*Cloudinary, error) {
	if rawURL == "" {
		return nil, errors.New("storage: CLOUDINARY_URL is required for the cloudinary provider")
	}
	cld, err := cloudinary.NewFromURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("storage: cloudinary: %w", err)
	}
	cld.Config.URL.Secure = true
	return &Cloudinary{cld: cld}, nil
}

func (c *Cloudinary) Name() string { return "cloudinary" }

// publicID strips the extension: Cloudinary appends format itself.
func publicID(key string) string {
	if i := strings.LastIndex(key, "."); i > strings.LastIndex(key, "/") {
		return key[:i]
	}
	return key
}

func resourceType(contentType string) api.AssetType {
	if strings.HasPrefix(contentType, "image/") || contentType == "application/pdf" {
		return api.Image
	}
	return api.File
}

func (c *Cloudinary) Put(ctx context.Context, key string, r io.Reader, contentType string) (*Object, error) {
	if err := SafeKey(key); err != nil {
		return nil, err
	}
	res, err := c.cld.Upload.Upload(ctx, r, uploader.UploadParams{
		PublicID:       publicID(key),
		ResourceType:   string(resourceType(contentType)),
		Overwrite:      api.Bool(true),
		UniqueFilename: api.Bool(false),
		UseFilename:    api.Bool(false),
	})
	if err != nil {
		return nil, fmt.Errorf("storage: cloudinary upload: %w", err)
	}
	if res.Error.Message != "" {
		return nil, fmt.Errorf("storage: cloudinary upload: %s", res.Error.Message)
	}
	return &Object{Key: key, URL: res.SecureURL, ContentType: contentType, Size: int64(res.Bytes)}, nil
}

func (c *Cloudinary) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.URL(key), nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, "", ErrNotFound
	}
	if resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, "", fmt.Errorf("storage: cloudinary get: status %d", resp.StatusCode)
	}
	return resp.Body, resp.Header.Get("Content-Type"), nil
}

func (c *Cloudinary) Delete(ctx context.Context, key string) error {
	id := publicID(key)
	for _, rt := range []api.AssetType{api.Image, api.File} {
		res, err := c.cld.Upload.Destroy(ctx, uploader.DestroyParams{PublicID: id, ResourceType: string(rt)})
		if err != nil {
			return err
		}
		if res.Result == "ok" {
			return nil
		}
	}
	return nil // "not found" on both types: treat as already deleted
}

func (c *Cloudinary) URL(key string) string {
	ext := ""
	if i := strings.LastIndex(key, "."); i > strings.LastIndex(key, "/") {
		ext = key[i+1:]
	}
	rt := api.Image
	if ext != "" && ext != "jpg" && ext != "jpeg" && ext != "png" && ext != "webp" && ext != "heic" && ext != "pdf" {
		rt = api.File
	}
	asset, err := c.cld.Media(publicID(key))
	if err != nil {
		return ""
	}
	asset.AssetType = rt
	if ext != "" {
		asset.DeliveryType = "upload"
		asset.PublicID = publicID(key) + "." + ext
	}
	u, err := asset.String()
	if err != nil {
		return ""
	}
	return u
}
